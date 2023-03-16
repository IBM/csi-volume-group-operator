package fake

import (
	"time"

	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
	"google.golang.org/grpc"
)

type VolumeGroup struct {
	NewVolumeGroupClient            func(cc *grpc.ClientConn, timeout time.Duration) VolumeGroup
	CreateVolumeGroupMock           func(name string, secrets, parameters map[string]string) (*csi.CreateVolumeGroupResponse, error)
	DeleteVolumeGroupMock           func(volumeGroupId string, secrets map[string]string) (*csi.DeleteVolumeGroupResponse, error)
	ModifyVolumeGroupMembershipMock func(volumeGroupId string, volumeIds []string, secrets map[string]string) (*csi.ModifyVolumeGroupMembershipResponse, error)
}

func (v VolumeGroup) CreateVolumeGroup(name string, secrets, parameters map[string]string) (*csi.CreateVolumeGroupResponse, error) {
	return v.CreateVolumeGroupMock(name, secrets, parameters)
}

func (v VolumeGroup) DeleteVolumeGroup(volumeGroupId string, secrets map[string]string) (*csi.DeleteVolumeGroupResponse, error) {
	return v.DeleteVolumeGroupMock(volumeGroupId, secrets)
}

func (v VolumeGroup) ModifyVolumeGroupMembership(volumeGroupId string, volumeIds []string, secrets map[string]string) (*csi.ModifyVolumeGroupMembershipResponse, error) {
	return v.ModifyVolumeGroupMembershipMock(volumeGroupId, volumeIds, secrets)
}
