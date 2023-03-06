package utils

import (
	"context"

	"testing"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createFakeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme, err := volumegroupv1.SchemeBuilder.Build()
	if err != nil {
		assert.Fail(t, "unable to build scheme")
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		assert.Fail(t, "failed to add corev1 scheme")
	}
	err = volumegroupv1.AddToScheme(scheme)
	if err != nil {
		assert.Fail(t, "failed to add replicationv1alpha1 scheme")
	}

	return scheme
}

func getFakeVG() *volumegroupv1.VolumeGroup {
	vgclassName := "test-vgclass"
	vgcName := "test-vgc"
	return &volumegroupv1.VolumeGroup{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		Spec: volumegroupv1.VolumeGroupSpec{
			VolumeGroupClassName: &vgclassName,
			Source: volumegroupv1.VolumeGroupSource{
				VolumeGroupContentName: &vgcName,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"label-key": "label-value"},
				}},
		},
	}
}

func getFakeVGC() *volumegroupv1.VolumeGroupContent {
	vgclassName := "test-vgclass"
	return &volumegroupv1.VolumeGroupContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vgc",
			Namespace: "test-namespace",
		},
		Spec: volumegroupv1.VolumeGroupContentSpec{
			VolumeGroupClassName: &vgclassName,
			Source: &volumegroupv1.VolumeGroupContentSource{
				//Driver:                "",
				VolumeGroupHandle: "vgh",
				//VolumeGroupAttributes: nil,
			},
		},
	}

}

func getFakePVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"label-key": "label-value"},
			Name:      "test-name",
			Namespace: "test-namespace",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "test-name",
		},
	}
}

func getFakeVGClass() *volumegroupv1.VolumeGroupClass {
	return &volumegroupv1.VolumeGroupClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-vgclass",
		},
		//Driver:                     "",
		Parameters: map[string]string{
			"volumegroup.storage.ibm.io/secret-name":      "demo-secret",
			"volumegroup.storage.ibm.io/secret-namespace": "default",
		},
	}
}

func getFakeSecrets() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{},
	}
}

func getRuntimeVGC(client client.Client, vgc volumegroupv1.VolumeGroupContent) *volumegroupv1.VolumeGroupContent {
	RTvgc := volumegroupv1.VolumeGroupContent{}
	namespacedVGC := types.NamespacedName{Name: vgc.Name, Namespace: vgc.Namespace}
	client.Get(context.TODO(), namespacedVGC, &RTvgc)
	return &RTvgc
}

func getFakePV() *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: "test-namespace",
		}, Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					VolumeHandle: "vh",
				},
			},
		},
	}
}
