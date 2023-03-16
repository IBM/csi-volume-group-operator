package utils

import (
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func FinalizerPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers())
		},
	}
}

func MultiPredicate() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only trigger the reconcile if the VolumeGroup or PVC's labels have changed
			if obj, ok := e.ObjectOld.(*volumegroupv1.VolumeGroup); ok {
				return !reflect.DeepEqual(obj.ObjectMeta.Generation, e.ObjectNew.(*volumegroupv1.VolumeGroup).ObjectMeta.Generation) || !reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers())
			} else if obj, ok := e.ObjectOld.(*corev1.PersistentVolumeClaim); ok {
				return !reflect.DeepEqual(obj.ObjectMeta.Labels, e.ObjectNew.(*corev1.PersistentVolumeClaim).ObjectMeta.Labels)
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			_, ok := e.Object.(*volumegroupv1.VolumeGroup)
			// Trigger the reconcile for all newly created VolumeGroups
			return ok
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			_, ok := e.Object.(*volumegroupv1.VolumeGroup)
			// Trigger the reconcile for all newly created VolumeGroups
			return ok
		},
	}

}
