package utils

import (
	"context"

	commonUtils "github.com/IBM/csi-volume-group-operator/controllers/common/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func RemoveResourceObjectFinalizers(name, namespace string, obj runtimeclient.Object, client runtimeclient.Client) error {
	err := GetNamespacedResourceObject(name, namespace, obj, client)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		obj.SetFinalizers([]string{})
		err := client.Update(context.TODO(), obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func RemoveFinalizerFromPVC(name, namespace, finalizer string, client runtimeclient.Client) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := GetNamespacedResourceObject(name, namespace, pvc, client); err != nil {
		return err
	}
	if commonUtils.Contains(pvc.ObjectMeta.Finalizers, finalizer) {
		pvc.ObjectMeta.Finalizers = commonUtils.Remove(pvc.ObjectMeta.Finalizers, finalizer)
	}
	err := client.Update(context.TODO(), pvc)
	return err
}
