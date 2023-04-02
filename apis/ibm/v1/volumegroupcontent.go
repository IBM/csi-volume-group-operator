package v1

import (
	"reflect"

	vgabstract "github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	"github.com/IBM/csi-volume-group-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func (vgc *VolumeGroupContent) GetVGCLassName() string {
	return utils.GetStringField(vgc.Spec, "VolumeGroupClassName")
}
func (vgc *VolumeGroupContent) GetVGHandle() string                        { return vgc.Spec.Source.VolumeGroupHandle }
func (vgc *VolumeGroupContent) GetSpec() vgabstract.VolumeGroupContentSpec { return vgc.Spec }
func (vgc *VolumeGroupContent) GetSource() vgabstract.VolumeGroupContentSource {
	return vgc.Spec.Source
}
func (vgc *VolumeGroupContent) GetVolumeGroupRef() reflect.Value {
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
func (vgc *VolumeGroupContent) GetPVList() []corev1.PersistentVolume { return vgc.Status.PVList }
