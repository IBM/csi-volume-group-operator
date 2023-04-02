package abstract

import (
	"github.com/IBM/csi-volume-group-operator/apis/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeGroupClass interface {
	metav1.Object
	runtime.Object
	GetDriver() string
	GetParameters() map[string]string
	GetDeletionPolicy() common.VolumeGroupDeletionPolicy
}

type VolumeGroupClassList interface {
	metav1.ListInterface
	runtime.Object
}
