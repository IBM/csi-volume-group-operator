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

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
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
	pvc *corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	pv, err := GetPVFromPVC(logger, client, pvc)
	if err != nil {
		return err
	}
	vgc, err := GetVGC(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return err
	}

	if pv != nil {
		return addPVToVGC(logger, client, pv, vgc)
	}
	return nil
}

func GetVGC(client client.Client, logger logr.Logger, vgcName string,
	vgName string, vgNamespace string) (*volumegroupv1.VolumeGroupContent, error) {
	logger.Info(fmt.Sprintf(messages.GetVGCOfVG, vgName, vgNamespace))
	vgc := &volumegroupv1.VolumeGroupContent{}
	namespacedVGC := types.NamespacedName{Name: vgcName, Namespace: vgNamespace}
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
	err = createSuccessVGCEvent(logger, client, vgc)
	return err
}

func createSuccessVGCEvent(logger logr.Logger, client client.Client, vgc *volumegroupv1.VolumeGroupContent) error {
	vgc.APIVersion = APIVersion
	vgc.Kind = vgcKind
	message := fmt.Sprintf(messages.VGCCreated, vgc.Namespace, vgc.Name)
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
	vgc.Status.GroupCreationTime = groupCreationTime
	vgc.Status.Ready = &ready
}

func GenerateVGC(vgname string, instance *volumegroupv1.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass, resp *volumegroup.Response, secretName string, secretNamespace string) *volumegroupv1.VolumeGroupContent {
	return &volumegroupv1.VolumeGroupContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vgname,
			Namespace: instance.Namespace,
		},
		Spec: generateVGCSpec(instance, vgClass, resp, secretName, secretNamespace),
	}
}

func generateVGCSpec(instance *volumegroupv1.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass,
	resp *volumegroup.Response, secretName string, secretNamespace string) volumegroupv1.VolumeGroupContentSpec {
	return volumegroupv1.VolumeGroupContentSpec{
		VolumeGroupClassName: instance.Spec.VolumeGroupClassName,
		VolumeGroupRef:       generateObjectReference(instance),
		Source:               generateVGCSource(vgClass, resp),
		VolumeGroupSecretRef: generateSecretReference(secretName, secretNamespace),
	}
}

func generateObjectReference(instance *volumegroupv1.VolumeGroup) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:            instance.Kind,
		Namespace:       instance.Namespace,
		Name:            instance.Name,
		UID:             instance.UID,
		APIVersion:      instance.APIVersion,
		ResourceVersion: instance.ResourceVersion,
	}
}

func generateSecretReference(secretName string, secretNamespace string) *corev1.SecretReference {
	return &corev1.SecretReference{
		Name:      secretName,
		Namespace: secretNamespace,
	}
}

func generateVGCSource(vgClass *volumegroupv1.VolumeGroupClass, resp *volumegroup.Response) *volumegroupv1.VolumeGroupContentSource {
	CreateVGResponse := resp.Response.(*csi.CreateVolumeGroupResponse)
	return &volumegroupv1.VolumeGroupContentSource{
		Driver:                vgClass.Driver,
		VolumeGroupHandle:     CreateVGResponse.VolumeGroup.VolumeGroupId,
		VolumeGroupAttributes: CreateVGResponse.VolumeGroup.VolumeGroupContext,
	}
}

func RemovePVFromVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume, vgc *volumegroupv1.VolumeGroupContent) error {
	logger.Info(fmt.Sprintf(messages.RemovePVFromVGC,
		pv.Namespace, pv.Name, vgc.Namespace, vgc.Name))
	vgc.Status.PVList = removeFromPVList(pv, vgc.Status.PVList)
	err := updateVGCStatusPVList(client, vgc, logger, vgc.Status.PVList)
	if err != nil {
		vgc.Status.PVList = appendPV(vgc.Status.PVList, *pv)
		logger.Error(err, fmt.Sprintf(messages.FailedToRemovePVFromVGC,
			pv.Name, vgc.Namespace, vgc.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedPVFromVGC,
		pv.Name, vgc.Namespace, vgc.Name))
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
	logger.Info(fmt.Sprintf(messages.AddPVToVG,
		pv.Name, vgc.Namespace, vgc.Name))
	vgc.Status.PVList = appendPV(vgc.Status.PVList, *pv)
	err := updateVGCStatusPVList(client, vgc, logger, vgc.Status.PVList)
	if err != nil {
		vgc.Status.PVList = removeFromPVList(pv, vgc.Status.PVList)
		logger.Error(err, fmt.Sprintf(messages.FailedToAddPVToVGC,
			pv.Name, vgc.Namespace, vgc.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedPVToVGC,
		pv.Name, vgc.Namespace, vgc.Name))
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
		vgc.Status.PVList = pvList
		err := vgcRetryOnConflictFunc(client, vgc, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func vgcRetryOnConflictFunc(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) error {
	err := UpdateObjectStatus(client, vgc)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, vgc)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateVGCtStatus, vgc.Namespace, vgc.Name))
	}
	return err
}

func UpdateStaticVGC(client client.Client, vg *volumegroupv1.VolumeGroup,
	vgClass *volumegroupv1.VolumeGroupClass, logger logr.Logger) error {
	vgc, err := GetVGC(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return err
	}
	updateStaticVGCSpec(vgClass, vgc, vg)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func updateStaticVGCSpec(vgClass *volumegroupv1.VolumeGroupClass, vgc *volumegroupv1.VolumeGroupContent, vg *volumegroupv1.VolumeGroup) {
	secretName, secretNamespace := GetSecretCred(vgClass)
	vgc.Spec.VolumeGroupClassName = vg.Spec.VolumeGroupClassName
	vgc.Spec.VolumeGroupRef = generateObjectReference(vg)
	vgc.Spec.Source.Driver = vgClass.Driver
	vgc.Spec.VolumeGroupSecretRef = generateSecretReference(secretName, secretNamespace)
}
