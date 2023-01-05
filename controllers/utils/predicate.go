package utils

import (
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsPVCLabelsChanged(oldObject, newObject client.Object) bool {
	return !reflect.DeepEqual(oldObject.(*corev1.PersistentVolumeClaim).Labels,
		newObject.(*corev1.PersistentVolumeClaim).Labels)
}

func IsPVCPhaseChanged(oldObject, newObject client.Object) bool {
	return !reflect.DeepEqual(oldObject.(*corev1.PersistentVolumeClaim).Status.Phase,
		newObject.(*corev1.PersistentVolumeClaim).Status.Phase)
}

func IsVGMetadataChanged(oldObject, newObject client.Object) bool {
	return !reflect.DeepEqual(oldObject.(*volumegroupv1.VolumeGroup).ObjectMeta,
		newObject.(*volumegroupv1.VolumeGroup).ObjectMeta)
}

func IsVGSpecChanged(oldObject, newObject client.Object) bool {
	return !reflect.DeepEqual(oldObject.(*volumegroupv1.VolumeGroup).Spec,
		newObject.(*volumegroupv1.VolumeGroup).Spec)
}

func IsVGFinalizersChanged(oldObject, newObject client.Object) bool {
	return !reflect.DeepEqual(oldObject.(*volumegroupv1.VolumeGroup).Finalizers,
		newObject.(*volumegroupv1.VolumeGroup).Finalizers)
}
