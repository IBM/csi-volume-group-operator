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

package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetVG(client client.Client, logger logr.Logger, vgName string, vgNamespace string) (*volumegroupv1.VolumeGroup, error) {
	logger.Info(fmt.Sprintf(messages.GetVG, vgName, vgNamespace))
	vg := &volumegroupv1.VolumeGroup{}
	namespacedVG := types.NamespacedName{Name: vgName, Namespace: vgNamespace}
	err := client.Get(context.TODO(), namespacedVG, vg)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "VolumeGroup not found", "VolumeGroup Name", vgName)
		}
		return nil, err
	}
	return vg, nil
}

func IsVgExist(client client.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent) (bool, error) {
	if !vgc.GetVGRef().IsNil() {
		if vg, err := GetVG(client, logger, vgc.GetVGRefName(), vgc.GetVGRefNamespace()); err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
		} else {
			return vg != nil, nil
		}
	}
	return false, nil
}

func UpdateVGSourceContent(client client.Client, instance abstract.VolumeGroup,
	vgcName string, logger logr.Logger) error {
	instance.UpdateVGCName(vgcName)
	if err := UpdateObject(client, instance); err != nil {
		logger.Error(err, "failed to update source", "VGName", instance.GetName())
		return err
	}
	return nil
}

func updateVGStatus(client client.Client, vg abstract.VolumeGroup, logger logr.Logger) error {
	logger.Info(fmt.Sprintf(messages.UpdateVGStatus, vg.GetNamespace(), vg.GetName()))
	if err := UpdateObjectStatus(client, vg); err != nil {
		if apierrors.IsConflict(err) {
			return err
		}
		logger.Error(err, "failed to update volumeGroup status", "VGName", vg.GetName())
		return err
	}
	return nil
}

func UpdateVGStatus(client client.Client, vg abstract.VolumeGroup, vgcName string,
	groupCreationTime *metav1.Time, ready bool, logger logr.Logger) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.UpdateBoundVGCName(vgcName)
		vg.UpdateGroupCreationTime(groupCreationTime)
		vg.UpdateReady(ready)
		vg.UpdateError(nil)
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return updateVGStatus(client, vg, logger)
}

func updateVGStatusPVCList(client client.Client, vg abstract.VolumeGroup, logger logr.Logger,
	pvcList []corev1.PersistentVolumeClaim) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.UpdatePVCList(pvcList)
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateVGStatusError(client client.Client, vg abstract.VolumeGroup, logger logr.Logger, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.UpdateError(&common.VolumeGroupError{Message: &message})
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func vgRetryOnConflictFunc(client client.Client, vg abstract.VolumeGroup, logger logr.Logger) error {
	err := updateVGStatus(client, vg, logger)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, vg)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateVGStatus, vg.GetNamespace(), vg.GetName()))
	}
	return err
}

func GetVGList(logger logr.Logger, client client.Client, driver string) (volumegroupv1.VolumeGroupList, error) {
	logger.Info(messages.ListVGs)
	vg := &volumegroupv1.VolumeGroupList{}
	err := client.List(context.TODO(), vg)
	if err != nil {
		return volumegroupv1.VolumeGroupList{}, err
	}
	vgList, err := getProvisionedVGs(logger, client, vg, driver)
	if err != nil {
		return volumegroupv1.VolumeGroupList{}, err
	}
	return vgList, nil
}

func getProvisionedVGs(logger logr.Logger, client client.Client, vgList *volumegroupv1.VolumeGroupList,
	driver string) (volumegroupv1.VolumeGroupList, error) {
	newVgList := volumegroupv1.VolumeGroupList{}
	for _, vg := range vgList.Items {
		isVGHasMatchingDriver, err := isVGHasMatchingDriver(logger, client, vg, driver)
		if err != nil {
			return volumegroupv1.VolumeGroupList{}, err
		}
		if isVGHasMatchingDriver {
			newVgList.Items = append(newVgList.Items, vg)
		}
	}
	return newVgList, nil
}

