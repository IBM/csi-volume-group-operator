/*
Copyright 2022.

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

package utils

import (
	corev1 "k8s.io/api/core/v1"
)

func removeByIndexFromPVCList(pvcList []corev1.PersistentVolumeClaim, index int) []corev1.PersistentVolumeClaim {
	return append(pvcList[:index], pvcList[index+1:]...)
}

func removeByIndexFromPVList(pvList []corev1.PersistentVolume,
	index int) []corev1.PersistentVolume {
	return append(pvList[:index], pvList[index+1:]...)
}
