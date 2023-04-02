package v1

import (
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
