package controllers

import (
	"fmt"

	"github.com/IBM/csi-volume-group-operator/controllers/utils"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	log             = logf.Log.WithName("predicate")
	addingPVC       = "addPVC"
	removingPVC     = "removePVC"
	vgReconcile     = "vgReconcile"
	deleteVG        = "deletingVG"
	createVG        = "creatingVG"
	createVGC       = "creatingVGC"
	updateVGC       = "updatingVGC"
	updateStatusVG  = "updatingStatusVG"
	updateStatusVGC = "updatingStatusVGC"
	vgPredicate     = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			log.Info("matan create")
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.Info("matan delete")
			log.Info(fmt.Sprintf("matan delete: %v", !e.DeleteStateUnknown))
			return !e.DeleteStateUnknown
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.Info("matan update")
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() || utils.IsVGFinalizersChanged(e.ObjectOld, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			log.Info("matan Generic")
			return true
		},
	}
)
