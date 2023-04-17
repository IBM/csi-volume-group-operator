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

package vgccontroller

import (
	"context"
	"fmt"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	commonUtils "github.com/IBM/csi-volume-group-operator/controllers/common/utils"
	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VolumeGroupContentReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	DriverConfig *config.DriverConfig
	GRPCClient   *grpcClient.Client
	VGClient     grpcClient.VolumeGroup
	VGObjects    abstract.VolumeGroupObjects
}

func (r *VolumeGroupContentReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Name", req.Name, "Request.Namespace", req.Namespace)
	logger.Info(messages.ReconcileVG)

	vgc, err := utils.GetVGC(r.Client, logger, req.Name, req.Namespace, r.VGObjects.VGC)
	if err != nil {
		if errors.IsNotFound(err) {

			logger.Info("VolumeGroupContent resource not found")

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}

	vgClassName := vgc.GetVGCLassName()
	if vgClassName == "" {
		if err := utils.UpdateThinVGC(r.Client, logger, vgc.GetNamespace(), vgc.GetName(), vgc); err != nil {
			return ctrl.Result{}, err
		}
		if err := utils.UpdateVGCStatus(r.Client, logger, vgc, utils.GetCurrentTime(), false); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	vgClass, err := utils.GetVGClass(r.Client, logger, vgClassName, r.VGObjects.VGClass)
	if err != nil {
		return ctrl.Result{}, utils.HandleVGCErrorMessage(logger, r.Client, vgc, err, vgcReconcile)
	}

	if r.DriverConfig.DriverName != vgClass.GetDriver() {
		return ctrl.Result{}, nil
	}

	if err = vgClass.ValidatePrefixedParameters(); err != nil {
		logger.Error(err, "failed to validate parameters of volumegroupClass", "VGClassName", vgClass.GetName())
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
		if err = r.handleVGCWithDeletionTimestamp(logger, vgc, r.VGObjects.VG, secret); err != nil {
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

func (r *VolumeGroupContentReconciler) handleVGCWithDeletionTimestamp(logger logr.Logger, vgc abstract.VolumeGroupContent,
	vg abstract.VolumeGroup, secret map[string]string) error {
	if isVgExist, err := utils.IsVgExist(r.Client, logger, vgc, vg); err != nil {
		return err
	} else if isVgExist {
		return fmt.Errorf(messages.VgIsStillExist, vgc.GetName(), vgc.GetNamespace())
	}
	if commonUtils.Contains(vgc.GetFinalizers(), utils.VgcFinalizer) && !utils.IsContainOtherFinalizers(vgc, logger) {
		if r.DriverConfig.DisableDeletePvcs == "false" {
			if err := utils.DeletePVCsUnderVGC(logger, r.Client, r.VGObjects, r.DriverConfig.DriverName); err != nil {
				return err
			}
		}
		if err := r.removeVGC(logger, vgc, secret); err != nil {
			return err
		}
		logger.Info("VolumeGroupContent object is terminated, skipping reconciliation")
	}
	return nil
}

func (r *VolumeGroupContentReconciler) removeVGC(logger logr.Logger, vgc abstract.VolumeGroupContent, secret map[string]string) error {
	if vgc.GetDeletionPolicy() == common.VolumeGroupContentDelete {
		vgId := vgc.GetVGHandle()
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

func (r *VolumeGroupContentReconciler) handleCreateVG(logger logr.Logger, vgc abstract.VolumeGroupContent, vgClass abstract.VolumeGroupClass, secret map[string]string) error {
	parameters := vgClass.FilterPrefixedParameters()
	createVGResponse := r.createVG(vgc.GetName(), parameters, secret)
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

func (r *VolumeGroupContentReconciler) handleStaticProvisionedVGC(vgc abstract.VolumeGroupContent, logger logr.Logger) (error, bool) {
	if vgcSource := vgc.GetSource(); !vgcSource.IsNil() {
		if vgc.GetVGHandle() != "" {
			return r.updateStaticVGC(vgc, logger), true
		}
	}
	return nil, false
}

func (r *VolumeGroupContentReconciler) updateStaticVGC(vgc abstract.VolumeGroupContent, logger logr.Logger) error {
	if err := r.updateStaticVGCSpec(vgc, logger); err != nil {
		return err
	}
	if err := utils.UpdateVGCStatus(r.Client, logger, vgc, utils.GetCurrentTime(), true); err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupContentReconciler) updateStaticVGCSpec(vgc abstract.VolumeGroupContent, logger logr.Logger) error {
	vgClass, err := utils.GetVGClass(r.Client, logger, vgc.GetVGCLassName(), r.VGObjects.VGClass)
	if err != nil {
		return err
	}
	if err = utils.UpdateStaticVGC(r.Client, logger, vgc.GetNamespace(), vgc.GetName(), vgClass, vgc); err != nil {
		return err
	}
	return nil
}
