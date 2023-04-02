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
	"reflect"

	"github.com/IBM/csi-volume-group-operator/apis/common"
	"github.com/IBM/csi-volume-group-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (vgc *VolumeGroupContent) GetVGCLassName() string {
	return utils.GetStringField(vgc.Spec, "VolumeGroupClassName")
}
func (vgc *VolumeGroupContent) GetVGHandle() string       { return vgc.Spec.Source.VolumeGroupHandle }
func (vgc *VolumeGroupContent) GetVGRefName() string      { return vgc.Spec.VolumeGroupRef.Name }
func (vgc *VolumeGroupContent) GetVGRefNamespace() string { return vgc.Spec.VolumeGroupRef.Namespace }
func (vgc *VolumeGroupContent) GetSource() reflect.Value {
	return utils.GetObjectField(vgc.Spec, "Source")
}
func (vgc *VolumeGroupContent) GetVGRef() reflect.Value {
	return utils.GetObjectField(vgc.Spec, "VolumeGroupRef")
}
func (vgc *VolumeGroupContent) GetDeletionPolicy() common.VolumeGroupDeletionPolicy {
	deletionPolicy := utils.GetStringField(vgc.Spec, "VolumeGroupDeletionPolicy")
	if deletionPolicy == "" {
		return ""
	}
	if deletionPolicy == string(common.VolumeGroupContentDelete) {
		return common.VolumeGroupContentDelete
	}
	return common.VolumeGroupContentRetain
}
func (vgc *VolumeGroupContent) GetVGSecretRef() *corev1.SecretReference {
	return vgc.Spec.VolumeGroupSecretRef
}
func (vgc *VolumeGroupContent) GetPVList() []corev1.PersistentVolume { return vgc.Status.PVList }
func (vgc *VolumeGroupContent) IsReady() bool {
	return utils.GetBoolField(vgc.Status, "Ready")
}
func (vgc *VolumeGroupContent) UpdateVGRef(vgRef *corev1.ObjectReference) {
	vgc.Spec.VolumeGroupRef = vgRef
}
func (vgc *VolumeGroupContent) UpdateVGClassName(vgclassName string) {
	vgc.Spec.VolumeGroupClassName = &vgclassName
}
func (vgc *VolumeGroupContent) UpdateSecretRef(secretRef *corev1.SecretReference) {
	vgc.Spec.VolumeGroupSecretRef = secretRef
}
func (vgc *VolumeGroupContent) UpdateDeletionPolicy(deletionPolicy *common.VolumeGroupDeletionPolicy) {
	vgc.Spec.VolumeGroupDeletionPolicy = deletionPolicy
}
func (vgc *VolumeGroupContent) UpdateVGHandle(vgHandle string) {
	vgc.Spec.Source.VolumeGroupHandle = vgHandle
}
func (vgc *VolumeGroupContent) UpdateVGAttributes(vgAttributes map[string]string) {
	vgc.Spec.Source.VolumeGroupAttributes = vgAttributes
}
func (vgc *VolumeGroupContent) UpdatePVList(PVList []corev1.PersistentVolume) {
	vgc.Status.PVList = PVList
}
func (vgc *VolumeGroupContent) UpdateError(vgError *common.VolumeGroupError) {
	vgc.Status.Error = vgError
}
func (vgc *VolumeGroupContent) UpdateGroupCreationTime(groupCreationTime *metav1.Time) {
	vgc.Status.GroupCreationTime = groupCreationTime
}
func (vgc *VolumeGroupContent) UpdateReady(ready bool) {
	vgc.Status.Ready = &ready
}
func (vgc *VolumeGroupContent) UpdateAPIVersion(apiVersion string) {
	vgc.APIVersion = apiVersion
}
func (vgc *VolumeGroupContent) UpdateKind(kind string) {
	vgc.Kind = kind
}
