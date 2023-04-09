/*
Copyright 2022.

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

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getVGClassDriver(client client.Client, logger logr.Logger, vgClassName string,
	vgClass abstract.VolumeGroupClass) (string, error) {
	vgClass, err := GetVGClass(client, logger, vgClassName, vgClass)
	if err != nil {
		return "", err
	}
	return vgClass.GetDriver(), nil
}

func GetVGClass(client client.Client, logger logr.Logger, vgClassName string,
	vgClass abstract.VolumeGroupClass) (abstract.VolumeGroupClass, error) {
	if vgClassName == "" {
		return nil, fmt.Errorf("VolumeGroupClass name is empty")
	}
	err := client.Get(context.TODO(), types.NamespacedName{Name: vgClassName}, vgClass)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "VolumeGroupClass not found", "VolumeGroupClass Name", vgClassName)
		} else {
			logger.Error(err, "Got an unexpected error while fetching VolumeGroupClass", "VolumeGroupClass", vgClassName)
		}

		return nil, err
	}
	return vgClass, nil
}
