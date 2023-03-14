package controllers

import (
	"context"
	"time"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Controller", func() {

	const timeout = time.Second * 30
	const interval = time.Second * 1
	var vg *volumegroupv1.VolumeGroup

	BeforeEach(func() {
		labels := make(map[string]string)
		labels["test"] = "matan"
		vg = &volumegroupv1.VolumeGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vg-name",
				Namespace: "default",
			},
			Spec: volumegroupv1.VolumeGroupSpec{
				Source: volumegroupv1.VolumeGroupSource{
					Selector: &metav1.LabelSelector{
						MatchLabels: labels,
					},
				},
			},
		}
	})

	Describe("test ibc controller", func() {

		Context("create an ibc instance", func() {

			It("should create all the relevant objects", func(done Done) {
				err := k8sClient.Create(context.Background(), vg)
				Expect(err).NotTo(HaveOccurred())

				found := &volumegroupv1.VolumeGroup{}
				key := types.NamespacedName{
					Name:      "vg-name",
					Namespace: "default",
				}

				By("Getting VolumeGroup object after creation")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					err := k8sClient.Get(context.Background(), key, found)
					return found, err
				}, timeout, interval).ShouldNot(BeNil())

				close(done)
			}, timeout.Seconds())
		})
	})
})
