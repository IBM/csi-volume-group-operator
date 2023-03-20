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
	"time"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/tests/envtest/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Controller", func() {
	Describe("test controllers", func() {

		Context("test vg controller", func() {

			BeforeEach(func() {
				err := cleanTestNamespace()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should create all the relevant objects", func(done Done) {
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeGroupObjects()
				Expect(err).NotTo(HaveOccurred())

				vgObj := &volumegroupv1.VolumeGroup{}

				By("Getting VolumeGroup object after creation")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					return vgObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				By("Getting VolumeGroupContent object after creation")
				Eventually(func() (*volumegroupv1.VolumeGroupContent, error) {
					return utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
				}, Timeout, Interval).ShouldNot(BeNil())

				close(done)
			}, Timeout.Seconds())
			It("should add and remove volume objects from volumeGroup objects when created before vg", func(done Done) {
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

				By("Checking if PVC is in VG")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					Expect(len(vgObj.Status.PVCList)).To(Equal(1))
					Expect(vgObj.Status.PVCList[0].Name).To(Equal(PVCName))
					Expect(vgObj.Status.PVCList[0].Namespace).To(Equal(Namespace))
					return vgObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				By("Checking if PV is in VGC")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					vgcObj, err := utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return nil, err
					}
					Expect(len(vgcObj.Status.PVList)).To(Equal(1))
					Expect(vgcObj.Status.PVList[0].Name).To(Equal(PVName))
					return vgObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				By("Checking if PVC and PV are not in VG and VGC")
				Eventually(func() error {
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return err
					}
					vgObj.Spec.Source.Selector.MatchLabels = map[string]string{
						"bad-key": "bad-value",
					}
					k8sClient.Update(context.TODO(), vgObj)
					time.Sleep(1 * time.Second)
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return err
					}
					vgcObj, err := utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return err
					}
					Expect(len(vgObj.Status.PVCList)).To(Equal(0))
					Expect(len(vgcObj.Status.PVList)).To(Equal(0))
					return err
				}, Timeout, Interval).Should(BeNil())

				close(done)
			}, Timeout.Seconds())
		})
		Context("test pvc controllers", func() {

			BeforeEach(func() {
				err := cleanTestNamespace()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should add and remove volume objects from volumeGroup objects when created after vg", func(done Done) {
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

				By("Checking if PVC is in VG")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					Expect(len(vgObj.Status.PVCList)).To(Equal(1))
					Expect(vgObj.Status.PVCList[0].Name).To(Equal(PVCName))
					Expect(vgObj.Status.PVCList[0].Namespace).To(Equal(Namespace))
					return vgObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				By("Checking if PV is in VGC")
				Eventually(func() (*volumegroupv1.VolumeGroup, error) {
					vgcObj, err := utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return nil, err
					}
					Expect(len(vgcObj.Status.PVList)).To(Equal(1))
					Expect(vgcObj.Status.PVList[0].Name).To(Equal(PVName))
					return vgObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				By("Checking if PVC and PV are not in VG and VGC")
				Eventually(func() error {
					if err := utils.GetNamespacedResourceObject(PVCName, Namespace, pvc, k8sClient); err != nil {
						return err
					}
					pvc.ObjectMeta.Labels = map[string]string{}
					if err := k8sClient.Status().Update(context.TODO(), pvc); err != nil {
						return err
					}
					time.Sleep(1 * time.Second)
					err = utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return err
					}
					vgcObj, err := utils.GetVGCObjectFromVG(VGName, Namespace, vgObj, k8sClient)
					if err != nil {
						return err
					}
					Expect(len(vgObj.Status.PVCList)).To(Equal(0))
					Expect(len(vgcObj.Status.PVList)).To(Equal(0))
					return err
				}, Timeout, Interval).Should(BeNil())

				close(done)
			}, Timeout.Seconds())
		})
		Context("test vgc controllers", func() {

			BeforeEach(func() {
				err := cleanTestNamespace()
				Expect(err).ToNot(HaveOccurred())
			})
			It("should not delete vgc when vgclass deletion policy is retain", func(done Done) {
				retainDeletionPolicy := volumegroupv1.VolumeGroupContentRetain
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())

				err = utils.CreateResourceObject(VGClass, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				vgclass := &volumegroupv1.VolumeGroupClass{}
				err = utils.GetNamespacedResourceObject(VGClassName, Namespace, vgclass, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				VGClass.VolumeGroupDeletionPolicy = &retainDeletionPolicy
				err = k8sClient.Update(context.TODO(), vgclass)
				Expect(err).NotTo(HaveOccurred())

				err = utils.CreateResourceObject(VG, k8sClient)
				Expect(err).NotTo(HaveOccurred())

				vgObj := &volumegroupv1.VolumeGroup{}

				By("Deleting VG")
				Eventually(func() (*volumegroupv1.VolumeGroupContent, error) {
					if err := utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient); err != nil {
						return nil, err
					}
					if err := k8sClient.Delete(context.TODO(), vgObj); err != nil {
						return nil, err
					}
					time.Sleep(1 * time.Second)

					vgcName := utils.GetVGCName(vgObj.GetUID())
					vgcObj := &volumegroupv1.VolumeGroupContent{}
					err := utils.GetNamespacedResourceObject(vgcName, Namespace, vgcObj, k8sClient)
					vgErr := utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					Expect(apierrors.IsNotFound(vgErr)).To(BeTrue())
					return vgcObj, err
				}, Timeout, Interval).ShouldNot(BeNil())

				close(done)
			}, Timeout.Seconds())
			It("should delete pvcs when deleting vg", func(done Done) {
				err := utils.CreateResourceObject(Secret, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = utils.CreateResourceObject(StorageClass, k8sClient)
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeGroupObjects()
				Expect(err).NotTo(HaveOccurred())
				err = createVolumeObjects()
				Expect(err).NotTo(HaveOccurred())

				vgObj := &volumegroupv1.VolumeGroup{}

				By("Deleting VG")
				Eventually(func() error {
					if err := utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient); err != nil {
						return err
					}
					if err := k8sClient.Delete(context.TODO(), vgObj); err != nil {
						return err
					}
					time.Sleep(3 * time.Second)

					vgcName := utils.GetVGCName(vgObj.GetUID())
					vgcObj := &volumegroupv1.VolumeGroupContent{}
					vgcErr := utils.GetNamespacedResourceObject(vgcName, Namespace, vgcObj, k8sClient)
					if vgcErr != nil && !apierrors.IsNotFound(vgcErr) {
						return vgcErr
					}
					fmt.Printf("matan80 %v\n", vgcObj.ObjectMeta.Finalizers)
					Expect(apierrors.IsNotFound(vgcErr)).To(BeTrue())
					vgErr := utils.GetNamespacedResourceObject(VGName, Namespace, vgObj, k8sClient)
					if vgErr != nil && !apierrors.IsNotFound(vgErr) {
						return vgErr
					}
					Expect(apierrors.IsNotFound(vgErr)).To(BeTrue())
					return nil
				}, Timeout, Interval).Should(BeNil())

				//By("Checking that has been PVC deleted")
				//Eventually(func() error {
				//	pvcObj := &corev1.PersistentVolumeClaim{}
				//	pvcErr := utils.GetNamespacedResourceObject(PVCName, Namespace, pvcObj, k8sClient)
				//	if pvcErr != nil && !apierrors.IsNotFound(pvcErr) {
				//		return pvcErr
				//	}
				//	Expect(apierrors.IsNotFound(pvcErr)).To(BeTrue())
				//	return nil
				//}, Timeout, Interval).Should(BeNil())

				close(done)
			}, Timeout.Seconds())
		})
	})
})

