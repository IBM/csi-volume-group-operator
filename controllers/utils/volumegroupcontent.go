package utils

import (
	"context"
	"fmt"

	volumegroupv1 "github.com/IBM/csi-volume-group-operator/api/v1"
	"github.com/IBM/csi-volume-group-operator/controllers/volumegroup"
	"github.com/IBM/csi-volume-group-operator/pkg/messages"
	csi "github.com/IBM/csi-volume-group/lib/go/volumegroup"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddMatchingPVToMatchingVGC(logger logr.Logger, client client.Client,
	pvc *corev1.PersistentVolumeClaim, vg *volumegroupv1.VolumeGroup) error {
	pv, err := GetPVFromPVC(logger, client, pvc)
	if err != nil {
		return err
	}
	vgc, err := GetVolumeGroupContent(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return err
	}

	if pv != nil {
		return addPVToVGC(logger, client, pv, vgc)
	}
	return nil
}

func GetVolumeGroupContent(client client.Client, logger logr.Logger,
	volumeGroupContentName string, vgName string, vgNamespace string) (*volumegroupv1.VolumeGroupContent, error) {
	logger.Info(fmt.Sprintf(messages.GetVolumeGroupContentOfVolumeGroup, vgName, vgNamespace))
	vgc := &volumegroupv1.VolumeGroupContent{}
	namespacedVGC := types.NamespacedName{Name: volumeGroupContentName, Namespace: vgNamespace}
	err := client.Get(context.TODO(), namespacedVGC, vgc)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "VolumeGroupContent not found", "VolumeGroupContent Name", volumeGroupContentName)
		}
		return nil, err
	}

	return vgc, nil
}

func CreateVolumeGroupContent(client client.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent) error {
	err := client.Create(context.TODO(), vgc)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logger.Info("VolumeGroupContent is already exists")
			return nil
		}
		logger.Error(err, "VolumeGroupContent creation failed", "VolumeGroupContent Name")
		return err
	}
	err = createSuccessVolumeGroupContentEvent(logger, client, vgc)
	return err
}

func createSuccessVolumeGroupContentEvent(logger logr.Logger, client client.Client, vgc *volumegroupv1.VolumeGroupContent) error {
	vgc.APIVersion = APIVersion
	vgc.Kind = volumeGroupContentKind
	message := fmt.Sprintf(messages.VolumeGroupContentCreated, vgc.Namespace, vgc.Name)
	err := createSuccessNamespacedObjectEvent(logger, client, vgc, message, createVGC)
	if err != nil {
		return nil
	}
	return nil
}

func UpdateVolumeGroupContentStatus(client client.Client, logger logr.Logger, vgc *volumegroupv1.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) error {
	updateVolumeGroupContentStatusFields(vgc, groupCreationTime, ready)
	if err := UpdateObjectStatus(client, vgc); err != nil {
		logger.Error(err, "failed to update status")
		return err
	}
	return nil
}

func updateVolumeGroupContentStatusFields(vgc *volumegroupv1.VolumeGroupContent, groupCreationTime *metav1.Time, ready bool) {
	vgc.Status.GroupCreationTime = groupCreationTime
	vgc.Status.Ready = &ready
}

func GenerateVolumeGroupContent(vgname string, instance *volumegroupv1.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass, resp *volumegroup.Response, secretName string, secretNamespace string) *volumegroupv1.VolumeGroupContent {
	return &volumegroupv1.VolumeGroupContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vgname,
			Namespace: instance.Namespace,
		},
		Spec: generateVolumeGroupContentSpec(instance, vgClass, resp, secretName, secretNamespace),
	}
}

func generateVolumeGroupContentSpec(instance *volumegroupv1.VolumeGroup, vgClass *volumegroupv1.VolumeGroupClass,
	resp *volumegroup.Response, secretName string, secretNamespace string) volumegroupv1.VolumeGroupContentSpec {
	return volumegroupv1.VolumeGroupContentSpec{
		VolumeGroupClassName: instance.Spec.VolumeGroupClassName,
		VolumeGroupRef:       generateObjectReference(instance),
		Source:               generateVolumeGroupContentSource(vgClass, resp),
		VolumeGroupSecretRef: generateSecretReference(secretName, secretNamespace),
	}
}

func generateObjectReference(instance *volumegroupv1.VolumeGroup) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:            instance.Kind,
		Namespace:       instance.Namespace,
		Name:            instance.Name,
		UID:             instance.UID,
		APIVersion:      instance.APIVersion,
		ResourceVersion: instance.ResourceVersion,
	}
}

func generateSecretReference(secretName string, secretNamespace string) *corev1.SecretReference {
	return &corev1.SecretReference{
		Name:      secretName,
		Namespace: secretNamespace,
	}
}

