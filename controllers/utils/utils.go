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
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

func removeByIndexFromPVCList(pvcList []corev1.PersistentVolumeClaim, index int) []corev1.PersistentVolumeClaim {
	return append(pvcList[:index], pvcList[index+1:]...)
}

func removeByIndexFromPVList(pvList []corev1.PersistentVolume,
	index int) []corev1.PersistentVolume {
	return append(pvList[:index], pvList[index+1:]...)
}

func GetStringField(object interface{}, fieldName string) string {
	fieldValue := getObjectField(object, fieldName)
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return ""
	}
	return fieldValue.Elem().String()
}

func getObjectField(object interface{}, fieldName string) reflect.Value {
	objectValue := reflect.ValueOf(object)
	for i := 0; i < objectValue.NumField(); i++ {
		fieldValue := objectValue.Field(i)
		fieldType := objectValue.Type().Field(i)
		if fieldType.Name == fieldName {
			return fieldValue
		}
	}
	return reflect.ValueOf(nil)
}
