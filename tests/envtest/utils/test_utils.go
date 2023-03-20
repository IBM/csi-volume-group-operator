package utils

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
)

func GetVGCObjectFromVG(vgName, Namespace string, vgObject runtimeclient.Object,
	client runtimeclient.Client) (*volumegroupv1.VolumeGroupContent, error) {
	if err := GetNamespacedResourceObject(vgName, Namespace, vgObject, client); err != nil {
		return nil, err
	}
	vgcName := GetVGCName(vgObject.GetUID())
	vgcObj := &volumegroupv1.VolumeGroupContent{}
	err := GetNamespacedResourceObject(vgcName, Namespace, vgcObj, client)
	return vgcObj, err
}

func GetVGCName(vgUID types.UID) string {
	return fmt.Sprintf("volumegroup-%s", vgUID)
}

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

func GetNamespacedResourceObject(name, namespace string, obj runtimeclient.Object, client runtimeclient.Client) error {
	objNamespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	err := client.Get(context.Background(), objNamespacedName, obj)
	return err
}

func CreateResourceObject(obj runtimeclient.Object, client runtimeclient.Client) error {
	obj.SetResourceVersion("")
	return client.Create(context.Background(), obj)
}
