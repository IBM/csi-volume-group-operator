package abstract

import (
	"reflect"

	"github.com/IBM/csi-volume-group-operator/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeGroupContent interface {
	GetVGCLassName() string
	GetVGHandle() string
	GetVGRefName() string
	GetVGRefNamespace() string
	GetSource() reflect.Value
	GetVGRef() reflect.Value
	GetDeletionPolicy() common.VolumeGroupDeletionPolicy
	GetPVList() []corev1.PersistentVolume
	GetVGSecretRef() *corev1.SecretReference
	metav1.Object
	runtime.Object
}

type VolumeGroupContentSource interface {
}

type VolumeGroupContentList interface {
	metav1.ListInterface
	runtime.Object
}
