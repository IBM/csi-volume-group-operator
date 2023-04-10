package mock_grpc_server

import (
	"context"

	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
)

type MockControllerServer struct {
	*csi.UnimplementedControllerServer
}

func (MockControllerServer) CreateVolumeGroup(context.Context, *csi.CreateVolumeGroupRequest) (*csi.CreateVolumeGroupResponse, error) {
	return &csi.CreateVolumeGroupResponse{
		VolumeGroup: &csi.VolumeGroup{
			VolumeGroupId: "test",
			VolumeGroupContext: map[string]string{
				"fake-label": "fake-value",
			},
		},
	}, nil
}

func (MockControllerServer) DeleteVolumeGroup(context.Context, *csi.DeleteVolumeGroupRequest) (*csi.DeleteVolumeGroupResponse, error) {
	return &csi.DeleteVolumeGroupResponse{}, nil
}

func (MockControllerServer) ModifyVolumeGroupMembership(context.Context, *csi.ModifyVolumeGroupMembershipRequest) (*csi.ModifyVolumeGroupMembershipResponse, error) {
	return &csi.ModifyVolumeGroupMembershipResponse{}, nil
}

func (MockControllerServer) ListVolumeGroups(context.Context, *csi.ListVolumeGroupsRequest) (*csi.ListVolumeGroupsResponse, error) {
	return &csi.ListVolumeGroupsResponse{}, nil
}
func (MockControllerServer) ControllerGetVolumeGroup(context.Context, *csi.ControllerGetVolumeGroupRequest) (*csi.ControllerGetVolumeGroupResponse, error) {
	return &csi.ControllerGetVolumeGroupResponse{}, nil
}
