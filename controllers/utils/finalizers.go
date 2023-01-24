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
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func AddFinalizerToVG(client runtimeclient.Client, logger logr.Logger, vg *volumegroupv1.VolumeGroup) error {
	if !Contains(vg.ObjectMeta.Finalizers, VolumeGroupFinalizer) {
		logger.Info("adding finalizer to VolumeGroup object", "Finalizer", VolumeGroupFinalizer)
		vg.ObjectMeta.Finalizers = append(vg.ObjectMeta.Finalizers, VolumeGroupFinalizer)
		if err := updateFinalizer(logger, client, vg.ObjectMeta.Finalizers, vg); err != nil {
			logger.Error(err, "failed to add finalizer to volumeGroup resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func AddFinalizerToVGC(client runtimeclient.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent) error {
	if !Contains(vgc.ObjectMeta.Finalizers, volumeGroupContentFinalizer) {
		logger.Info("adding finalizer to volumeGroupContent object", "Name", vgc.Name, "Finalizer", volumeGroupContentFinalizer)
		vgc.ObjectMeta.Finalizers = append(vgc.ObjectMeta.Finalizers, volumeGroupContentFinalizer)
		if err := updateFinalizer(logger, client, vgc.ObjectMeta.Finalizers, vgc); err != nil {
			logger.Error(err, "failed to add finalizer to volumeGroupContent resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func RemoveFinalizerFromVG(client runtimeclient.Client, logger logr.Logger, vg *volumegroupv1.VolumeGroup) error {
	if Contains(vg.ObjectMeta.Finalizers, VolumeGroupFinalizer) {
		logger.Info("removing finalizer from VolumeGroup object", "Finalizer", VolumeGroupFinalizer)
		vg.ObjectMeta.Finalizers = remove(vg.ObjectMeta.Finalizers, VolumeGroupFinalizer)
		if err := updateFinalizer(logger, client, vg.ObjectMeta.Finalizers, vg); err != nil {
			logger.Error(err, "failed to remove finalizer to VolumeGroup resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func RemoveFinalizerFromVGC(client runtimeclient.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent) error {
	if Contains(vgc.ObjectMeta.Finalizers, volumeGroupContentFinalizer) {
		logger.Info("removing finalizer from VolumeGroupContent object", "Name", vgc.Name, "Finalizer", volumeGroupContentFinalizer)
		vgc.ObjectMeta.Finalizers = remove(vgc.ObjectMeta.Finalizers, volumeGroupContentFinalizer)
		if err := updateFinalizer(logger, client, vgc.ObjectMeta.Finalizers, vgc); err != nil {
			logger.Error(err, "failed to remove finalizer to VolumeGroupContent resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func AddFinalizerToPVC(client runtimeclient.Client, logger logr.Logger, pvc *corev1.PersistentVolumeClaim) error {
	if !Contains(pvc.ObjectMeta.Finalizers, pvcVolumeGroupFinalizer) {
		logger.Info("adding finalizer to PersistentVolumeClaim object", "Namespace", pvc.Namespace, "Name", pvc.Name, "Finalizer", pvcVolumeGroupFinalizer)
		pvc.ObjectMeta.Finalizers = append(pvc.ObjectMeta.Finalizers, pvcVolumeGroupFinalizer)
		if err := updateFinalizer(logger, client, pvc.ObjectMeta.Finalizers, pvc); err != nil {
			logger.Error(err, "failed to add finalizer to PersistentVolumeClaim resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func RemoveFinalizerFromPVC(client runtimeclient.Client, logger logr.Logger, driver string,
	pvc *corev1.PersistentVolumeClaim) error {
	removeFinalizer, err := isFinalizerShouldBeREmovedFromPVC(logger, client, driver, pvc)
	if err != nil {
		return err
	}

	if removeFinalizer {
		logger.Info("removing finalizer from PersistentVolumeClaim object", "Namespace", pvc.Namespace, "Name", pvc.Name, "Finalizer", pvcVolumeGroupFinalizer)
		uErr := getNamespacedObject(client, pvc)
		if uErr != nil {
			return uErr
		}
		pvc.ObjectMeta.Finalizers = remove(pvc.ObjectMeta.Finalizers, pvcVolumeGroupFinalizer)
		if err := updateFinalizer(logger, client, pvc.ObjectMeta.Finalizers, pvc); err != nil {
			logger.Error(err, "failed to remove finalizer to PersistentVolumeClaim resource", "finalizer", VolumeGroupFinalizer)
			return err
		}
	}

	return nil
}

func isFinalizerShouldBeREmovedFromPVC(logger logr.Logger, client runtimeclient.Client, driver string,
	pvc *corev1.PersistentVolumeClaim) (bool, error) {
	vgList, err := GetVGList(logger, client, driver)
	if err != nil {
		return false, err
	}
	return !IsPVCPartAnyVG(pvc, vgList.Items) && Contains(pvc.ObjectMeta.Finalizers, pvcVolumeGroupFinalizer), nil
}

func updateFinalizer(logger logr.Logger, client runtimeclient.Client,
	finalizers []string, obj runtimeclient.Object) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return finalizerRetryOnConflictFunc(logger, client, finalizers, obj)
	})
	return err
}

func finalizerRetryOnConflictFunc(logger logr.Logger, client runtimeclient.Client,
	finalizers []string, obj runtimeclient.Object) error {
	obj.SetFinalizers(finalizers)
	err := UpdateObject(client, obj)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, obj)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateFinalizer))
	}
	return err
}
