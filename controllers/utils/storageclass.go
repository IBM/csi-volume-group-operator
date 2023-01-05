package utils

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"

	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPersistentVolumeClaimClass(claim *corev1.PersistentVolumeClaim) (string, error) {
	if class, found := claim.Annotations[corev1.BetaStorageClassAnnotation]; found {
		return class, nil
	}

	if claim.Spec.StorageClassName != nil {
		return *claim.Spec.StorageClassName, nil
	}

	err := fmt.Errorf(messages.FailedToGetStorageClassName, claim.Name)
	return "", err
}

func getStorageClassProvisioner(logger logr.Logger, client client.Client, scName string) (string, error) {
	sc, err := getStorageClass(logger, client, scName)
	if err != nil {
		return "", err
	}
	return sc.Provisioner, nil
}

func isSCHasParam(sc *storagev1.StorageClass, param string) bool {
	scParams := sc.Parameters
	_, ok := scParams[param]
	return ok
}

func getStorageClass(logger logr.Logger, client client.Client, scName string) (*storagev1.StorageClass, error) {
	sc := &storagev1.StorageClass{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: scName}, sc)
	if err != nil {
		logger.Error(err, fmt.Sprintf(messages.FailedToGetStorageClass, scName))
		return nil, err
	}
	return sc, nil
}
