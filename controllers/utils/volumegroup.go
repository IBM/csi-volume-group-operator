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

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateVolumeGroupSourceContent(client client.Client, instance *volumegroupv1.VolumeGroup,
	vgcName string, logger logr.Logger) error {
	instance.Spec.Source.VolumeGroupContentName = &vgcName
	if err := UpdateObject(client, instance); err != nil {
		logger.Error(err, "failed to update source", "VGName", instance.Name)
		return err
	}
	return nil
}

func updateVolumeGroupStatus(client client.Client, instance *volumegroupv1.VolumeGroup, logger logr.Logger) error {
	logger.Info(fmt.Sprintf(messages.UpdateVolumeGroupStatus, instance.Namespace, instance.Name))
	if err := UpdateObjectStatus(client, instance); err != nil {
		if apierrors.IsConflict(err) {
			return err
		}
		logger.Error(err, "failed to update volumeGroup status", "VGName", instance.Name)
		return err
	}
	return nil
}

func UpdateVolumeGroupStatus(client client.Client, vg *volumegroupv1.VolumeGroup, vgc *volumegroupv1.VolumeGroupContent,
	groupCreationTime *metav1.Time, ready bool, logger logr.Logger) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.Status.BoundVolumeGroupContentName = &vgc.Name
		vg.Status.GroupCreationTime = groupCreationTime
		vg.Status.Ready = &ready
		vg.Status.Error = nil
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return updateVolumeGroupStatus(client, vg, logger)
}

func updateVolumeGroupStatusPVCList(client client.Client, vg *volumegroupv1.VolumeGroup, logger logr.Logger,
	pvcList []corev1.PersistentVolumeClaim) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.Status.PVCList = pvcList
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateVolumeGroupStatusError(client client.Client, vg *volumegroupv1.VolumeGroup, logger logr.Logger, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vg.Status.Error = &volumegroupv1.VolumeGroupError{Message: &message}
		err := vgRetryOnConflictFunc(client, vg, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func vgRetryOnConflictFunc(client client.Client, vg *volumegroupv1.VolumeGroup, logger logr.Logger) error {
	err := updateVolumeGroupStatus(client, vg, logger)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, vg)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateVolumeGroupStatus, vg.Namespace, vg.Name))
	}
	return err
}

func GetVGList(logger logr.Logger, client client.Client, driver string) (volumegroupv1.VolumeGroupList, error) {
	logger.Info(messages.ListVolumeGroups)
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
	vgClassDriver, err := getVGClassDriver(client, logger, *vg.Spec.VolumeGroupClassName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return vgClassDriver == driver, nil
}
func IsPVCMatchesVG(logger logr.Logger, client client.Client,
	pvc *corev1.PersistentVolumeClaim, vg volumegroupv1.VolumeGroup) (bool, error) {

	logger.Info(fmt.Sprintf(messages.CheckIfPersistentVolumeClaimMatchesVolumeGroup,
		pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
	areLabelsMatchLabelSelector, err := areLabelsMatchLabelSelector(
		client, pvc.ObjectMeta.Labels, *vg.Spec.Source.Selector)

	if areLabelsMatchLabelSelector {
		logger.Info(fmt.Sprintf(messages.PersistentVolumeClaimMatchedToVolumeGroup,
			pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
		return true, err
	} else {
		logger.Info(fmt.Sprintf(messages.PersistentVolumeClaimNotMatchedToVolumeGroup,
			pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
		return false, err
	}
}

func RemovePVCFromVG(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.RemovePersistentVolumeClaimFromVolumeGroup,
		pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
	vg.Status.PVCList = removeFromPVCList(pvc, vg.Status.PVCList)
	err := updateVolumeGroupStatusPVCList(client, vg, logger, vg.Status.PVCList)
	if err != nil {
		vg.Status.PVCList = appendPVC(vg.Status.PVCList, *pvc)
		logger.Error(err, fmt.Sprintf(messages.FailedToRemovePersistentVolumeClaimFromVolumeGroup,
			pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedPersistentVolumeClaimFromVolumeGroup,
		pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
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

func getVgId(logger logr.Logger, client client.Client, vg *volumegroupv1.VolumeGroup) (string, error) {
	vgc, err := GetVolumeGroupContent(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return "", err
	}
	return string(vgc.Spec.Source.VolumeGroupHandle), nil
}

func AddPVCToVG(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.AddPersistentVolumeClaimToVolumeGroup,
		pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
	vg.Status.PVCList = appendPVC(vg.Status.PVCList, *pvc)
	err := updateVolumeGroupStatusPVCList(client, vg, logger, vg.Status.PVCList)
	if err != nil {
		vg.Status.PVCList = removeFromPVCList(pvc, vg.Status.PVCList)
		logger.Error(err, fmt.Sprintf(messages.FailedToAddPersistentVolumeClaimToVolumeGroup,
			pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedPersistentVolumeClaimToVolumeGroup,
		pvc.Namespace, pvc.Name, vg.Namespace, vg.Name))
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
		if IsPVCPartOfVG(pvc, vg.Status.PVCList) {
			return true
		}
	}
	return false
}

func IsPVCPartOfVG(pvc *corev1.PersistentVolumeClaim, pvcList []corev1.PersistentVolumeClaim) bool {
	for _, pvcFromList := range pvcList {
		if pvcFromList.Name == pvc.Name && pvcFromList.Namespace == pvc.Namespace {
			return true
		}
	}
	return false
}