func generateVolumeGroupContentSource(vgClass *volumegroupv1.VolumeGroupClass, resp *volumegroup.Response) *volumegroupv1.VolumeGroupContentSource {
	CreateVolumeGroupResponse := resp.Response.(*csi.CreateVolumeGroupResponse)
	return &volumegroupv1.VolumeGroupContentSource{
		Driver:                vgClass.Driver,
		VolumeGroupHandle:     CreateVolumeGroupResponse.VolumeGroup.VolumeGroupId,
		VolumeGroupAttributes: CreateVolumeGroupResponse.VolumeGroup.VolumeGroupContext,
	}
}

func RemovePVFromVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume, vgc *volumegroupv1.VolumeGroupContent) error {
	logger.Info(fmt.Sprintf(messages.RemovePersistentVolumeFromVolumeGroupContent,
		pv.Namespace, pv.Name, vgc.Namespace, vgc.Name))
	vgc.Status.PVList = removeFromPVList(pv, vgc.Status.PVList)
	err := updateVolumeGroupContentStatusPVList(client, vgc, logger, vgc.Status.PVList)
	if err != nil {
		vgc.Status.PVList = appendPersistentVolume(vgc.Status.PVList, *pv)
		logger.Error(err, fmt.Sprintf(messages.FailedToRemovePersistentVolumeFromVolumeGroupContent,
			pv.Name, vgc.Namespace, vgc.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.RemovedPersistentVolumeFromVolumeGroupContent,
		pv.Name, vgc.Namespace, vgc.Name))
	return nil
}

func removeFromPVList(pv *corev1.PersistentVolume, pvList []corev1.PersistentVolume) []corev1.PersistentVolume {
	for index, pvFromList := range pvList {
		if pvFromList.Name == pv.Name && pvFromList.Namespace == pv.Namespace {
			pvList = removeByIndexFromPersistentVolumeList(pvList, index)
			return pvList
		}
	}
	return pvList
}

func addPVToVGC(logger logr.Logger, client client.Client, pv *corev1.PersistentVolume,
	vgc *volumegroupv1.VolumeGroupContent) error {
	logger.Info(fmt.Sprintf(messages.AddPersistentVolumeToVolumeGroupContent,
		pv.Name, vgc.Namespace, vgc.Name))
	vgc.Status.PVList = appendPersistentVolume(vgc.Status.PVList, *pv)
	err := updateVolumeGroupContentStatusPVList(client, vgc, logger, vgc.Status.PVList)
	if err != nil {
		vgc.Status.PVList = removeFromPVList(pv, vgc.Status.PVList)
		logger.Error(err, fmt.Sprintf(messages.FailedToAddPersistentVolumeToVolumeGroupContent,
			pv.Name, vgc.Namespace, vgc.Name))
		return err
	}
	logger.Info(fmt.Sprintf(messages.AddedPersistentVolumeToVolumeGroupContent,
		pv.Name, vgc.Namespace, vgc.Name))
	return nil
}

func appendPersistentVolume(pvListInVGC []corev1.PersistentVolume, pv corev1.PersistentVolume) []corev1.PersistentVolume {
	for _, pvFromList := range pvListInVGC {
		if pvFromList.Name == pv.Name {
			return pvListInVGC
		}
	}
	pvListInVGC = append(pvListInVGC, pv)
	return pvListInVGC
}

func updateVolumeGroupContentStatusPVList(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger,
	pvList []corev1.PersistentVolume) error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		vgc.Status.PVList = pvList
		err := vgcRetryOnConflictFunc(client, vgc, logger)
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func vgcRetryOnConflictFunc(client client.Client, vgc *volumegroupv1.VolumeGroupContent, logger logr.Logger) error {
	err := UpdateObjectStatus(client, vgc)
	if apierrors.IsConflict(err) {
		uErr := getNamespacedObject(client, vgc)
		if uErr != nil {
			return uErr
		}
		logger.Info(fmt.Sprintf(messages.RetryUpdateVolumeGroupStatus, vgc.Namespace, vgc.Name))
	}
	return err
}

func UpdateStaticVGC(client client.Client, vg *volumegroupv1.VolumeGroup,
	vgClass *volumegroupv1.VolumeGroupClass, logger logr.Logger) error {
	vgc, err := GetVolumeGroupContent(client, logger, *vg.Spec.Source.VolumeGroupContentName, vg.Name, vg.Namespace)
	if err != nil {
		return err
	}
	updateStaticVGCSpec(vgClass, vgc, vg)
	if err = UpdateObject(client, vgc); err != nil {
		return err
	}
	return nil
}

func updateStaticVGCSpec(vgClass *volumegroupv1.VolumeGroupClass, vgc *volumegroupv1.VolumeGroupContent, vg *volumegroupv1.VolumeGroup) {
	secretName, secretNamespace := GetSecretCred(vgClass)
	vgc.Spec.VolumeGroupClassName = vg.Spec.VolumeGroupClassName
	vgc.Spec.VolumeGroupRef = generateObjectReference(vg)
	vgc.Spec.Source.Driver = vgClass.Driver
	vgc.Spec.VolumeGroupSecretRef = generateSecretReference(secretName, secretNamespace)
}
