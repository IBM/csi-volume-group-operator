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

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/envtest/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Controller", func() {
	Describe("Test controllers", func() {
		Context("Test PVC controller", func() {

			BeforeEach(func() {
				err := cleanTestNamespace()
				Expect(err).ToNot(HaveOccurred())
			})
			It("Should add and remove volume objects from volumeGroup objects when created after vg", func(done Done) {
				By("Creating volumeGroup objects before VolumeObjects")
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = utils.CreateResourceObject(StorageClass, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeGroupObjects()
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeObjects()
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(1 * time.Second)

				vgObj := &volumegroupv1.VolumeGroup{}
				pvc := &corev1.PersistentVolumeClaim{}

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

				By("Removing labels from PVC")
				err = utils.GetNamespacedResourceObject(PVCName, Namespace, pvc, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				pvc.ObjectMeta.Labels = map[string]string{}
				err = k8sClient.Status().Update(context.TODO(), pvc)
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
