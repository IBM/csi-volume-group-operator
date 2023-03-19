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
	"context"
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Controller", func() {

	BeforeEach(func() {
	})

	Describe("test vg controller", func() {

		Context("create an vg instance", func() {

			It("should create all the relevant objects", func(done Done) {
				err := k8sClient.Create(context.Background(), VGClass)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), Secret)
				Expect(err).NotTo(HaveOccurred())
				err = k8sClient.Create(context.Background(), VG)
				Expect(err).NotTo(HaveOccurred())

				vgObj := &volumegroupv1.VolumeGroup{}
				vgObjKey := types.NamespacedName{
					Name:      VGName,
					Namespace: Namespace,
				}

				By("Getting VolumeGroupContent object after creation")
				Eventually(func() (*volumegroupv1.VolumeGroupContent, error) {
					if err := k8sClient.Get(context.Background(), vgObjKey, vgObj); err != nil {
						return nil, err
					}
					key := types.NamespacedName{
						Name:      fmt.Sprintf("volumegroup-%s", vgObj.UID),
						Namespace: Namespace,
					}
					found := &volumegroupv1.VolumeGroupContent{}
					err = k8sClient.Get(context.Background(), key, found)
					return found, err
				}, Timeout, Interval).ShouldNot(BeNil())

				//By("Checking if PVC and PV is in VG and VGC")
				//Eventually(func() (*volumegroupv1.VolumeGroupContent, error) {
				//	if err := k8sClient.Get(context.Background(), vgObjKey, vgObj); err != nil {
				//		return nil, err
				//	}
				//	key := types.NamespacedName{
				//		Name:      fmt.Sprintf("volumegroup-%s", vgObj.UID),
				//		Namespace: Namespace,
				//	}
				//	found := &volumegroupv1.VolumeGroupContent{}
				//	err = k8sClient.Get(context.Background(), key, found)
				//	return found, err
				//}, Timeout, Interval).ShouldNot(BeNil())

				close(done)
			}, Timeout.Seconds())
		})
	})
})
