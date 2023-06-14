# CSI Volume Group Operator

## Overview

Volume Group Operator is based on [IBM/csi-volume-group spec](https://github.com/IBM/csi-volume-group) specification and can be used by any storage
provider.

## Design

Volume Group Operator follows controller pattern and provides extended APIs for volume grouping.
The extended APIs are provided via Custom Resource Definition (CRD).

### [VolumeGroupClass](https://github.com/IBM/csi-volume-group-operator/blob/develop/config/crd/bases/volumegroup.storage.openshift.io_volumegroupclasses.yaml)

`VolumeGroupClass` is a cluster scoped resource that contains driver related configuration parameters.

`driver` is name of the storage provisioner

`parameters` contains key-value pairs that are passed down to the driver. Users can add their own key-value pairs.
Keys with `volumegroup.storage.openshift.io/` prefix are reserved by operator and not passed down to the driver.

#### Reserved parameter keys

- `volumegroup.storage.openshift.io/secret-name`
- `volumegroup.storage.openshift.io/secret-namespace`

```yaml
apiVersion: volumegroup.storage.openshift.io/v1
kind: VolumeGroupClass
metadata:
  name: volume-group-class-sample
spec:
  driver: example.provisioner.io
  parameters:
    volumegroup.storage.openshift.io/secret-name: demo-secret
    volumegroup.storage.openshift.io/secret-namespace: default
```

### [VolumeGroupContent](https://github.com/IBM/csi-volume-group-operator/blob/develop/config/crd/bases/volumegroup.storage.openshift.io_volumegroupcontents.yaml)

VolumeGroupContent is a namespaced resource that contains references to storage objects that are part of a group.

`driver` is name of the storage provisioner

`volumeGroupHandle` is the unique identifier for the group

`VolumeGroupDeletionPolicy` is the deletion policy for the group. Possible values are `Delete` and `Retain`.

`VolumeGroupClassName` is the name of the `VolumeGroupClass` that contains driver related configuration parameters.

```yaml
apiVersion: volumegroup.storage.openshift.io/v1
kind: VolumeGroupContent
metadata:
  name: volume-group-content-sample
  namespace: default
spec:
  source:
    driver: example.provisioner.io
    volumeGroupHandle: volume-group-handle-sample
  VolumeGroupDeletionPolicy: Delete
  VolumeGroupClassName: volume-group-class-sample
```

### [VolumeGroup](https://github.com/IBM/csi-volume-group-operator/blob/develop/config/crd/bases/volumegroup.storage.openshift.io_volumegroups.yaml)

VolumeGroup is a namespaced resource that contains references to storage objects that are part of a group.

`driver` is name of the storage provisioner

`selector` is a label selector that is used to select `PVC` objects that are part of the group.

`VolumeGroupClassName` is the name of the `VolumeGroupClass` that contains driver related configuration parameters.

```yaml
apiVersion: volumegroup.storage.openshift.io/v1
kind: VolumeGroup
metadata:
  name: volume-group-sample
  namespace: default
spec:
  VolumeGroupClassName: volume-group-class-sample
  source:
    driver: example.provisioner.io
    selector:
      matchLabels:
        volume-group-key: volume-group-value
```

## VolumeGroup controller command line options

### Important optional arguments that are highly recommended to be used

- `--driver-name` - Name of the CSI driver.
- `--csi-address` - Address of the CSI driver socket. Default is /run/csi/socket
- `--rpc-timeout` - Timeout for CSI driver RPCs. Default is 60s.
- `--multiple-vgs-to-pvc` - Allow multiple volume groups to be attached to a single PVC. Default is true.
- `--disable-delete-pvcs` - Disable deletion of PVCs when volume group is deleted. Default is false.
