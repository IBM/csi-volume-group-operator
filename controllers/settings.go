package controllers

import (
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
)
