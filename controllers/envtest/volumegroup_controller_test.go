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
	"time"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/envtest/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	Describe("Test controllers", func() {

		Context("Test VG controller", func() {

			BeforeEach(func() {
				err := cleanTestNamespace()
				Expect(err).ToNot(HaveOccurred())
			})

			It("Should create all the relevant objects", func(done Done) {
				By("Creating volumeGroup objects")
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeGroupObjects()
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)

				vgObj := &volumegroupv1.VolumeGroup{}

				By("Validating VolumeGroup object created")
				err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())

				By("Validating VolumeGroupContent object created")
				_, err = utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())

				close(done)
			}, Timeout.Seconds())
			It("should add and remove volume objects from volumeGroup objects when created before vg", func(done Done) {
				By("Creating volume objects before volumeGroup objects")
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = utils.CreateResourceObject(StorageClass, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeObjects()
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeGroupObjects()
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)

				vgObj := &volumegroupv1.VolumeGroup{}

				By("Validating that PVC is in VG")
				err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(vgObj.Status.PVCList)).To(Equal(1))
				Expect(vgObj.Status.PVCList[0].Name).To(Equal(PVCName))
				Expect(vgObj.Status.PVCList[0].Namespace).To(Equal(Namespace))

				By("Validating that PV is in VGC")
				vgcObj, err := utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(vgcObj.Status.PVList)).To(Equal(1))
				Expect(vgcObj.Status.PVList[0].Name).To(Equal(PVName))

				By("Removing labels from VG")
				err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				vgObj.Spec.Source.Selector.MatchLabels = map[string]string{
					"bad-key": "bad-value",
				}
				err = k8sClient.Update(context.TODO(), vgObj)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)

				By("Validating that PVC and PV are not in VG and VGC")
				err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				vgcObj, err = utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(vgObj.Status.PVCList)).To(Equal(0))
				Expect(len(vgcObj.Status.PVList)).To(Equal(0))

				close(done)
			}, Timeout.Seconds())
		})
	})
})
