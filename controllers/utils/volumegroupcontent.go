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

package utils

import (
	"context"
	"fmt"

	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	pkg_utils "github.com/IBM/csi-volume-group-operator/pkg/utils"
	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddMatchingPVToMatchingVGC(logger logr.Logger, client client.Client,
	pvc *corev1.PersistentVolumeClaim, vgObjects abstract.VolumeGroupObjects) error {
	vg := vgObjects.VG

	pv, err := GetPVFromPVC(logger, client, pvc)
	if err != nil {
		return err
	}
	vgc, err := GetVGC(client, logger, vg.GetVGCName(), vg.GetNamespace(), vgObjects.VGC)
	if err != nil {
		return err
	}

	if pv != nil {
		return addPVToVGC(logger, client, pv, vgc)
	}
	return nil
}

func GetVGC(client client.Client, logger logr.Logger, vgcName string, vgcNamespace string,
	vgc abstract.VolumeGroupContent) (abstract.VolumeGroupContent, error) {
	logger.Info(fmt.Sprintf(messages.GetVGC, vgcName, vgcNamespace))
	namespacedVGC := types.NamespacedName{Name: vgcName, Namespace: vgcNamespace}
	err := client.Get(context.TODO(), namespacedVGC, vgc)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "VolumeGroupContent not found", "VolumeGroupContent Name", vgcName)
		}
		return nil, err
	}

	return vgc, nil
}

func CreateVGC(client client.Client, logger logr.Logger, vgc abstract.VolumeGroupContent) error {
	err := client.Create(context.TODO(), vgc)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("VolumeGroupContent is already exists")
			return nil
		}
		logger.Error(err, "VolumeGroupContent creation failed")
		return err
	}
	return err
}

func CreateSuccessVGCEvent(logger logr.Logger, client client.Client, vgc abstract.VolumeGroupContent) error {
	vgc.UpdateAPIVersion(APIVersion)
	vgc.UpdateKind(vgcKind)
	message := fmt.Sprintf(messages.VGCCreated, vgc.GetNamespace(), vgc.GetName())
	err := createSuccessNamespacedObjectEvent(logger, client, vgc, message, createVGC)
	if err != nil {
		return nil
	}
	return nil
}

func UpdateVGCStatus(client client.Client, logger logr.Logger, vgc abstract.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) error {
	updateVGCStatusFields(vgc, groupCreationTime, ready)
	if err := UpdateObjectStatus(client, vgc); err != nil {
		logger.Error(err, "failed to update status")
		return err
	}
	return nil
}

func updateVGCStatusFields(vgc abstract.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) {
	vgc.UpdateGroupCreationTime(groupCreationTime)
	vgc.UpdateReady(ready)
}

func RemovePVFromVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume, vgc abstract.VolumeGroupContent) error {
	logger.Info(fmt.Sprintf(messages.RemovePVFromVGC, pv.Name, vgc.GetNamespace(), vgc.GetName()))
	vgc.UpdatePVList(removeFromPVList(pv, vgc.GetPVList()))
	err := updateVGCStatusPVList(client, vgc, logger, vgc.GetPVList())
	if err != nil {
		vgc.UpdatePVList(appendPV(vgc.GetPVList(), *pv))
		logger.Error(err, fmt.Sprintf(messages.FailedToRemovePVFromVGC, pv.Name, vgc.GetNamespace(), vgc.GetName()))
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedPVFromVGC, pv.Name, vgc.GetNamespace(), vgc.GetName()))
	return nil
}

func removeFromPVList(pv *corev1.PersistentVolume, pvList []corev1.PersistentVolume) []corev1.PersistentVolume {
	for index, pvFromList := range pvList {
		if pvFromList.Name == pv.Name && pvFromList.Namespace == pv.Namespace {
			pvList = removeByIndexFromPVList(pvList, index)
			return pvList
		}
	}
	return pvList
}

func addPVToVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume,
	vgc abstract.VolumeGroupContent) error {
	logger.Info(fmt.Sprintf(messages.AddPVToVG, pv.Name, vgc.GetNamespace(), vgc.GetName()))
	vgc.UpdatePVList(appendPV(vgc.GetPVList(), *pv))
	err := updateVGCStatusPVList(client, vgc, logger, vgc.GetPVList())
	if err != nil {
		vgc.UpdatePVList(removeFromPVList(pv, vgc.GetPVList()))
		logger.Error(err, fmt.Sprintf(messages.FailedToAddPVToVGC, pv.Name, vgc.GetNamespace(), vgc.GetName()))
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedPVToVGC, pv.Name, vgc.GetNamespace(), vgc.GetName()))
	return nil
}

func appendPV(pvListInVGC []corev1.PersistentVolume, pv corev1.PersistentVolume) []corev1.PersistentVolume {
	for _, pvFromList := range pvListInVGC {
		if pvFromList.Name == pv.Name {
			return pvListInVGC
		}
	}
	pvListInVGC = append(pvListInVGC, pv)
	return pvListInVGC
}

