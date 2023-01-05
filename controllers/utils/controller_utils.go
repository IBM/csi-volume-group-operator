package utils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateObject(client client.Client, updateObject client.Object) error {
	if err := client.Update(context.TODO(), updateObject); err != nil {
		return fmt.Errorf("failed to update %s (%s/%s) %w", updateObject.GetObjectKind(), updateObject.GetNamespace(), updateObject.GetName(), err)
	}
	return nil
}

func UpdateObjectStatus(client client.Client, updateObject client.Object) error {
	if err := client.Status().Update(context.TODO(), updateObject); err != nil {
		if apierrors.IsConflict(err) {
			return err
		}
		return fmt.Errorf("failed to update %s (%s/%s) status %w", updateObject.GetObjectKind(), updateObject.GetNamespace(), updateObject.GetName(), err)
	}
	return nil
}

func getNamespacedObject(client client.Client, obj client.Object) error {
	namespacedObject := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	err := client.Get(context.TODO(), namespacedObject, obj)
	if err != nil {
		return err
	}
	return nil
}

func GetMessageFromError(err error) string {
	s, ok := status.FromError(err)
	if !ok {
		// This is not gRPC error. The operation must have failed before gRPC
		// method was called, otherwise we would get gRPC error.
		return err.Error()
	}

	return s.Message()
}

func generateString() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 16)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func AddVolumesToVolumeGroup(logger logr.Logger, client client.Client, vgClient grpcClient.VolumeGroup,
	pvcs []corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.AddVolumeToVolumeGroup, vg.Namespace, vg.Name))
	vg.Status.PVCList = appendMultiplePVCs(vg.Status.PVCList, pvcs)

	err := ModifyVolumeGroup(logger, client, vg, vgClient)
	if err != nil {
		vg.Status.PVCList = removeMultiplePVCs(vg.Status.PVCList, pvcs)
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedVolumeToVolumeGroup, vg.Namespace, vg.Name))
	return nil
}

func AddVolumeToPvcListAndPvList(logger logr.Logger, client client.Client,
	pvc *corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	err := AddPVCToVG(logger, client, pvc, vg)
	if err != nil {
		return err
	}

	err = AddMatchingPVToMatchingVGC(logger, client, pvc, vg)
	if err != nil {
		return err
	}

	if err = AddFinalizerToPVC(client, logger, pvc); err != nil {
		return err
	}

	message := fmt.Sprintf(messages.AddedPersistentVolumeClaimToVolumeGroup, pvc.Namespace, pvc.Name, vg.Namespace, vg.Name)
	return HandleSuccessMessage(logger, client, vg, message, addingPVC)
}

func RemoveVolumeFromVolumeGroup(logger logr.Logger, client client.Client, vgClient grpcClient.VolumeGroup,
	pvcs []corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	logger.Info(fmt.Sprintf(messages.RemoveVolumeFromVolumeGroup, vg.Namespace, vg.Name))
	vg.Status.PVCList = removeMultiplePVCs(vg.Status.PVCList, pvcs)

	err := ModifyVolumeGroup(logger, client, vg, vgClient)
	if err != nil {
		vg.Status.PVCList = appendMultiplePVCs(vg.Status.PVCList, pvcs)
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedVolumeFromVolumeGroup, vg.Namespace, vg.Name))
	return nil
}

func RemoveVolumeFromPvcListAndPvList(logger logr.Logger, client client.Client, driver string,
	pvc *corev1.PersistentVolumeClaim, vg volumegroupv1.VolumeGroup) error {
	err := RemovePVCFromVG(logger, client, pvc, &vg)
	if err != nil {
		return err
	}
	pv, err := GetPVFromPVC(logger, client, pvc)
	if err != nil {
		return err
	}
	vgc, err := GetVolumeGroupContent(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return err
	}

	if pv != nil {
		err = RemovePVFromVGC(logger, client, pv, vgc)
		if err != nil {
			return err
		}
	}

	err = RemoveFinalizerFromPVC(client, logger, driver, pvc)
	if err != nil {
		return err
	}

	message := fmt.Sprintf(messages.RemovedPersistentVolumeClaimFromVolumeGroup, pvc.Namespace, pvc.Name, vg.Namespace, vg.Name)
	return HandleSuccessMessage(logger, client, &vg, message, removingPVC)
}
