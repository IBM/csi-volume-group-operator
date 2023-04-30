/*
Copyright 2023.

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

package vgcontroller

import (
	"context"
	"fmt"

	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	commonUtils "github.com/IBM/csi-volume-group-operator/controllers/common/utils"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VolumeGroupReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	DriverConfig *config.DriverConfig
	GRPCClient   *grpcClient.Client
	VGClient     grpcClient.VolumeGroup
	VGObjects    abstract.VolumeGroupObjects
}

func (r *VolumeGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Name", req.Name, "Request.Namespace", req.Namespace)
	logger.Info(messages.ReconcileVG)

	instance := r.VGObjects.VG
	if err := r.Client.Get(context.TODO(), req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("VolumeGroup resource not found")

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, vgReconcile)
	}

	vgClass, err := utils.GetVGClass(r.Client, logger, instance.GetVGCLassName(), r.VGObjects.VGClass)
	if err != nil {
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, vgReconcile)
	}

	if r.DriverConfig.DriverName != vgClass.GetDriver() {
		return ctrl.Result{}, nil
	}

	if err = utils.ValidatePrefixedParameters(vgClass.GetParameters()); err != nil {
		logger.Error(err, "failed to validate parameters of volumegroupClass", "VGClassName", vgClass.GetName())
		if uErr := utils.UpdateVGStatusError(r.Client, instance, logger, err.Error()); uErr != nil {
			return ctrl.Result{}, uErr
		}
		return ctrl.Result{}, err
	}

	if instance.GetDeletionTimestamp().IsZero() {
		if err = utils.AddFinalizerToVG(r.Client, logger, instance); err != nil {
			return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, createVG)
		}

	} else {
		if commonUtils.Contains(instance.GetFinalizers(), utils.VGFinalizer) {
			if err = r.removeInstance(logger, instance); err != nil {
				return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, deleteVG)
			}
		}
		logger.Info("volumeGroup object is terminated, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	groupCreationTime := utils.GetCurrentTime()

	err, isStaticProvisioned := r.handleStaticProvisionedVG(instance, logger, groupCreationTime, vgClass)
	if isStaticProvisioned {
		return ctrl.Result{}, err
	}

	vgName, err := utils.MakeVGName(utils.VGNamePrefix, string(instance.GetUID()))
	if err != nil {
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, createVG)
	}
	secretName, secretNamespace := utils.GetSecretCred(vgClass)
	r.VGObjects.VGC.GenerateVGC(vgName, instance, vgClass, secretName, secretNamespace)
	vgc := r.VGObjects.VGC
	logger.Info("GenerateVolumeGroupContent", "vgc", vgc)
	if err = utils.CreateVGC(r.Client, logger, vgc); err != nil {
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, createVGC)
	}
	if isVGCReady, err := r.isVGCReady(logger, vgc); err != nil {
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, createVGC)
	} else if !isVGCReady {
		return ctrl.Result{Requeue: true}, nil
	}

	err = r.updateItems(instance, logger, groupCreationTime, vgc.GetName())
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.updatePVCs(logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.createSuccessVGEvent(logger, instance)
	if err != nil {
		return ctrl.Result{}, utils.HandleErrorMessage(logger, r.Client, instance, err, vgReconcile)
	}
	return ctrl.Result{}, nil
}

func (r *VolumeGroupReconciler) updatePVCs(logger logr.Logger) error {
	vg := r.VGObjects.VG

	matchingPvcs, err := r.getMatchingPVCs(logger, vg)
	if err != nil {
		return utils.HandleErrorMessage(logger, r.Client, vg, err, vgReconcile)
	}
	if utils.IsPVCListEqual(matchingPvcs, vg.GetPVCList()) {
		return nil
	}
	err = utils.ModifyVolumesInVG(logger, r.Client, r.VGClient, matchingPvcs, vg, r.VGObjects.VGClass)
	if err != nil {
		return utils.HandleErrorMessage(logger, r.Client, vg, err, vgReconcile)
	}
	err = utils.UpdatePvcAndPvList(logger, r.VGObjects, r.Client, r.DriverConfig.DriverName, matchingPvcs)
	if err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupReconciler) handleStaticProvisionedVG(vg abstract.VolumeGroup, logger logr.Logger, groupCreationTime *metav1.Time, vgClass abstract.VolumeGroupClass) (error, bool) {
	if vg.GetVGCName() != "" {
		err := r.updateItems(vg, logger, groupCreationTime, vg.GetVGCName())
		if err != nil {
			return err, true
		}
		err = utils.UpdateStaticVGCFromVG(r.Client, vg, vgClass, logger)
		if err != nil {
			return err, true
		}
		err = r.updatePVCs(logger)
		if err != nil {
			return err, true
		}
		return nil, true
	}
	return nil, false
}

func (r *VolumeGroupReconciler) updateItems(instance abstract.VolumeGroup, logger logr.Logger, groupCreationTime *metav1.Time, vgcName string) error {
	if err := utils.UpdateVGSourceContent(r.Client, instance, vgcName, logger); err != nil {
		return utils.HandleErrorMessage(logger, r.Client, instance, err, updateVGC)
	}
	if err := utils.UpdateVGStatus(r.Client, instance, vgcName, groupCreationTime, true, logger); err != nil {
		return utils.HandleErrorMessage(logger, r.Client, instance, err, updateStatusVG)
	}
	return nil
}

func (r *VolumeGroupReconciler) removeInstance(logger logr.Logger, instance abstract.VolumeGroup) error {
	vgc, err := utils.GetVGC(r.Client, logger, instance.GetVGCName(), instance.GetNamespace())
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

	} else {
		err = r.removeVGCObject(logger, vgc)
		if err != nil {
			return err
		}
	}
	if err = utils.RemoveFinalizerFromVG(r.Client, logger, instance); err != nil {
		return err
	}
	return nil
}

func (r *VolumeGroupReconciler) removeVGCObject(logger logr.Logger, vgc abstract.VolumeGroupContent) error {
	if vgc.GetDeletionPolicy() == common.VolumeGroupContentDelete {
		if err := r.Client.Delete(context.TODO(), vgc); err != nil {
			logger.Error(err, "Failed to delete volume group content", "VGCName", vgc.GetName())
			return err
		}
	}
	return nil
}

func (r *VolumeGroupReconciler) isPVCShouldBeRemovedFromVg(logger logr.Logger, vg abstract.VolumeGroup,
	pvc *corev1.PersistentVolumeClaim) (bool, error) {
	if !utils.IsPVCInPVCList(pvc, vg.GetPVCList()) {
		return false, nil
	}

	isPVCMatchesVG, err := utils.IsPVCMatchesVG(logger, r.Client, pvc, vg)
	if err != nil {
		return false, err
	}
	return !isPVCMatchesVG, nil
}

func (r *VolumeGroupReconciler) isPVCShouldBeInVg(logger logr.Logger, vg abstract.VolumeGroup,
	pvc *corev1.PersistentVolumeClaim, vgList abstract.VolumeGroupList, vgClass abstract.VolumeGroupClass) (bool, error) {

	isPVCMatchesVG, err := utils.IsPVCMatchesVG(logger, r.Client, pvc, vg)
	if err != nil {
		return false, err
	}
	if !isPVCMatchesVG {
		return false, nil
	}

	if err := r.isPVCCanBeAddedToVG(logger, pvc, vgList, vgClass); err != nil {
		return false, err
	}
	return true, nil
}

func (r VolumeGroupReconciler) isPVCCanBeAddedToVG(logger logr.Logger, pvc *corev1.PersistentVolumeClaim,
	vgList abstract.VolumeGroupList, vgClass abstract.VolumeGroupClass) error {
	if r.DriverConfig.MultipleVGsToPVC == "true" {
		return nil
	}

	vgs, err := utils.GetVGs(logger, r.Client, r.DriverConfig.DriverName, vgList, vgClass)
	if err != nil {
		return err
	}
	err = utils.IsPVCCanBeAddedToVG(logger, r.Client, pvc, vgs)
	return err
}

func (r VolumeGroupReconciler) createSuccessVGEvent(logger logr.Logger, vg abstract.VolumeGroup) error {
	message := fmt.Sprintf(messages.VGCreated, vg.GetNamespace(), vg.GetName())
	err := utils.HandleSuccessMessage(logger, r.Client, vg, message, vgReconcile)
	if err != nil {
		return nil
	}
	return nil
}

func (r *VolumeGroupReconciler) getMatchingPVCs(logger logr.Logger, vg abstract.VolumeGroup) ([]corev1.PersistentVolumeClaim, error) {
	var matchingPvcs []corev1.PersistentVolumeClaim
	pvcList, err := utils.GetPVCList(logger, r.Client, r.DriverConfig.DriverName)
	if err != nil {
		return nil, err
	}
	for _, pvc := range pvcList.Items {
		isPVCShouldBeInVg, err := r.isPVCShouldBeInVg(logger, vg, &pvc, r.VGObjects.VGList, r.VGObjects.VGClass)
		if err != nil {
			return nil, err
		}
		isPVCShouldBeHandled, err := utils.IsPVCNeedToBeHandled(logger, &pvc, r.Client, r.DriverConfig.DriverName)
		if err != nil {
			return nil, err
		}
		if isPVCShouldBeInVg && !utils.IsPVCInPVCList(&pvc, matchingPvcs) && isPVCShouldBeHandled {
			matchingPvcs = append(matchingPvcs, pvc)
		}
	}
	return matchingPvcs, err
}

func (r *VolumeGroupReconciler) isVGCReady(logger logr.Logger, vgc abstract.VolumeGroupContent) (bool, error) {
	vgcFromCluster, err := utils.GetVGC(r.Client, logger, vgc.GetName(), vgc.GetNamespace())
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}
	return vgcFromCluster.IsReady(), nil
}