func createVolumeGroupObjects() error {
	err := utils.CreateResourceObject(VGClass, k8sClient)
	if err != nil {
		return err
	}
	err = utils.CreateResourceObject(VG, k8sClient)
	return err
}

func createVolumeObjects() error {
	err := utils.CreateResourceObject(PV, k8sClient)
	if err != nil {
		return err
	}
	err = utils.CreateResourceObject(PVC, k8sClient)
	if err != nil {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{}
	err = utils.GetNamespacedResourceObject(PVCName, Namespace, pvc, k8sClient)
	if err != nil {
		return err
	}
	pvc.Status.Phase = corev1.ClaimBound
	err = k8sClient.Status().Update(context.TODO(), pvc)
	return err
}

func cleanTestNamespace() error {
	err := cleanVolumeGroupObjects()
	if err != nil {
		return err
	}
	err = cleanVolumeObjects()
	if err != nil {
		return err
	}
	err = k8sClient.DeleteAllOf(context.Background(), &corev1.Secret{}, client.InNamespace(Namespace))
	if err != nil {
		return err
	}
	err = k8sClient.DeleteAllOf(context.Background(), &storagev1.StorageClass{})
	return err
}

func cleanVolumeGroupObjects() error {
	err := k8sClient.DeleteAllOf(context.Background(), &volumegroupv1.VolumeGroup{}, client.InNamespace(Namespace))
	if err != nil {
		return err
	}
	err = k8sClient.DeleteAllOf(context.Background(), &volumegroupv1.VolumeGroupContent{}, client.InNamespace(Namespace))
	if err != nil {
		return err
	}
	err = k8sClient.DeleteAllOf(context.Background(), &volumegroupv1.VolumeGroupClass{}, client.InNamespace(Namespace))
	return err
}

func cleanVolumeObjects() error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := utils.RemoveResourceObjectFinalizers(PVCName, Namespace, pvc, k8sClient); err != nil {
		return err
	}
	pv := &corev1.PersistentVolume{}
	if err := utils.RemoveResourceObjectFinalizers(PVName, Namespace, pv, k8sClient); err != nil {
		return err
	}
	err := k8sClient.DeleteAllOf(context.Background(), &corev1.PersistentVolumeClaim{}, client.InNamespace(Namespace))
	if err != nil {
		return err
	}
	err = k8sClient.DeleteAllOf(context.Background(), &corev1.PersistentVolume{})
	return err
}
