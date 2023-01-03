package utils

import (
	"context"
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getPVCListVolumeIds(logger logr.Logger, client runtimeclient.Client, pvcList []corev1.PersistentVolumeClaim) ([]string, error) {
	volumeIds := []string{}
	for _, pvc := range pvcList {
		pv, err := GetPVFromPVC(logger, client, &pvc)
		if err != nil {
			return nil, err
		}
		if pv != nil {
			volumeIds = append(volumeIds, string(pv.Spec.CSI.VolumeHandle))
		}
	}
	return volumeIds, nil
}

func GetPersistentVolumeClaim(logger logr.Logger, client runtimeclient.Client, name, namespace string) (*corev1.PersistentVolumeClaim, error) {
	logger.Info(fmt.Sprintf(messages.GetPersistentVolumeClaim, namespace, name))
	pvc := &corev1.PersistentVolumeClaim{}
	namespacedPVC := types.NamespacedName{Name: name, Namespace: namespace}
	err := client.Get(context.TODO(), namespacedPVC, pvc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, fmt.Sprintf(messages.PersistentVolumeClaimNotFound, namespace, name))
		} else {
			logger.Error(err, fmt.Sprintf(messages.UnExpectedPersistentVolumeClaimError, namespace, name))
		}
		return nil, err
	}
	return pvc, nil
}

func IsPVCCanBeAddedToVG(logger logr.Logger, client runtimeclient.Client,
	pvc *corev1.PersistentVolumeClaim, vgs []volumegroupv1.VolumeGroup) error {
	vgsWithPVC := []string{}
	newVGsForPVC := []string{}
	for _, vg := range vgs {
		if IsPVCPartOfVG(pvc, vg.Status.PVCList) {
			vgsWithPVC = append(vgsWithPVC, vg.Name)
		} else if isPVCMatchesVG, _ := IsPVCMatchesVG(logger, client, pvc, vg); isPVCMatchesVG {
			newVGsForPVC = append(newVGsForPVC, vg.Name)
		}
	}
	return checkIfPVCCanBeAddedToVG(logger, pvc, vgsWithPVC, newVGsForPVC)
}

func checkIfPVCCanBeAddedToVG(logger logr.Logger, pvc *corev1.PersistentVolumeClaim,
	vgsWithPVC, newVGsForPVC []string) error {
	if len(vgsWithPVC) > 0 && len(newVGsForPVC) > 0 {
		message := fmt.Sprintf(messages.PersistentVolumeClaimIsAlreadyBelongToGroup, pvc.Namespace, pvc.Name, newVGsForPVC, vgsWithPVC)
		logger.Info(message)
		return fmt.Errorf(message)
	}
	if len(newVGsForPVC) > 1 {
		message := fmt.Sprintf(messages.PersistentVolumeClaimMatchedWithMultipleNewGroups, pvc.Namespace, pvc.Name, newVGsForPVC)
		logger.Info(message)
		return fmt.Errorf(message)
	}
	return nil
}

func IsPVCInStaticVG(logger logr.Logger, client runtimeclient.Client, pvc *corev1.PersistentVolumeClaim) (bool, error) {
	sc, err := getStorageClass(logger, client, *pvc.Spec.StorageClassName)
	if err != nil {
		return false, err
	}
	return isSCHasParam(sc, storageClassVGParameter), nil
}

func GetPVCList(logger logr.Logger, client runtimeclient.Client, driver string) (corev1.PersistentVolumeClaimList, error) {
	pvcList, err := getPVCList(logger, client)
	if err != nil {
		return corev1.PersistentVolumeClaimList{}, err
	}
	boundPVCList, err := getBoundPVCList(pvcList)
	if err != nil {
		return corev1.PersistentVolumeClaimList{}, err
	}

	return getProvisionedPVCList(logger, client, driver, boundPVCList)
}

func getPVCList(logger logr.Logger, client runtimeclient.Client) (corev1.PersistentVolumeClaimList, error) {
	logger.Info(messages.ListPersistentVolumeClaim)
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := client.List(context.TODO(), pvcList); err != nil {
		logger.Error(err, messages.FailedToListPersistentVolumeClaim)
		return corev1.PersistentVolumeClaimList{}, err
	}
	return *pvcList, nil
}

func getBoundPVCList(pvcList corev1.PersistentVolumeClaimList) (corev1.PersistentVolumeClaimList, error) {
	newPVCList := corev1.PersistentVolumeClaimList{}
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase == corev1.ClaimBound {
			newPVCList.Items = append(newPVCList.Items, pvc)
		}
	}
	return newPVCList, nil
}

func getProvisionedPVCList(logger logr.Logger, client runtimeclient.Client, driver string,
	pvcList corev1.PersistentVolumeClaimList) (corev1.PersistentVolumeClaimList, error) {
	newPVCList := corev1.PersistentVolumeClaimList{}
	for _, pvc := range pvcList.Items {
		isPVCHasMatchingDriver, err := IsPVCHasMatchingDriver(logger, client, &pvc, driver)
		if err != nil {
			return corev1.PersistentVolumeClaimList{}, err
		}
		if isPVCHasMatchingDriver {
			newPVCList.Items = append(newPVCList.Items, pvc)
		}
	}
	return newPVCList, nil
}

func IsPVCHasMatchingDriver(logger logr.Logger, client runtimeclient.Client,
	pvc *corev1.PersistentVolumeClaim, driver string) (bool, error) {
	scProvisioner, err := getStorageClassProvisioner(logger, client, *pvc.Spec.StorageClassName)
	if err != nil {
		return false, err
	}
	return scProvisioner == driver, nil
}
