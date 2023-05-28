package utils

import (
	"context"
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	PvcPredicate = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isLabelsChanged(e.ObjectOld, e.ObjectNew) || isPhaseChanged(e.ObjectOld, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
	FinalizerPredicate = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers())
		},
	}
)

func isLabelsChanged(oldObject, newObject runtimeclient.Object) bool {
	return !reflect.DeepEqual(oldObject.(*corev1.PersistentVolumeClaim).Labels,
		newObject.(*corev1.PersistentVolumeClaim).Labels)
}

func isPhaseChanged(oldObject, newObject runtimeclient.Object) bool {
	return !reflect.DeepEqual(oldObject.(*corev1.PersistentVolumeClaim).Status.Phase,
		newObject.(*corev1.PersistentVolumeClaim).Status.Phase)
}

func CreateRequests(client runtimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(object runtimeclient.Object) []reconcile.Request {
			var vgList volumegroupv1.VolumeGroupList
			if err := client.List(context.TODO(), &vgList); err != nil {
				return []ctrl.Request{}
			}
			pvc := &corev1.PersistentVolumeClaim{}
			if err := client.Get(context.TODO(), types.NamespacedName{
				Namespace: object.GetNamespace(),
				Name:      object.GetName(),
			}, pvc); err != nil {
				return []ctrl.Request{}
			}
			// Create a reconcile request for each matching VolumeGroup.
			requests := make([]ctrl.Request, len(vgList.Items))

			for _, vg := range vgList.Items {
				selector, err := metav1.LabelSelectorAsSelector(vg.Spec.Source.Selector)
				if err != nil {
					continue
				}
				if selector.Matches(labels.Set(pvc.Labels)) {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Namespace: vg.Namespace,
							Name:      vg.Name,
						},
					})
				}
			}
			return requests
		})
}
