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
package vgcontroller

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForCrds(logger logr.Logger, client client.Client, vgGroup, vgVersion string) error {
	err := waitForVGResource(logger, client, VolumeGroup, vgGroup, vgVersion)
	if err != nil {
		logger.Error(err, "failed to wait for VolumeGroup CRD")

		return err
	}

	err = waitForVGResource(logger, client, VolumeGroupClass, vgGroup, vgVersion)
	if err != nil {
		logger.Error(err, "failed to wait for VolumeGroupClass CRD")

		return err
	}

	err = waitForVGResource(logger, client, VolumeGroupContent, vgGroup, vgVersion)
	if err != nil {
		logger.Error(err, "failed to wait for VolumeGroupContent CRD")

		return err
	}

	return nil
}

func waitForVGResource(logger logr.Logger, client client.Client, resourceName, vgGroup, vgVersion string) error {
	unstructuredResource := &unstructured.UnstructuredList{}
	unstructuredResource.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   vgGroup,
		Kind:    resourceName,
		Version: vgVersion,
	})
	for {
		err := client.List(context.TODO(), unstructuredResource)
		if err == nil {
			return nil
		}
		// return errors other than NoMatch
		if !meta.IsNoMatchError(err) {
			logger.Error(err, "got an unexpected error while waiting for resource", "Resource", resourceName)

			return err
		}
		logger.Info("resource does not exist", "Resource", resourceName)
		time.Sleep(5 * time.Second)
	}
}
