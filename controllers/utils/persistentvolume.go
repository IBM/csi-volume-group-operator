/*
Copyright 2023.

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
	"fmt"

	vgerrors "github.com/IBM/csi-volume-group-operator/pkg/errors"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetPVFromPVC(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolume, error) {
	logger.Info(fmt.Sprintf(messages.GetPersistentVolumeOfPersistentVolumeClaim, pvc.Namespace, pvc.Name))
	pvName, err := getPersistentVolumeName(logger, client, pvc)
	if err != nil {
		return nil, err
	}
	if pvName == "" {
		return nil, nil
	}

	pv, err := getPersistentVolume(logger, client, pvName)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, &vgerrors.PersistentVolumeDoesNotExist{pvName, pvc.Namespace, err.Error()}
		}
		return nil, err
	}
	return pv, nil
}

func getPersistentVolumeName(logger logr.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim) (string, error) {
	pvName := pvc.Spec.VolumeName
	if pvName == "" {
		logger.Info(messages.PersistentVolumeClaimDoesNotHavePersistentVolume)
	}
	return pvName, nil
}

func getPersistentVolume(logger logr.Logger, client client.Client, pvName string) (*corev1.PersistentVolume, error) {
	logger.Info(fmt.Sprintf(messages.GetPersistentVolume, pvName))
	pv := &corev1.PersistentVolume{}
	namespacedPV := types.NamespacedName{Name: pvName}
	err := client.Get(context.TODO(), namespacedPV, pv)
	if err != nil {
		logger.Error(err, fmt.Sprintf(messages.FailedToGetPersistentVolume, pvName))
		return nil, err
	}
	return pv, nil
}
