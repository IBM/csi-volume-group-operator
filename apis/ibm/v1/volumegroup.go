package v1

import (
	vgabstract "github.com/IBM/csi-volume-group-operator/apis/abstract"
	"github.com/IBM/csi-volume-group-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (vg *VolumeGroup) GetVGCName() string {
	return utils.GetStringField(vg.Spec.Source, "VolumeGroupContentName")
}
func (vg *VolumeGroup) GetVGCLassName() string {
	return utils.GetStringField(vg.Spec, "VolumeGroupClassName")
}
func (vg *VolumeGroup) GetSpec() vgabstract.VolumeGroupSpec     { return vg.Spec }
func (vg *VolumeGroup) GetSource() vgabstract.VolumeGroupSource { return vg.Spec.Source }
func (vg *VolumeGroup) GetSelector() *metav1.LabelSelector      { return vg.Spec.Source.Selector }
func (vg *VolumeGroup) IsReady() bool                           { return *vg.Status.Ready }
