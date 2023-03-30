/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volumegroupcontent

import (
	"context"
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type VolumeGroupContentReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	DriverConfig *config.DriverConfig
	GRPCClient   *grpcClient.Client
	VGClient     grpcClient.VolumeGroup
}

func (r *VolumeGroupContentReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Name", req.Name, "Request.Namespace", req.Namespace)
	logger.Info(messages.ReconcileVG)

	vgc, err := utils.GetVGC(r.Client, logger, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {

			logger.Info("VolumeGroupContent resource not found")

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}

	vgClassName := utils.GetStringField(vgc.Spec, "VolumeGroupClassName")
	if vgClassName == "" {
		if err := utils.UpdateThinVGC(r.Client, vgc.Namespace, vgc.Name, logger); err != nil {
			return ctrl.Result{}, err
		}
		if err := utils.UpdateVGCStatus(r.Client, logger, vgc, utils.GetCurrentTime(), false); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	vgClass, err := utils.GetVGClass(r.Client, logger, vgClassName)
	if err != nil {
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}

	if r.DriverConfig.DriverName != vgClass.Driver {
		return ctrl.Result{}, nil
	}

	if err = utils.ValidatePrefixedParameters(vgClass.Parameters); err != nil {
		logger.Error(err, "failed to validate parameters of volumegroupClass", "VGClassName", vgClass.Name)
		if uErr := utils.UpdateVGCStatusError(r.Client, vgc, logger, err.Error()); uErr != nil {
			return ctrl.Result{}, uErr
		}
		return ctrl.Result{}, err
	}
	secret, err := utils.GetSecretDataFromClass(r.Client, vgClass, logger)
	if err != nil {
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}

	if vgc.GetDeletionTimestamp().IsZero() {
		if err = utils.AddFinalizerToVGC(r.Client, logger, vgc); err != nil {
			return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, createVGC)
		}
	} else {
		if err = r.handleVGCWithDeletionTimestamp(logger, vgc, secret); err != nil {
			return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, deleteVGC)
		}
		return ctrl.Result{}, nil
	}

	err, isStaticProvisioned := r.handleStaticProvisionedVGC(vgc, logger)
	if isStaticProvisioned {
		return ctrl.Result{}, err
	}

	if err = r.handleCreateVG(logger, vgc, vgClass, secret); err != nil {
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, createVGC)
	}

	if err = utils.CreateSuccessVGCEvent(logger, r.Client, vgc); err != nil {
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}
	return ctrl.Result{}, nil
}

func (r *VolumeGroupContentReconciler) handleVGCWithDeletionTimestamp(logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent, secret map[string]string) error {
	if isVgExist, err := utils.IsVgExist(r.Client, logger, vgc); err != nil {
		return err
	} else if isVgExist {
		return fmt.Errorf(messages.VgIsStillExist, vgc.Name, vgc.Namespace)
	}
	if utils.Contains(vgc.GetFinalizers(), utils.VgcFinalizer) {
		if r.DriverConfig.DisableDeletePvcs == "false" {
			if err := utils.DeletePVCsUnderVGC(logger, r.Client, vgc, r.DriverConfig.DriverName); err != nil {
				return err
			}
		}
		if err := r.removeVGC(logger, vgc, secret); err != nil {
			return err
		}
	}
	logger.Info("VolumeGroupContent object is terminated, skipping reconciliation")
	return nil
}

func (r *VolumeGroupContentReconciler) removeVGC(logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent, secret map[string]string) error {
	if *vgc.Spec.VolumeGroupDeletionPolicy == volumegroupv1.VolumeGroupContentDelete {
		vgId := vgc.Spec.Source.VolumeGroupHandle
		if err := r.deleteVG(logger, vgId, secret); err != nil {
			return err
		}
	}
	err := utils.RemoveFinalizerFromVGC(r.Client, logger, vgc)
	if err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupContentReconciler) deleteVG(logger logr.Logger, vgId string, secrets map[string]string) error {
	param := volumegroup.CommonRequestParameters{
		VolumeGroupID: vgId,
		Secrets:       secrets,
		VolumeGroup:   r.VGClient,
	}

	volumeGroupRequest := volumegroup.NewVolumeGroupRequest(param)

	resp := volumeGroupRequest.Delete()

	if resp.Error != nil {
		logger.Error(resp.Error, "failed to delete volume group")
		return resp.Error
	}

	return nil
}

func (r *VolumeGroupContentReconciler) handleCreateVG(logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent, vgClass *volumegroupv1.VolumeGroupClass, secret map[string]string) error {
	parameters := utils.FilterPrefixedParameters(utils.VGAsPrefix, vgClass.Parameters)
	createVGResponse := r.createVG(vgc.Name, parameters, secret)
	if createVGResponse.Error != nil {
		logger.Error(createVGResponse.Error, "failed to create volume group")
		return createVGResponse.Error
	}
	if err := utils.UpdateVGCByResponse(r.Client, vgc, createVGResponse); err != nil {
		return err
	}
	if err := utils.UpdateVGCStatus(r.Client, logger, vgc, utils.GetCurrentTime(), true); err != nil {
		return utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, updateStatusVGC)
	}
	return nil
}

func (r *VolumeGroupContentReconciler) createVG(vgName string, parameters, secrets map[string]string) *volumegroup.Response {
	param := volumegroup.CommonRequestParameters{
		Name:        vgName,
		Parameters:  parameters,
		Secrets:     secrets,
		VolumeGroup: r.VGClient,
	}

	volumeGroupRequest := volumegroup.NewVolumeGroupRequest(param)

	resp := volumeGroupRequest.Create()

	return resp
}

func (r *VolumeGroupContentReconciler) handleStaticProvisionedVGC(vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) (error, bool) {
	if vgcSpec := utils.GetObjectField(vgc.Spec, "Source"); !vgcSpec.IsNil() {
		if vgc.Spec.Source.VolumeGroupHandle != "" {
			return r.updateStaticVGC(vgc, logger), true
		}
	}
	return nil, false
}

func (r *VolumeGroupContentReconciler) updateStaticVGC(vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) error {
	if err := r.updateStaticVGCSpec(vgc, logger); err != nil {
		return err
	}
	if err := utils.UpdateVGCStatus(r.Client, logger, vgc, utils.GetCurrentTime(), true); err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupContentReconciler) updateStaticVGCSpec(vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) error {
	vgClass, err := utils.GetVGClass(r.Client, logger, *vgc.Spec.VolumeGroupClassName)
	if err != nil {
		return err
	}
	if err = utils.UpdateStaticVGC(r.Client, vgc.Namespace, vgc.Name, vgClass, logger); err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupContentReconciler) SetupWithManager(mgr ctrl.Manager, cfg *config.DriverConfig) error {
	r.VGClient = grpcClient.NewVolumeGroupClient(r.GRPCClient.Client, cfg.RPCTimeout)

	pred := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&volumegroupv1.VolumeGroupContent{}, builder.WithPredicates(pred)).
		Complete(r)
}
