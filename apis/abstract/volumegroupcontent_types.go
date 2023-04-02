/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
	IsReady() bool
	UpdateVGRef(vgRef *corev1.ObjectReference)
	UpdateVGClassName(vgclassName string)
	UpdateSecretRef(secretRef *corev1.SecretReference)
	UpdateDeletionPolicy(deletionPolicy *common.VolumeGroupDeletionPolicy)
	UpdateVGHandle(vgHandle string)
	UpdateVGAttributes(vgAttributes map[string]string)
	UpdatePVList(PVList []corev1.PersistentVolume)
	UpdateError(vgError *common.VolumeGroupError)
	UpdateGroupCreationTime(groupCreationTime *metav1.Time)
	UpdateReady(ready bool)
	UpdateAPIVersion(apiVersion string)
	UpdateKind(kind string)
	metav1.Object
	runtime.Object
}

type VolumeGroupContentSource interface {
}

type VolumeGroupContentList interface {
	metav1.ListInterface
	runtime.Object
}
