package utils

import "reflect"

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
