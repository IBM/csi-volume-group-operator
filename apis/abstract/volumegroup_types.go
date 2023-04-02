package abstract

import (
	"github.com/IBM/csi-volume-group-operator/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeGroup interface {
	GetVGCName() string
	GetVGCLassName() string
	GetSelector() *metav1.LabelSelector
	GetPVCList() []corev1.PersistentVolumeClaim
	IsReady() bool
	UpdateVGCName(vgcName string)
	UpdateBoundVGCName(vgcName string)
	UpdateGroupCreationTime(groupCreationTime *metav1.Time)
	UpdateReady(ready bool)
	UpdateError(vgError *common.VolumeGroupError)
	UpdatePVCList(PVCList []corev1.PersistentVolumeClaim)
	metav1.Object
	runtime.Object
}

type VolumeGroupList interface {
	metav1.ListInterface
	runtime.Object
}
