package envtest

import (
	"context"
	"fmt"
	"time"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Controller", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1
	var (
		vg          *volumegroupv1.VolumeGroup
		vgclass     *volumegroupv1.VolumeGroupClass
		secret      *corev1.Secret
		vgName      = "vg-name"
		vgClassName = "vgclass-name"
		namespace   = "default"
		secretName  = "fake-secret"
	)

	BeforeEach(func() {
		vg = &volumegroupv1.VolumeGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vgName,
				Namespace: namespace,
			},
			Spec: volumegroupv1.VolumeGroupSpec{
				VolumeGroupClassName: &vgClassName,
				Source: volumegroupv1.VolumeGroupSource{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"fake-label": "fake-value",
						},
					},
				},
			},
		}
		vgclass = &volumegroupv1.VolumeGroupClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: vgClassName,
			},
			Driver: driverName,
			Parameters: map[string]string{
				"volumegroup.storage.ibm.io/secret-name":      secretName,
				"volumegroup.storage.ibm.io/secret-namespace": namespace,
			},
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
		}
	})

	Describe("test vg controller", func() {

		Context("create an vg instance", func() {

			It("should create all the relevant objects", func(done Done) {
				err := k8sClient.Create(context.Background(), vgclass)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), secret)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), vg)
				Expect(err).NotTo(HaveOccurred())

				vgObj := &volumegroupv1.VolumeGroup{}
				vgObjKey := types.NamespacedName{
					Name:      vgName,
					Namespace: namespace,
				}
				//time.Sleep(15 * time.Second)

				By("Getting VolumeGroupContent object after creation")
				Eventually(func() (*volumegroupv1.VolumeGroupContent, error) {
					if err := k8sClient.Get(context.Background(), vgObjKey, vgObj); err != nil {
						return nil, err
					}
					key := types.NamespacedName{
						Name:      fmt.Sprintf("volumegroup-%s", vgObj.UID),
						Namespace: namespace,
					}
					found := &volumegroupv1.VolumeGroupContent{}
					err = k8sClient.Get(context.Background(), key, found)
					return found, err
				}, timeout, interval).ShouldNot(BeNil())

				close(done)
			}, timeout.Seconds())
		})
	})
})
