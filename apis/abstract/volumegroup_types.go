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
	"github.com/IBM/csi-volume-group-operator/apis/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type VolumeGroup interface {
	GetVGCName() string
	GetVGCLassName() string
	GetApiVersion() string
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
