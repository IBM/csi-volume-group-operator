/*
Copyright 2025.

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
	"reflect"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
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
		func(ctx context.Context, object runtimeclient.Object) []reconcile.Request {
			var vgList volumegroupv1.VolumeGroupList
			if err := client.List(ctx, &vgList); err != nil {
				return []ctrl.Request{}
			}
			// Create a reconcile request for each matching VolumeGroup.
			var requests []ctrl.Request
			for _, vg := range vgList.Items {
				if vg.Spec.Source.Selector == nil {
					continue
				}
				isVgMatchPvc, err := areLabelsMatchLabelSelector(object.GetLabels(), *vg.Spec.Source.Selector)
				if err != nil {
					return []ctrl.Request{}
				}
				if isVgMatchPvc {
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
