# CreateVolume Parameters
There are several optional parameters that may be passed into `CreateVolumeRequest.parameters` map, these parameters can be configured in StorageClass, see [example](../example/ebs/csi-storageclass.yaml).

| Parameters | Values                | Default | Description                                                                                                                                      |
|------------|-----------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| "fsType"   | xfs, ext2, ext3, ext4 | ext4    | File system type that will be formatted during volume creation. This parameter is case sensitive!                                                |
| "type"     | ESSD_PL0, PTSSD       | ESSD_PL0| EBS volume type.                                                                                                                                 |
| "zone"     |                       |         | Which zone EBS volume will be created. If empty, csi will choose one zone randomly. If use WaitForFirstConsumer model, you should make it empty! |

**Appendix**
* Unless explicitly noted, all parameters are case insensitive.