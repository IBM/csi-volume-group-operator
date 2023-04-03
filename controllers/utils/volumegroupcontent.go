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
	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
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
	pvc *corev1.PersistentVolumeClaim, vg abstract.VolumeGroup) error {
	pv, err := GetPVFromPVC(logger, client, pvc)
	if err != nil {
		return err
	}
	vgc, err := GetVGC(client, logger, vg.GetVGCName(), vg.GetNamespace())
	if err != nil {
		return err
	}

	if pv != nil {
		return addPVToVGC(logger, client, pv, vgc)
	}
	return nil
}

func GetVGC(client client.Client, logger logr.Logger, vgcName string, vgcNamespace string) (*volumegroupv1.VolumeGroupContent, error) {
	logger.Info(fmt.Sprintf(messages.GetVGC, vgcName, vgcNamespace))
	vgc := &volumegroupv1.VolumeGroupContent{}
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

func CreateVGC(client client.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent) error {
	err := client.Create(context.TODO(), vgc)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("VolumeGroupContent is already exists")
			return nil
		}
		logger.Error(err, "VolumeGroupContent creation failed", "VolumeGroupContent Name")
		return err
	}
	return err
}

func CreateSuccessVGCEvent(logger logr.Logger, client client.Client, vgc *volumegroupv1.VolumeGroupContent) error {
	vgc.UpdateAPIVersion(APIVersion)
	vgc.UpdateKind(vgcKind)
	message := fmt.Sprintf(messages.VGCCreated, vgc.GetNamespace(), vgc.GetName())
	err := createSuccessNamespacedObjectEvent(logger, client, vgc, message, createVGC)
	if err != nil {
		return nil
	}
	return nil
}

func UpdateVGCStatus(client client.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) error {
	updateVGCStatusFields(vgc, groupCreationTime, ready)
	if err := UpdateObjectStatus(client, vgc); err != nil {
		logger.Error(err, "failed to update status")
		return err
	}
	return nil
}

func updateVGCStatusFields(vgc *volumegroupv1.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) {
	vgc.UpdateGroupCreationTime(groupCreationTime)
	vgc.UpdateReady(ready)
}

func GenerateVGC(vgname string, instance abstract.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass, secretName string, secretNamespace string) *volumegroupv1.VolumeGroupContent {
	return &volumegroupv1.VolumeGroupContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vgname,
			Namespace: instance.GetNamespace(),
		},
		Spec: generateVGCSpec(instance, vgClass, secretName, secretNamespace),
	}
}

func generateVGCSpec(instance abstract.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass,
	secretName string, secretNamespace string) volumegroupv1.VolumeGroupContentSpec {
	vgClassName := instance.GetVGCLassName()
	supportVolumeGroupSnapshot := false
	return volumegroupv1.VolumeGroupContentSpec{
		VolumeGroupClassName:       &vgClassName,
		VolumeGroupRef:             generateObjectReference(instance),
		Source:                     generateVGCSource(vgClass),
		VolumeGroupDeletionPolicy:  getVolumeGroupDeletionPolicy(vgClass),
		SupportVolumeGroupSnapshot: &supportVolumeGroupSnapshot,
		VolumeGroupSecretRef:       generateSecretReference(secretName, secretNamespace),
	}
}

func getVolumeGroupDeletionPolicy(vgClass *volumegroupv1.VolumeGroupClass) *common.VolumeGroupDeletionPolicy {
	defaultDeletionPolicy := common.VolumeGroupContentDelete
	deletionPolicy := vgClass.GetDeletionPolicy()
	if deletionPolicy != "" {
		return &deletionPolicy
	}
	return &defaultDeletionPolicy
}

func generateObjectReference(instance abstract.VolumeGroup) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:            instance.GetObjectKind().GroupVersionKind().Kind,
		Namespace:       instance.GetNamespace(),
		Name:            instance.GetName(),
		UID:             instance.GetUID(),
		APIVersion:      instance.APIVersion,
		ResourceVersion: instance.GetResourceVersion(),
	}
}

func generateSecretReference(secretName string, secretNamespace string) *corev1.SecretReference {
	return &corev1.SecretReference{
		Name:      secretName,
		Namespace: secretNamespace,
	}
}

func generateVGCSource(vgClass *volumegroupv1.VolumeGroupClass) *volumegroupv1.VolumeGroupContentSource {
	return &volumegroupv1.VolumeGroupContentSource{
		Driver: vgClass.GetDriver(),
	}
}

func RemovePVFromVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume, vgc *volumegroupv1.VolumeGroupContent) error {
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
	vgc *volumegroupv1.VolumeGroupContent) error {
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

func updateVGCStatusPVList(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger,
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

func UpdateVGCStatusError(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger, message string) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vgc.UpdateError(&common.VolumeGroupError{Message: &message})
		err := vgcRetryOnConflictFunc(client, vgc, logger)
		return err
	})
	return err
}

func vgcRetryOnConflictFunc(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) error {
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

func UpdateStaticVGCFromVG(client client.Client, vg abstract.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass, logger logr.Logger) error {
	vgc, err := GetVGC(client, logger, vg.GetVGCName(), vg.Namespace)
	if err != nil {
		return err
	}
	vgc.UpdateVGRef(generateObjectReference(vg))
	updateStaticVGCSpec(vgClass, vgc)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func UpdateStaticVGC(client client.Client, vgcNamespace, vgcName string,
	vgClass *volumegroupv1.VolumeGroupClass, logger logr.Logger) error {
	vgc, err := GetVGC(client, logger, vgcName, vgcNamespace)
	if err != nil {
		return err
	}
	updateStaticVGCSpec(vgClass, vgc)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func updateStaticVGCSpec(vgClass *volumegroupv1.VolumeGroupClass, vgc *volumegroupv1.VolumeGroupContent) {
	secretName, secretNamespace := GetSecretCred(vgClass)
	if vgc.GetVGCLassName() == "" {
		vgClassName := vgClass.GetName()
		vgc.UpdateVGClassName(vgClassName)
	}
	if vgc.GetVGSecretRef() == nil {
		vgc.UpdateSecretRef(generateSecretReference(secretName, secretNamespace))
	}
	if vgc.GetDeletionPolicy() == "" {
		vgc.UpdateDeletionPolicy(getVolumeGroupDeletionPolicy(vgClass))
	}
}

func UpdateThinVGC(client client.Client, vgcNamespace, vgcName string, logger logr.Logger) error {
	vgc, err := GetVGC(client, logger, vgcName, vgcNamespace)
	if err != nil {
		return err
	}
	updateThinVGCSpec(vgc)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func updateThinVGCSpec(vgc *volumegroupv1.VolumeGroupContent) {
	if vgc.GetDeletionPolicy() == "" {
		defaultDeletionPolicy := common.VolumeGroupContentRetain
		vgc.UpdateDeletionPolicy(&defaultDeletionPolicy)
	}
}

func UpdateVGCByResponse(client client.Client, vgc *volumegroupv1.VolumeGroupContent, resp *volumegroup.Response) error {
	CreateVGResponse := resp.Response.(*csi.CreateVolumeGroupResponse)
	vgc.UpdateVGHandle(CreateVGResponse.VolumeGroup.VolumeGroupId)
	vgc.UpdateVGAttributes(CreateVGResponse.VolumeGroup.VolumeGroupContext)
	if err := UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func DeletePVCsUnderVGC(logger logr.Logger, client client.Client, vgc *volumegroupv1.VolumeGroupContent, driver string) error {
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
		err := deletePVC(logger, client, pvcName, pvcNamespace, driver)
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
