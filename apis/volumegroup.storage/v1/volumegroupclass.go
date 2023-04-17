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
package v1

import (
	"github.com/IBM/csi-volume-group-operator/apis/common"
	"github.com/IBM/csi-volume-group-operator/pkg/utils"
)

func (vgc *VolumeGroupClass) GetDriver() string                { return vgc.Driver }
func (vgc *VolumeGroupClass) GetParameters() map[string]string { return vgc.Parameters }
func (vgc *VolumeGroupClass) GetDeletionPolicy() common.VolumeGroupDeletionPolicy {
	deletionPolicy := utils.GetStringField(vgc, "VolumeGroupDeletionPolicy")
	if deletionPolicy == "" {
		return ""
	}
	if deletionPolicy == string(common.VolumeGroupContentDelete) {
		return common.VolumeGroupContentDelete
	}
	return common.VolumeGroupContentRetain
}
func (vgc *VolumeGroupClass) GetSecretCred() (string, string) {
	secretName := vgc.GetParameters()[PrefixedVGSecretNameKey]
	secretNamespace := vgc.GetParameters()[PrefixedVGSecretNamespaceKey]
	return secretName, secretNamespace
}
func (vgc *VolumeGroupClass) FilterPrefixedParameters() map[string]string {
	return utils.FilterPrefixedParameters(vgAsPrefix, vgc.GetParameters())
}
func (vgc *VolumeGroupClass) ValidatePrefixedParameters() error {
	return utils.ValidatePrefixedParameters(vgc.GetParameters(), vgAsPrefix,
		PrefixedVGSecretNameKey, PrefixedVGSecretNamespaceKey)
}
