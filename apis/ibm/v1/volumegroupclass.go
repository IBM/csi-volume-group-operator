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
	return common.VolumeGroupContentDelete
}