func isVGHasMatchingDriver(logger logr.Logger, client client.Client, vg volumegroupv1.VolumeGroup,
	driver string) (bool, error) {
	vgClassDriver, err := getVGClassDriver(client, logger, vg.GetVGCLassName())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return vgClassDriver == driver, nil
}
func IsPVCMatchesVG(logger logr.Logger, client client.Client,
	pvc *corev1.PersistentVolumeClaim, vg abstract.VolumeGroup) (bool, error) {

	logger.Info(fmt.Sprintf(messages.CheckIfPVCMatchesVG,
		pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
	areLabelsMatchLabelSelector, err := areLabelsMatchLabelSelector(
		client, pvc.ObjectMeta.Labels, *vg.GetSelector())

	if areLabelsMatchLabelSelector {
		logger.Info(fmt.Sprintf(messages.PVCMatchedToVG,
			pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
		return true, err
	} else {
		logger.Info(fmt.Sprintf(messages.PVCNotMatchedToVG,
			pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
		return false, err
	}
}

func RemovePVCFromVG(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim, vg abstract.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.RemovePVCFromVG, pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
	vg.UpdatePVCList(removeFromPVCList(pvc, vg.GetPVCList()))
	err := updateVGStatusPVCList(client, vg, logger, vg.GetPVCList())
	if err != nil {
		vg.UpdatePVCList(appendPVC(vg.GetPVCList(), *pvc))
		logger.Error(err, fmt.Sprintf(messages.FailedToRemovePVCFromVG,
			pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedPVCFromVG, pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
	return nil
}

func removeMultiplePVCs(pvcList []corev1.PersistentVolumeClaim,
	pvcs []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	for _, pvc := range pvcs {
		pvcList = removeFromPVCList(&pvc, pvcList)
	}
	return pvcList
}

func removeFromPVCList(pvc *corev1.PersistentVolumeClaim, pvcList []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	for index, pvcFromList := range pvcList {
		if pvcFromList.Name == pvc.Name && pvcFromList.Namespace == pvc.Namespace {
			pvcList = removeByIndexFromPVCList(pvcList, index)
			return pvcList
		}
	}
	return pvcList
}

func getVgId(logger logr.Logger, client client.Client, vg abstract.VolumeGroup) (string, error) {
	vgc, err := GetVGC(client, logger, vg.GetVGCName(), vg.GetNamespace())
	if err != nil {
		return "", err
	}
	return vgc.GetVGHandle(), nil
}

func AddPVCToVG(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim, vg abstract.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.AddPVCToVG, pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
	vg.UpdatePVCList(appendPVC(vg.GetPVCList(), *pvc))
	err := updateVGStatusPVCList(client, vg, logger, vg.GetPVCList())
	if err != nil {
		vg.UpdatePVCList(removeFromPVCList(pvc, vg.GetPVCList()))
		logger.Error(err, fmt.Sprintf(messages.FailedToAddPVCToVG,
			pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedPVCToVG, pvc.Namespace, pvc.Name, vg.GetNamespace(), vg.GetName()))
	return nil
}

func appendMultiplePVCs(pvcListInVG []corev1.PersistentVolumeClaim,
	pvcs []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	for _, pvc := range pvcs {
		pvcListInVG = appendPVC(pvcListInVG, pvc)
	}
	return pvcListInVG
}

func appendPVC(pvcListInVG []corev1.PersistentVolumeClaim, pvc corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	for _, pvcFromList := range pvcListInVG {
		if pvcFromList.Name == pvc.Name && pvcFromList.Namespace == pvc.Namespace {
			return pvcListInVG
		}
	}
	pvcListInVG = append(pvcListInVG, pvc)
	return pvcListInVG
}

func IsPVCPartAnyVG(pvc *corev1.PersistentVolumeClaim, vgs []volumegroupv1.VolumeGroup) bool {
	for _, vg := range vgs {
		if IsPVCInPVCList(pvc, vg.GetPVCList()) {
			return true
		}
	}
	return false
}

func IsPVCInPVCList(pvc *corev1.PersistentVolumeClaim, pvcList []corev1.PersistentVolumeClaim) bool {
	for _, pvcFromList := range pvcList {
		if pvcFromList.Name == pvc.Name && pvcFromList.Namespace == pvc.Namespace {
			return true
		}
	}
	return false
}
