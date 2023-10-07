# Driver Options

There are a couple of driver options that can be passed as arguments when starting the driver container.

## EBS CSI

| Option argument          | value sample                      | default                       | Description                                                                                                                                                                   |
|--------------------------|-----------------------------------|-------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| endpoint                 | unix:///custom/csi.sock           | unix:///tmp/csi.sock          | The socket on which the driver will listen for CSI RPCs                                                                                                                       |
| reserveVolumesFactor     | 0.1,0.25,0.3 ...                  | 0.3                           | Value for the maximum number of volumes attachable per node. If specified, the limit applies to all nodes. If not specified, the value is approximated from the instance type |
| topology-region-key      | ebs.topology.kubernetes.io/region | topology.kubernetes.io/region | topology region key for node label and volume affinity.                                                                                                                       |
| topology-zone-key        | ebs.topology.kubernetes.io/zone   | topology.kubernetes.io/zone   | topology zone key for node label and volume affinity.                                                                                                                         |
| httpEndpoint             | :8080                             |                               | The TCP network address where the HTTP server for metrics will listen. The default is empty string, which means the server is disabled.                                       |

## NAS CSI

| Option argument   | value sample            | default                     | Description                                                                           |
|-------------------|-------------------------|-----------------------------|---------------------------------------------------------------------------------------|
| endpoint          | unix:///custom/csi.sock | unix:///tmp/csi.sock        | The socket on which the driver will listen for CSI RPCs                               |
| node-id           | nodeName1               |                             | The default is empty string, which means it will be got from VOLC ECS metadata server |

## TOS CSI

| Option argument   | value sample             | default                     | Description                                                                           |
|-------------------|--------------------------|-----------------------------|---------------------------------------------------------------------------------------|
| endpoint          | unix:///custom/csi.sock  | unix:///tmp/csi.sock        | The socket on which the driver will listen for CSI RPCs                               |
| node-id           | nodeName1                |                             | The default is empty string, which means it will be got from VOLC ECS metadata server |
