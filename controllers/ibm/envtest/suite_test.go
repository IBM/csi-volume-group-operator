/*
Copyright 2022.

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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
	ibmcontroller "github.com/IBM/csi-volume-group-operator/controllers/ibm"
	"github.com/IBM/csi-volume-group-operator/controllers/ibm/envtest/utils"
	ibmvgccontroller "github.com/IBM/csi-volume-group-operator/controllers/ibm/volumegroupcontent"
	"github.com/IBM/csi-volume-group-operator/pkg/client/fake"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/tests/mock_grpc_server"
	//+kubebuilder:scaffold:imports
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	cancel    context.CancelFunc
	ctx       context.Context
	server    *mock_grpc_server.MockServer
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = volumegroupv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	server, err = mock_grpc_server.CreateMockServer()
	Expect(err).ToNot(HaveOccurred())
	addr := server.Address()
	csiConn, err := fake.New(addr, DriverName)
	Expect(err).ToNot(HaveOccurred())
	driverConfig := &config.DriverConfig{
		DriverName:        DriverName,
		DriverEndpoint:    addr,
		RPCTimeout:        time.Minute,
		MultipleVGsToPVC:  "false",
		DisableDeletePvcs: "false",
	}
	mockVolumeGroup := fake.VolumeGroup{
		CreateVolumeGroupMock: func(name string, secrets, parameters map[string]string) (*csi.CreateVolumeGroupResponse, error) {
			return &csi.CreateVolumeGroupResponse{}, nil
		},
		DeleteVolumeGroupMock: func(volumeGroupId string, secrets map[string]string) (*csi.DeleteVolumeGroupResponse, error) {
			return &csi.DeleteVolumeGroupResponse{}, nil
		},
		ModifyVolumeGroupMembershipMock: func(volumeGroupId string, volumeIds []string, secrets map[string]string) (*csi.ModifyVolumeGroupMembershipResponse, error) {
			return &csi.ModifyVolumeGroupMembershipResponse{}, nil
		},
	}

	err = (&ibmcontroller.VolumeGroupReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		DriverConfig: driverConfig,
		Log:          ctrl.Log.WithName("controllers").WithName("VolumeGroup"),
		GRPCClient:   csiConn,
		VGClient:     mockVolumeGroup,
	}).SetupWithManager(mgr, driverConfig)
	Expect(err).ToNot(HaveOccurred())

	err = (&ibmvgccontroller.VolumeGroupContentReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		DriverConfig: driverConfig,
		Log:          ctrl.Log.WithName("VolumeGroupContentController"),
		GRPCClient:   csiConn,
		VGClient:     mockVolumeGroup,
	}).SetupWithManager(mgr, driverConfig)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	server.Stop()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createNonVolumeK8SResources() error {
	err := utils.CreateResourceObject(Secret, k8sClient)
	if err != nil {
		return err
	}
	return utils.CreateResourceObject(StorageClass, k8sClient)
}

func createVolumeGroupObjects(deletionPolicy volumegroupv1.VolumeGroupDeletionPolicy) error {
	err := utils.CreateResourceObject(VGClass, k8sClient)
	if err != nil {
		return err
	}

	vgclass := &volumegroupv1.VolumeGroupClass{}
	err = utils.GetNamespacedResourceObject(VGClassName, Namespace, vgclass, k8sClient)
	if err != nil {
		return err
	}
	vgclass.VolumeGroupDeletionPolicy = &deletionPolicy
	err = k8sClient.Update(context.TODO(), vgclass)
	if err != nil {
		return err
	}

	err = utils.CreateResourceObject(VG, k8sClient)
	return err
}

func createVolumeObjects() error {
	if err := utils.CreateResourceObject(PV, k8sClient); err != nil {
		return err
	}
	if err := utils.CreateResourceObject(PVC, k8sClient); err != nil {
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{}
	if err := utils.GetNamespacedResourceObject(PVCName, Namespace, pvc, k8sClient); err != nil {
		return err
	}
	pvc.Status.Phase = corev1.ClaimBound
	err := k8sClient.Status().Update(context.TODO(), pvc)
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
