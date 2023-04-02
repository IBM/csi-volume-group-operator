package abstract

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeGroup interface {
	GetVGCName() string
	GetVGCLassName() string
	GetSpec() VolumeGroupSpec
	GetSource() VolumeGroupSource
	GetSelector() *metav1.LabelSelector
	GetPVCList() []corev1.PersistentVolume
	IsReady() bool
	metav1.Object
	runtime.Object
}

type VolumeGroupSpec interface {
}

type VolumeGroupSource interface {
}

type VolumeGroupList interface {
	metav1.ListInterface
	runtime.Object
}
