# CreateVolume Parameters
There are several optional parameters that may be passed into `CreateVolumeRequest.parameters` map, these parameters can be configured in StorageClass, see [example](../example/ebs/csi-storageclass.yaml).

| Parameters | Values                | Default | Description                                                                                                                                      |
|------------|-----------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| "fsType"   | xfs, ext2, ext3, ext4 | ext4    | File system type that will be formatted during volume creation. This parameter is case sensitive!                                                |
| "type"     | ESSD_PL0, PTSSD       | ESSD_PL0| EBS volume type.                                                                                                                                 |
| "zone"     |                       |         | Which zone EBS volume will be created. If empty, csi will choose one zone randomly. If use WaitForFirstConsumer model, you should make it empty! |
| "tags"     | `key1\|value1,key2\|value2` | None | User tags applied to newly created EBS volumes. The parameter name and tag keys/values are case sensitive. |

## Volume tags

The `tags` parameter follows the [VKE CSI format](https://www.volcengine.com/docs/6460/160889): comma-separated tags, with the first `|` in each entry separating the key and value.

```yaml
parameters:
  type: ESSD_PL0
  csi.storage.k8s.io/fstype: xfs
  tags: "team|storage,env|test"
```

See the complete [StorageClass example](../example/ebs/csi-storageclass-tags.yaml).

- Omitting `tags` or setting it to `""` does not add any user tags, but the default `createdby` tag (see below) is still applied.
- Duplicate keys use the last value. `key|` represents an empty value. Spaces are preserved, not trimmed.
- Entries without `|` (including a trailing comma) and keys starting with `volc:` or `sys:` are rejected with `InvalidArgument` before cloud API calls. Remaining tag constraints, including tag count and allowed characters, are validated by the EBS API.
- Tags are included in the `CreateVolume` request for both fresh volumes and volumes restored from snapshots. No separate tagging API is called, and no cluster or Kubernetes metadata tags are added automatically.
- Tags apply at creation time only. Existing volumes returned by an idempotent request are not retagged; this parameter does not update static PVs or tag snapshots.

### Default `createdby` tag

Every newly created EBS volume (both fresh volumes and volumes restored from snapshots) receives a default tag `createdby=volcengine-ebs-csi-driver` when the `tags` parameter does not explicitly contain a `createdby` key. This is a default, not a forced override:

- If the user sets `createdby` in `tags` (including `createdby|` with an empty value), the user-supplied value is preserved and the default is not applied.
- The default tag is added in the CSI controller before the `CreateVolume` cloud API call, so it counts toward any EBS API tag-count limit alongside user tags.
- Existing volumes are not retroactively tagged.

**Appendix**
* Unless explicitly noted, all parameters are case insensitive.
