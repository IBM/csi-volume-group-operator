package utils

import (
	"context"
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func CreateRequests(client runtimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(object runtimeclient.Object) []reconcile.Request {
			var vgList volumegroupv1.VolumeGroupList
			if err := client.List(context.TODO(), &vgList); err != nil {
				return []ctrl.Request{}
			}
			// TODO  CSI-5437 - add a label selector check to the VolumeGroup to filter the list
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
