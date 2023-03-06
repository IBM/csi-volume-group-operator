package utils

import (
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"testing"

	vgclient "github.com/IBM/csi-volume-group-operator/pkg/client"
	vgfakeclient "github.com/IBM/csi-volume-group-operator/pkg/client/fake"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAddVolumeToPvcListAndPvList(t *testing.T) {
	t.Helper()
	scheme := createFakeScheme(t)

	type args struct {
		pvc *corev1.PersistentVolumeClaim
		vg  *volumegroupv1.VolumeGroup
	}
	tests := []struct {
		name    string
		args    args
		vgc     *volumegroupv1.VolumeGroupContent
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				pvc: getFakePVC(),
				vg:  getFakeVG(),
			},
			vgc:     getFakeVGC(),
			wantErr: false,
		},
		{
			name: "test with err - no vgc",
			args: args{
				pvc: getFakePVC(),
				vg:  getFakeVG(),
			},
			vgc:     nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := getFakePV()
			obj := []runtime.Object{tt.args.vg, tt.args.pvc, pv}
			if tt.vgc != nil {
				obj = append(obj, tt.vgc)
			}
			log := logf.Log.WithName("controller_volumegroup_test")
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj...).Build()
			if err := AddVolumeToPvcListAndPvList(log, fakeClient, tt.args.pvc, tt.args.vg); err != nil {
				if !tt.wantErr {
					t.Errorf("AddVolumesToVolumeGroup() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if len(tt.args.vg.Status.PVCList) != 1 || tt.args.vg.Status.PVCList[0].Name != "test-name" {
					t.Errorf("AddVolumeToPvcListAndPvList() PVCListLen = %v, wantLen = 1", len(tt.args.vg.Status.PVCList))
				}
				runtimeVGC := getRuntimeVGC(fakeClient, *tt.vgc)
				if len(runtimeVGC.Status.PVList) != 1 || runtimeVGC.Status.PVList[0].Name != "test-name" {
					t.Errorf("AddVolumeToPvcListAndPvList() PVListLen = %v, wantLen = 1", len(runtimeVGC.Status.PVList))
				}
				if len(tt.args.pvc.ObjectMeta.Finalizers) != 1 || tt.args.pvc.ObjectMeta.Finalizers[0] != pvcVolumeGroupFinalizer {
					t.Errorf("AddVolumeToPvcListAndPvList() FinalizersLen = %v, wantLen = 1", len(tt.args.pvc.ObjectMeta.Finalizers))
				}
			}
		})
	}
} // v

func TestAddVolumesToVolumeGroup(t *testing.T) {
	scheme := createFakeScheme(t)
	type args struct {
		pvcs []corev1.PersistentVolumeClaim
		vg   *volumegroupv1.VolumeGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success run with pvc in VG",
			args: args{
				pvcs: []corev1.PersistentVolumeClaim{*getFakePVC()},
				vg:   getFakeVG(),
			},
			wantErr: false,
		},
		{
			name: "failed run with no pvc in VG",
			args: args{
				pvcs: []corev1.PersistentVolumeClaim{*getFakePVC()},
				vg:   getFakeVG(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := []runtime.Object{tt.args.vg, getFakeVGC(), getFakePV(), getFakeVGClass(), getFakeSecrets()}
			log := logf.Log.WithName("controller_volumegroup_test")
			vgClient := vgfakeclient.VolumeGroup{
				ModifyVolumeGroupMembershipMock: func(volumeGroupId string, volumeIds []string, secrets map[string]string) (*csi.ModifyVolumeGroupMembershipResponse, error) {
					if tt.wantErr {
						return nil, fmt.Errorf("error with modify func")
					}
					return &csi.ModifyVolumeGroupMembershipResponse{}, nil
				},
			}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj...).Build()
			if err := AddVolumesToVolumeGroup(log, fakeClient, vgClient, tt.args.pvcs, tt.args.vg); err != nil {
				if tt.wantErr {
					if len(tt.args.vg.Status.PVCList) != 0 {
						t.Errorf("AddVolumeToPvcListAndPvList() PVCListLen = %v, wantLen = 0", len(tt.args.vg.Status.PVCList))
					}
				} else {
					t.Errorf("AddVolumeToPvcListAndPvList() error = %v, wantErr %v", err, tt.wantErr)

				}
			} else {
				if len(tt.args.vg.Status.PVCList) != 1 {
					t.Errorf("AddVolumeToPvcListAndPvList() PVCListLen = %v, wantLen = 1", len(tt.args.vg.Status.PVCList))
				}
			}
		})
	}
} // v

func TestGetMessageFromError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "success msg",
			args: args{err: fmt.Errorf("err msg")},
			want: "err msg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageFromError(tt.args.err); got != tt.want {
				t.Errorf("GetMessageFromError() = %v, addNeededWantedResult %v", got, tt.want)
			}
		})
	}
} // v

