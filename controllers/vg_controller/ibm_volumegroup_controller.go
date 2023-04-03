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

	"github.com/IBM/csi-volume-group-operator/pkg/config"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/apis/ibm/v1"
	grpcClient "github.com/IBM/csi-volume-group-operator/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IBMVolumeGroupReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	DriverConfig *config.DriverConfig
	GRPCClient   *grpcClient.Client
	VGClient     grpcClient.VolumeGroup
}

//+kubebuilder:rbac:groups=csi.ibm.com,resources=volumegroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=csi.ibm.com,resources=volumegroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=csi.ibm.com,resources=volumegroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=csi.ibm.com,resources=volumegroupclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=csi.ibm.com,resources=volumegroupcontents,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims/finalizers,verbs=update

func (r *IBMVolumeGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("Request.Name", req.Name, "Request.Namespace", req.Namespace)
	logger.Info(messages.ReconcileVG)
	return ctrl.Result{}, nil
}

func (r *IBMVolumeGroupReconciler) SetupWithManager(mgr ctrl.Manager, cfg *config.DriverConfig) error {
	logger := r.Log.WithName("SetupWithManager")
	err := waitForCrds(logger, r.Client, volumegroupv1.GroupVersion.Group, volumegroupv1.GroupVersion.Version)
	if err != nil {
		r.Log.Error(err, "failed to wait for crds")

		return err
	}
	pred := predicate.GenerationChangedPredicate{}

	r.VGClient = grpcClient.NewVolumeGroupClient(r.GRPCClient.Client, cfg.RPCTimeout)

	return ctrl.NewControllerManagedBy(mgr).
		For(&volumegroupv1.VolumeGroup{}).
		WithEventFilter(pred).Complete(r)
}