func updateVGCStatusPVList(client client.Client, vgc abstract.VolumeGroupContent, logger logr.Logger,
	pvList []corev1.PersistentVolume) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vgc.UpdatePVList(pvList)
		err := vgcRetryOnConflictFunc(client, vgc, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateVGCStatusError(client client.Client, vgc abstract.VolumeGroupContent, logger logr.Logger, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vgc.UpdateError(&common.VolumeGroupError{Message: &message})
		err := vgcRetryOnConflictFunc(client, vgc, logger)
		return err
	})
	return err
}

func vgcRetryOnConflictFunc(client client.Client, vgc abstract.VolumeGroupContent, logger logr.Logger) error {
	err := UpdateObjectStatus(client, vgc)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, vgc)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateVGCtStatus, vgc.GetNamespace(), vgc.GetName()))
	}
	return err
}

func UpdateStaticVGCFromVG(client client.Client, logger logr.Logger, vgObjects abstract.VolumeGroupObjects) error {
	vg := vgObjects.VG

	vgc, err := GetVGC(client, logger, vg.GetVGCName(), vg.GetNamespace(), vgObjects.VGC)
	if err != nil {
		return err
	}
	vgc.UpdateVGRef(pkg_utils.GenerateObjectReference(vg))
	updateStaticVGCSpec(vgObjects.VGClass, vgc)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func UpdateStaticVGC(client client.Client, logger logr.Logger, vgcNamespace, vgcName string,
	vgClass abstract.VolumeGroupClass, vgc abstract.VolumeGroupContent) error {
	vgc, err := GetVGC(client, logger, vgcName, vgcNamespace, vgc)
	if err != nil {
		return err
	}
	updateStaticVGCSpec(vgClass, vgc)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func updateStaticVGCSpec(vgClass abstract.VolumeGroupClass, vgc abstract.VolumeGroupContent) {
	secretName, secretNamespace := vgClass.GetSecretCred()
	if vgc.GetVGCLassName() == "" {
		vgClassName := vgClass.GetName()
		vgc.UpdateVGClassName(vgClassName)
	}
	if vgc.GetVGSecretRef() == nil {
		vgc.UpdateSecretRef(pkg_utils.GenerateSecretReference(secretName, secretNamespace))
	}
	if vgc.GetDeletionPolicy() == "" {
		vgc.UpdateDeletionPolicy(pkg_utils.GetVolumeGroupDeletionPolicy(vgClass))
	}
}

func UpdateThinVGC(client client.Client, logger logr.Logger, vgcNamespace, vgcName string,
	vgc abstract.VolumeGroupContent) error {
	vgcFromCluster, err := GetVGC(client, logger, vgcName, vgcNamespace, vgc)
	if err != nil {
		return err
	}
	updateThinVGCSpec(vgcFromCluster)
	if err = UpdateObject(client, vgcFromCluster); err != nil {
		return err
	}
	return nil
}

func updateThinVGCSpec(vgc abstract.VolumeGroupContent) {
	if vgc.GetDeletionPolicy() == "" {
		defaultDeletionPolicy := common.VolumeGroupContentRetain
		vgc.UpdateDeletionPolicy(&defaultDeletionPolicy)
	}
}

func UpdateVGCByResponse(client client.Client, vgc abstract.VolumeGroupContent, resp *volumegroup.Response) error {
	CreateVGResponse := resp.Response.(*csi.CreateVolumeGroupResponse)
	vgc.UpdateVGHandle(CreateVGResponse.VolumeGroup.VolumeGroupId)
	vgc.UpdateVGAttributes(CreateVGResponse.VolumeGroup.VolumeGroupContext)
	if err := UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func DeletePVCsUnderVGC(logger logr.Logger, client client.Client, VGObjects abstract.VolumeGroupObjects, driver string) error {
	vgc := VGObjects.VGC

	logger.Info(fmt.Sprintf(messages.DeletePVCsUnderVGC, vgc.GetNamespace(), vgc.GetName()))
	for _, pv := range vgc.GetPVList() {
		pvcName := getPVCNameFromPV(pv)
		pvcNamespace := getPVCNamespaceFromPV(pv)
		if pvcNamespace == "" || pvcName == "" {
			pvc, err := getMatchingPVCFromPVCListToPV(logger, client, pv.Name, driver)
			if err != nil {
				return err
			}
			if pvc.Name == "" || pvc.Namespace == "" {
				logger.Info(fmt.Sprintf(messages.CannotFindMatchingPVCForPV, pv.Name))
				continue
			}
			pvcName = pvc.Name
			pvcNamespace = pvc.Namespace
		}
		err := deletePVC(logger, client, pvcName, pvcNamespace, driver, VGObjects.VGList, VGObjects.VGClass)
		if err != nil {
			return err
		}
		err = RemovePVFromVGC(logger, client, &pv, vgc)
		if err != nil {
			return err
		}
	}
	return nil
}
