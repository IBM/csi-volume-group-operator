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

package persistentvolumeclaim

import (
	"context"
	"fmt"

	csiv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PersistentVolumeClaimReconciler struct {
	Client            client.Client
	Scheme            *runtime.Scheme
	Log               logr.Logger
	DriverConfig      *config.DriverConfig
	GRPCClient        *grpcClient.Client
	VolumeGroupClient grpcClient.VolumeGroup
}

func (r *PersistentVolumeClaimReconciler) Reconcile(_ context.Context, req reconcile.Request) (result reconcile.Result, err error) {
	result = reconcile.Result{}
	reqLogger := r.Log.WithValues(messages.RequestNamespace, req.Namespace, messages.RequestName, req.Name)
	reqLogger.Info(messages.ReconcilePersistentVolumeClaim)
	pvc, err := utils.GetPersistentVolumeClaim(reqLogger, r.Client, req.Name, req.Namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return result, nil
		}
		return result, err
	}

	isPVCNeedToBeHandled, err := r.isPVCNeedToBeHandled(reqLogger, pvc)
	if err != nil {
		return result, err
	}
	if !isPVCNeedToBeHandled {
		return result, nil
	}

	err = r.removePersistentVolumeClaimFromVolumeGroupObjects(reqLogger, pvc)
	if err != nil {
		return result, err
	}
	err = r.addPersistentVolumeClaimToVolumeGroupObjects(reqLogger, pvc)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *PersistentVolumeClaimReconciler) isPVCNeedToBeHandled(reqLogger logr.Logger, pvc *corev1.PersistentVolumeClaim) (bool, error) {
	isPVCHasMatchingDriver, err := utils.IsPVCHasMatchingDriver(reqLogger, r.Client, pvc, r.DriverConfig.DriverName)
	if err != nil {
		return false, err
	}
	if !isPVCHasMatchingDriver {
		return false, nil
	}
	if pvc.Status.Phase != corev1.ClaimBound {
		reqLogger.Info(messages.PersistentVolumeClaimIsNotInBoundPhase)
		return false, nil
	}
	isSCHasVGParam, err := utils.IsPVCInStaticVG(reqLogger, r.Client, pvc)
	if err != nil {
		return false, err
	}
	if isSCHasVGParam {
		storageClassName, sErr := utils.GetPersistentVolumeClaimClass(pvc)
		if sErr != nil {
			return false, sErr
		}
		msg := fmt.Sprintf(messages.StorageClassHasVGParameter, storageClassName, pvc.Namespace, pvc.Name)
		reqLogger.Info(msg)
		mErr := fmt.Errorf(msg)
		err = utils.HandlePVCErrorMessage(reqLogger, r.Client, pvc, mErr, addingPVC)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (r PersistentVolumeClaimReconciler) removePersistentVolumeClaimFromVolumeGroupObjects(
	logger logr.Logger, pvc *corev1.PersistentVolumeClaim) error {
	vgList, err := utils.GetVGList(logger, r.Client, r.DriverConfig.DriverName)
	if err != nil {
		return err
	}

	for _, vg := range vgList.Items {
		if !utils.IsPVCPartOfVG(pvc, vg.Status.PVCList) {
			continue
		}
		IsPVCMatchesVG, err := utils.IsPVCMatchesVG(logger, r.Client, pvc, vg)
		if err != nil {
			return utils.HandleErrorMessage(logger, r.Client, &vg, err, removingPVC)
		}

		if !IsPVCMatchesVG {
			err := utils.RemoveVolumeFromVolumeGroup(logger, r.Client, r.VolumeGroupClient,
				[]corev1.PersistentVolumeClaim{*pvc}, &vg)
			if err != nil {
				return utils.HandleErrorMessage(logger, r.Client, &vg, err, removingPVC)
			}
			err = utils.RemoveVolumeFromPvcListAndPvList(logger, r.Client, r.DriverConfig.DriverName, pvc, vg)
			return utils.HandleErrorMessage(logger, r.Client, &vg, err, removingPVC)
		}
	}
	return nil
}

func (r PersistentVolumeClaimReconciler) addPersistentVolumeClaimToVolumeGroupObjects(
	logger logr.Logger, pvc *corev1.PersistentVolumeClaim) error {
	var err error
	vgList, err := utils.GetVGList(logger, r.Client, r.DriverConfig.DriverName)
	if err != nil {
		return err
	}
	err = r.isPVCCanBeAddedToVG(logger, pvc, vgList)
	if err != nil {
		return err
	}

	for _, vg := range vgList.Items {
		if !utils.IsPVCPartOfVG(pvc, vg.Status.PVCList) {
			isPVCMatchesVG, err := utils.IsPVCMatchesVG(logger, r.Client, pvc, vg)
			if err != nil {
				return utils.HandleErrorMessage(logger, r.Client, &vg, err, addingPVC)
			}
			if isPVCMatchesVG {
				err := utils.AddVolumesToVolumeGroup(logger, r.Client, r.VolumeGroupClient,
					[]corev1.PersistentVolumeClaim{*pvc}, &vg)
				if err != nil {
					return utils.HandleErrorMessage(logger, r.Client, &vg, err, addingPVC)
				}
				err = utils.AddVolumeToPvcListAndPvList(logger, r.Client, pvc, &vg)
				return utils.HandleErrorMessage(logger, r.Client, &vg, err, addingPVC)
			}
		}

	}
	return nil
}

func (r PersistentVolumeClaimReconciler) isPVCCanBeAddedToVG(logger logr.Logger, pvc *corev1.PersistentVolumeClaim,
	vgList csiv1.VolumeGroupList) error {
	if r.DriverConfig.MultipleVGsToPVC == "true" {
		return nil
	}
	err := utils.IsPVCCanBeAddedToVG(logger, r.Client, pvc, vgList.Items)
	if hErr := utils.HandlePVCErrorMessage(logger, r.Client, pvc, err, addingPVC); hErr != nil {
		return hErr
	}
	return err
}

func (r *PersistentVolumeClaimReconciler) SetupWithManager(mgr ctrl.Manager, cfg *config.DriverConfig) error {
	r.VolumeGroupClient = grpcClient.NewVolumeGroupClient(r.GRPCClient.Client, cfg.RPCTimeout)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolumeClaim{}, builder.WithPredicates(pvcPredicate)).
		Complete(r)
}
