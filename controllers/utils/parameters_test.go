package utils

import (
	"reflect"
	"testing"
)

const (
	nonEmptyValue          = "test-value"
	EmptyValue             = ""
	mockParameterKey       = "test-key"
	mockParameterKeySuffix = "test-key-suffix"
)

func TestFilterPrefixedParameters(t *testing.T) {
	type args struct {
		prefix string
		param  map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "remove param with vg prefix",
			args: args{
				prefix: VolumeGroupAsPrefix,
				param: map[string]string{
					VolumeGroupAsPrefix + mockParameterKeySuffix: nonEmptyValue,
					mockParameterKey: nonEmptyValue,
				},
			},
			want: map[string]string{
				mockParameterKey: nonEmptyValue,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterPrefixedParameters(tt.args.prefix, tt.args.param); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterPrefixedParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePrefixedParameters(t *testing.T) {
	type args struct {
		param map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with vg prefix but no matching suffix",
			args: args{param: map[string]string{
				VolumeGroupAsPrefix + mockParameterKeySuffix: nonEmptyValue,
			}},
			wantErr: true,
		},
		{
			name: "with vg prefix and secret suffix but no value",
			args: args{param: map[string]string{
				PrefixedVolumeGroupSecretNameKey: EmptyValue,
			}},
			wantErr: true,
		},
		{
			name: "with vg prefix and secret namespace suffix but no value",
			args: args{param: map[string]string{
				PrefixedVolumeGroupSecretNamespaceKey: EmptyValue,
			}},
			wantErr: true,
		},
		{
			name: "with vg prefix and secret suffix with value",
			args: args{param: map[string]string{
				PrefixedVolumeGroupSecretNameKey: nonEmptyValue,
			}},
			wantErr: false,
		},
		{
			name: "with vg prefix and secret namespace suffix with value",
			args: args{param: map[string]string{
				PrefixedVolumeGroupSecretNamespaceKey: nonEmptyValue,
			}},
			wantErr: false,
		},
		{
			name: "no vg prefix",
			args: args{param: map[string]string{
				mockParameterKey: nonEmptyValue,
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePrefixedParameters(tt.args.param); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrefixedParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
