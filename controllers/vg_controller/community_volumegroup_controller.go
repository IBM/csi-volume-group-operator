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

package vgcontroller

import (
	"context"

	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/IBM/csi-volume-group-operator/apis/abstract"
	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/volumegroup.storage/v1"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CommunityVolumeGroupReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	DriverConfig *config.DriverConfig
	GRPCClient   *grpcClient.Client
	VGClient     grpcClient.VolumeGroup
}

//+kubebuilder:rbac:groups=volumegroup.storage.openshift.io,resources=volumegroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=volumegroup.storage.openshift.io,resources=volumegroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=volumegroup.storage.openshift.io,resources=volumegroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=volumegroup.storage.openshift.io,resources=volumegroupclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=volumegroup.storage.openshift.io,resources=volumegroupcontents,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims/finalizers,verbs=update

func (r *CommunityVolumeGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Name", req.Name, "Request.Namespace", req.Namespace)
	logger.Info(messages.ReconcileVG)
	vgObjects := abstract.VolumeGroupObjects{
		VG:          &volumegroupv1.VolumeGroup{},
		VGList:      &volumegroupv1.VolumeGroupList{},
		VGC:         &volumegroupv1.VolumeGroupContent{},
		VGCList:     &volumegroupv1.VolumeGroupContentList{},
		VGClass:     &volumegroupv1.VolumeGroupClass{},
		VGCLassList: &volumegroupv1.VolumeGroupClassList{},
	}
	reconciler := VolumeGroupReconciler{
		Client:       r.Client,
		Log:          r.Log,
		Scheme:       r.Scheme,
		DriverConfig: r.DriverConfig,
		GRPCClient:   r.GRPCClient,
		VGClient:     r.VGClient,
		VGObjects:    vgObjects,
	}
	return reconciler.Reconcile(ctx, req)
}

func (r *CommunityVolumeGroupReconciler) SetupWithManager(mgr ctrl.Manager, cfg *config.DriverConfig) error {
	logger := r.Log.WithName("SetupWithManager")
	err := waitForCRDs(logger, r.Client, volumegroupv1.GroupVersion.Group, volumegroupv1.GroupVersion.Version)
	if err != nil {
		r.Log.Error(err, "failed to wait for crds")

		return err
	}
	generationPred := predicate.GenerationChangedPredicate{}
	pred := predicate.Or(generationPred, utils.FinalizerPredicate)

	r.VGClient = grpcClient.NewVolumeGroupClient(r.GRPCClient.Client, cfg.RPCTimeout)

	return ctrl.NewControllerManagedBy(mgr).
		For(&volumegroupv1.VolumeGroup{}, builder.WithPredicates(pred)).
		Watches(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, utils.CreateRequests(r.Client, &volumegroupv1.VolumeGroupList{}),
			builder.WithPredicates(utils.PvcPredicate)).Complete(r)
}