func TestIsAddNeeded(t *testing.T) {
	tests := []struct {
		name                  string
		vgReady               bool
		pvcInVg               bool
		addNeededWantedResult bool
	}{
		{
			name:                  "add is needed",
			vgReady:               true,
			addNeededWantedResult: true,
		},
		{
			name:                  "add is not needed - vg not ready",
			vgReady:               false,
			addNeededWantedResult: false,
		},
		{
			name: "add is not needed - vg ready, pvc in vg",

			vgReady:               true,
			pvcInVg:               true,
			addNeededWantedResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vg := getFakeVG()
			pvc := getFakePVC()
			vg.Status.Ready = &tt.vgReady
			if tt.pvcInVg {
				vg.Status.PVCList = append(vg.Status.PVCList, *pvc)
			}
			if addNeededResult := IsAddNeeded(*vg, pvc); addNeededResult != tt.addNeededWantedResult {
				t.Errorf("IsAddNeeded() = %v, addNeededWantedResult %v", addNeededResult, tt.addNeededWantedResult)
			}
		})
	}
} // v

func TestIsPVCListEqual(t *testing.T) {
	pvc1 := *getFakePVC()
	pvc2 := *getFakePVC()
	pvc2.Name = "second pvc"
	type args struct {
		x []corev1.PersistentVolumeClaim
		y []corev1.PersistentVolumeClaim
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "equal empty",
			args: args{
				x: []corev1.PersistentVolumeClaim{},
				y: []corev1.PersistentVolumeClaim{},
			},
			want: true,
		},
		{
			name: "equal with pvc",
			args: args{
				x: []corev1.PersistentVolumeClaim{pvc1},
				y: []corev1.PersistentVolumeClaim{pvc1},
			},
			want: true,
		},
		{
			name: "not equal with pvc in one",
			args: args{
				x: []corev1.PersistentVolumeClaim{pvc1},
				y: []corev1.PersistentVolumeClaim{},
			},
			want: false,
		},
		{
			name: "not equal with pvc in one",
			args: args{
				x: []corev1.PersistentVolumeClaim{pvc1},
				y: []corev1.PersistentVolumeClaim{pvc2},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPVCListEqual(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("IsPVCListEqual() = %v, addNeededWantedResult %v", got, tt.want)
			}
		})
	}
} // v

func TestIsRemoveNeeded(t *testing.T) {
	type args struct {
		vg  volumegroupv1.VolumeGroup
		pvc *corev1.PersistentVolumeClaim
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRemoveNeeded(tt.args.vg, tt.args.pvc); got != tt.want {
				t.Errorf("IsRemoveNeeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModifyVolumesInVG(t *testing.T) {
	type args struct {
		logger       logr.Logger
		client       client.Client
		vgClient     vgclient.VolumeGroup
		matchingPvcs []corev1.PersistentVolumeClaim
		vg           volumegroupv1.VolumeGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ModifyVolumesInVG(tt.args.logger, tt.args.client, tt.args.vgClient, tt.args.matchingPvcs, tt.args.vg); (err != nil) != tt.wantErr {
				t.Errorf("ModifyVolumesInVG() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveVolumeFromPvcListAndPvList(t *testing.T) {
	type args struct {
		logger logr.Logger
		client client.Client
		driver string
		pvc    corev1.PersistentVolumeClaim
		vg     *volumegroupv1.VolumeGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RemoveVolumeFromPvcListAndPvList(tt.args.logger, tt.args.client, tt.args.driver, tt.args.pvc, tt.args.vg); (err != nil) != tt.wantErr {
				t.Errorf("RemoveVolumeFromPvcListAndPvList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveVolumeFromVolumeGroup(t *testing.T) {
	type args struct {
		logger   logr.Logger
		client   client.Client
		vgClient vgclient.VolumeGroup
		pvcs     []corev1.PersistentVolumeClaim
		vg       *volumegroupv1.VolumeGroup
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RemoveVolumeFromVolumeGroup(tt.args.logger, tt.args.client, tt.args.vgClient, tt.args.pvcs, tt.args.vg); (err != nil) != tt.wantErr {
				t.Errorf("RemoveVolumeFromVolumeGroup() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateObject(t *testing.T) {
	type args struct {
		client       client.Client
		updateObject client.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateObject(tt.args.client, tt.args.updateObject); (err != nil) != tt.wantErr {
				t.Errorf("UpdateObject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateObjectStatus(t *testing.T) {
	type args struct {
		client       client.Client
		updateObject client.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateObjectStatus(tt.args.client, tt.args.updateObject); (err != nil) != tt.wantErr {
				t.Errorf("UpdateObjectStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdatePvcAndPvList(t *testing.T) {
	type args struct {
		logger       logr.Logger
		vg           *volumegroupv1.VolumeGroup
		client       client.Client
		driver       string
		matchingPvcs []corev1.PersistentVolumeClaim
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdatePvcAndPvList(tt.args.logger, tt.args.vg, tt.args.client, tt.args.driver, tt.args.matchingPvcs); (err != nil) != tt.wantErr {
				t.Errorf("UpdatePvcAndPvList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_generateString(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateString(); got != tt.want {
				t.Errorf("generateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getNamespacedObject(t *testing.T) {
	type args struct {
		client client.Client
		obj    client.Object
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := getNamespacedObject(tt.args.client, tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("getNamespacedObject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
