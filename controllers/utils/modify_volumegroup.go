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
	"fmt"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ModifyVG(logger logr.Logger, client client.Client, vg abstract.VolumeGroup,
	vgClient grpcClient.VolumeGroup, vgObjects abstract.VolumeGroupObjects) error {
	params, err := generateModifyVGParams(logger, client, vg, vgClient, vgObjects)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf(messages.ModifyVG, params.VolumeGroupID, params.VolumeIds))
	volumeGroupRequest := volumegroup.NewVolumeGroupRequest(params)
	modifyVGResponse := volumeGroupRequest.Modify()
	responseError := modifyVGResponse.Error
	if responseError != nil {
		logger.Error(responseError, fmt.Sprintf(messages.FailedToModifyVG, vg.GetNamespace(), vg.GetName()))
		return responseError
	}
	logger.Info(fmt.Sprintf(messages.ModifiedVG, params.VolumeGroupID))
	return nil
}
func generateModifyVGParams(logger logr.Logger, client client.Client, vg abstract.VolumeGroup,
	vgClient grpcClient.VolumeGroup, vgObjects abstract.VolumeGroupObjects) (volumegroup.CommonRequestParameters, error) {
	vgId, err := getVgId(logger, client, vg)
	if err != nil {
		return volumegroup.CommonRequestParameters{}, err
	}
	volumeIds, err := getPVCListVolumeIds(logger, client, vg.GetPVCList())
	if err != nil {
		return volumegroup.CommonRequestParameters{}, err
	}
	secrets, err := getSecrets(logger, client, vg, vgClass)
	if err != nil {
		return volumegroup.CommonRequestParameters{}, err
	}

	return volumegroup.CommonRequestParameters{
		Secrets:       secrets,
		VolumeGroup:   vgClient,
		VolumeGroupID: vgId,
		VolumeIds:     volumeIds,
	}, nil
}
func getSecrets(logger logr.Logger, client client.Client, vg abstract.VolumeGroup,
	vgClass abstract.VolumeGroupClass) (map[string]string, error) {
	vgc, err := GetVGClass(client, logger, vg.GetVGCLassName(), vgClass)
	if err != nil {
		return nil, err
	}
	secrets, err := GetSecretDataFromClass(client, vgc, logger)
	if err != nil {
		if uErr := UpdateVGStatusError(client, vg, logger, err.Error()); uErr != nil {
			return nil, err
		}
		return nil, err
	}
	return secrets, nil
}
