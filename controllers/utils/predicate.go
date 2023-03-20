package utils

import (
	"context"
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func FinalizerPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return !reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers())
		},
	}
}

func CreateRequests(kclient client.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(object client.Object) []reconcile.Request {
			// Get all VolumeGroup objects with a label selector that matches the labels of the PersistentVolumeClaim.
			//selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			//	MatchLabels: object.GetLabels(),
			//})
			//if err != nil {
			//	return []ctrl.Request{}
			//}
			//listOptions := []client.ListOption{
			//	client.MatchingLabelsSelector{Selector: selector},
			//}

			var vgList volumegroupv1.VolumeGroupList
			if err := kclient.List(context.TODO(), &vgList); err != nil {
				return []ctrl.Request{}
			}

			// Create a reconcile request for each matching VolumeGroup.
			requests := make([]ctrl.Request, len(vgList.Items))
			for _, vg := range vgList.Items {
				requests = append(requests, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: vg.Namespace,
						Name:      vg.Name,
					},
				})
			}
			return requests
		})
}
