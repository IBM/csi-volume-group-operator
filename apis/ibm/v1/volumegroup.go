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
	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	"github.com/IBM/csi-volume-group-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (vg *VolumeGroup) GetVGCName() string {
	return utils.GetStringField(vg.Spec.Source, "VolumeGroupContentName")
}
func (vg *VolumeGroup) GetVGCLassName() string {
	return utils.GetStringField(vg.Spec, "VolumeGroupClassName")
}
func (vg *VolumeGroup) GetApiVersion() string                      { return vg.APIVersion }
func (vg *VolumeGroup) GetSelector() *metav1.LabelSelector         { return vg.Spec.Source.Selector }
func (vg *VolumeGroup) GetPVCList() []corev1.PersistentVolumeClaim { return vg.Status.PVCList }
func (vg *VolumeGroup) IsReady() bool {
	return utils.GetBoolField(vg.Status, "Ready")
}
func (vg *VolumeGroup) UpdateVGCName(vgcName string) {
	vg.Spec.Source.VolumeGroupContentName = &vgcName
}
func (vg *VolumeGroup) UpdateBoundVGCName(vgcName string) {
	vg.Status.BoundVolumeGroupContentName = &vgcName
}
func (vg *VolumeGroup) UpdateGroupCreationTime(groupCreationTime *metav1.Time) {
	vg.Status.GroupCreationTime = groupCreationTime
}
func (vg *VolumeGroup) UpdateReady(ready bool) {
	vg.Status.Ready = &ready
}
func (vg *VolumeGroup) UpdateError(vgError *common.VolumeGroupError) {
	vg.Status.Error = vgError
}
func (vg *VolumeGroup) UpdatePVCList(PVCList []corev1.PersistentVolumeClaim) {
	vg.Status.PVCList = PVCList
}
func (vg *VolumeGroup) Copy() abstract.VolumeGroup {
	return vg.DeepCopy()
}

func (vgList *VolumeGroupList) GetItems() []abstract.VolumeGroup {
	vgs := []abstract.VolumeGroup{}
	for _, vg := range vgList.Items {
		vgs = append(vgs, &vg)
	}
	return vgs
}
