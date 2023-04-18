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
package utils

import (
	"reflect"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/apis/common"
	corev1 "k8s.io/api/core/v1"
)

func GetStringField(object interface{}, fieldName string) string {
	fieldValue := GetObjectField(object, fieldName)
	if !fieldValue.IsValid() {
		return ""
	}
	if fieldValue.Kind() == reflect.String {
		return fieldValue.String()
	}
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return ""
	}
	return fieldValue.Elem().String()
}

func GetBoolField(object interface{}, fieldName string) bool {
	fieldValue := GetObjectField(object, fieldName)
	if !fieldValue.IsValid() {
		return false
	}
	if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
		return false
	}
	return fieldValue.Elem().Bool()
}

func GetObjectField(object interface{}, fieldName string) reflect.Value {
	objectValue := reflect.ValueOf(object)
	if objectValue.Kind() == reflect.Ptr {
		objectValue = objectValue.Elem()
	}
	if !objectValue.IsValid() {
		return reflect.ValueOf(nil)
	}
	for i := 0; i < objectValue.NumField(); i++ {
		fieldValue := objectValue.Field(i)
		fieldType := objectValue.Type().Field(i)
		if fieldType.Name == fieldName {
			return fieldValue
		}
	}
	return reflect.ValueOf(nil)
}

func GenerateSecretReference(secretName string, secretNamespace string) *corev1.SecretReference {
	return &corev1.SecretReference{
		Name:      secretName,
		Namespace: secretNamespace,
	}
}

func GetVolumeGroupDeletionPolicy(vgClass abstract.VolumeGroupClass) *common.VolumeGroupDeletionPolicy {
	defaultDeletionPolicy := common.VolumeGroupContentDelete
	deletionPolicy := vgClass.GetDeletionPolicy()
	if deletionPolicy != "" {
		return &deletionPolicy
	}
	return &defaultDeletionPolicy
}

func GenerateObjectReference(vg abstract.VolumeGroup) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:            vg.GetObjectKind().GroupVersionKind().Kind,
		Namespace:       vg.GetNamespace(),
		Name:            vg.GetName(),
		UID:             vg.GetUID(),
		APIVersion:      vg.GetApiVersion(),
		ResourceVersion: vg.GetResourceVersion(),
	}
}
