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

package envtest

import (
	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	VG = &volumegroupv1.VolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      VGName,
			Namespace: Namespace,
		},
		Spec: volumegroupv1.VolumeGroupSpec{
			VolumeGroupClassName: &VGClassName,
			Source: volumegroupv1.VolumeGroupSource{
				Selector: &metav1.LabelSelector{
					MatchLabels: FakeMatchLabels,
				},
			},
		},
	}
	VGClass = &volumegroupv1.VolumeGroupClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: VGClassName,
		},
		Driver:     DriverName,
		Parameters: StorageClassParameters,
	}
	Secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SecretName,
			Namespace: Namespace,
		},
	}
)
