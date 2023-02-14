package utils

import (
	"context"
	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	vgclient "github.com/IBM/csi-volume-group-operator/pkg/client"
	vgfakeclient "github.com/IBM/csi-volume-group-operator/pkg/client/fake"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func createFakeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme, err := volumegroupv1.SchemeBuilder.Build()
	if err != nil {
		assert.Fail(t, "unable to build scheme")
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		assert.Fail(t, "failed to add corev1 scheme")
	}
	err = volumegroupv1.AddToScheme(scheme)
	if err != nil {
		assert.Fail(t, "failed to add replicationv1alpha1 scheme")
	}

	return scheme
}

func getVGObj() *volumegroupv1.VolumeGroup {
	vgclassName := "test-vgclass"
	vgcName := "test-vgc"
	return &volumegroupv1.VolumeGroup{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		Spec: volumegroupv1.VolumeGroupSpec{
			VolumeGroupClassName: &vgclassName,
			Source: volumegroupv1.VolumeGroupSource{
				VolumeGroupContentName: &vgcName,
			},
		},
	}
}

func getVGCObj() *volumegroupv1.VolumeGroupContent {
	vgclassName := "test-vgclass"
	return &volumegroupv1.VolumeGroupContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vgc",
			Namespace: "test-namespace",
		},
		Spec: volumegroupv1.VolumeGroupContentSpec{
			VolumeGroupClassName: &vgclassName,
			Source: &volumegroupv1.VolumeGroupContentSource{
				//Driver:                "",
				VolumeGroupHandle: "vgh",
				//VolumeGroupAttributes: nil,
			},
		},
	}

}

func getPVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "test-name",
		},
	}
}

func getRuntimeVGC(client client.Client, vgc volumegroupv1.VolumeGroupContent) *volumegroupv1.VolumeGroupContent {
	RTvgc := volumegroupv1.VolumeGroupContent{}
	namespacedVGC := types.NamespacedName{Name: vgc.Name, Namespace: vgc.Namespace}
	client.Get(context.TODO(), namespacedVGC, &RTvgc)
	return &RTvgc
}

func getPV() *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		}, Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					VolumeHandle: "vh",
				},
			},
		},
	}
}

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
				pvc: getPVC(),
				vg:  getVGObj(),
			},
			vgc:     getVGCObj(),
			wantErr: false,
		},
		{
			name: "test with err - no vgc",
			args: args{
				pvc: getPVC(),
				vg:  getVGObj(),
			},
			vgc:     nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pv := getPV()
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
}

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
			name: "t",
			args: args{
				pvcs: []corev1.PersistentVolumeClaim{*getPVC()},
				vg:   getVGObj(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := []runtime.Object{tt.args.vg, getVGCObj(), getPV()}
			log := logf.Log.WithName("controller_volumegroup_test")
			vgClient := vgfakeclient.VolumeGroup{}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj...).Build()
			if err := AddVolumesToVolumeGroup(log, fakeClient, vgClient, tt.args.pvcs, tt.args.vg); err != nil {
				if tt.wantErr {
					if len(tt.args.vg.Status.PVCList) != 100 {
						t.Errorf("AddVolumeToPvcListAndPvList() PVCListLen = %v, wantLen = 0", len(tt.args.vg.Status.PVCList))
					}
				} else {
					t.Errorf("AddVolumeToPvcListAndPvList() error = %v, wantErr %v", err, tt.wantErr)

				}
			}
		})
	}
}

func TestGetMessageFromError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMessageFromError(tt.args.err); got != tt.want {
				t.Errorf("GetMessageFromError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAddNeeded(t *testing.T) {
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
			if got := IsAddNeeded(tt.args.vg, tt.args.pvc); got != tt.want {
				t.Errorf("IsAddNeeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPVCListEqual(t *testing.T) {
	type args struct {
		x []corev1.PersistentVolumeClaim
		y []corev1.PersistentVolumeClaim
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
			if got := IsPVCListEqual(tt.args.x, tt.args.y); got != tt.want {
				t.Errorf("IsPVCListEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
