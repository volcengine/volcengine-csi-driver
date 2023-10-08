# Driver Metrics

## EBS API Metrics
### Prerequisites

1. Enable metrics by setting `enableMetrics: true` in ./charts/csi-ebs/values.yaml.

2. Deploy EBS CSI Driver:
```sh
$ helm upgrade --install csi-ebs --namespace kube-system ./charts/csi-ebs --values ./charts/csi-ebs/values.yaml
```

The EBS CSI Driver will emit [EBS API](https://www.volcengine.com/docs/26444/434983) metrics to the following TCP endpoint: `0.0.0.0:19809/metrics` if `enableMetrics: true` has been configured in the Helm chart.

The metrics will appear in the following format: 
```sh
# HELP volc_api_request_duration_seconds [ALPHA] Latency of VOLC API calls
# TYPE volc_api_request_duration_seconds histogram
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.005"} 0
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.01"} 0
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.025"} 0
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.05"} 0
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.1"} 0
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.25"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="0.5"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="1"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="2.5"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="5"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="10"} 1
volc_api_request_duration_seconds_bucket{action="AttachVolume",method="GET",version="2020-04-01",le="+Inf"} 1
volc_api_request_duration_seconds_sum{action="AttachVolume",method="GET",version="2020-04-01"} 0.170908792
volc_api_request_duration_seconds_count{action="AttachVolume",method="GET",version="2020-04-01"} 1
...
```

To manually scrape EBS metrics: 
```sh
$ export ebs_csi_controller=$(kubectl get lease -n kube-system ebs-csi-volcengine-com -o=jsonpath="{.spec.holderIdentity}")
$ kubectl port-forward $ebs_csi_controller 19809:19809 -n kube-system
$ curl 127.0.0.1:19809/metrics
```

## Volume Stats Metrics

The EBS/NAS CSI Driver emits Kubelet mounted volume metrics for volumes created with the driver. 

The following metrics are currently supported:

| Metric name | Metric type | Description | Labels |
|-------------|-------------|-------------|-------------|
|kubelet_volume_stats_capacity_bytes|Gauge|The capacity in bytes of the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_available_bytes|Gauge|The number of available bytes in the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_used_bytes|Gauge|The number of used bytes in the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes|Gauge|The maximum number of inodes in the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes_free|Gauge|The number of free inodes in the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 
|kubelet_volume_stats_inodes_used|Gauge|The number of used inodes in the volume|namespace=\<persistentvolumeclaim-namespace\> <br/> persistentvolumeclaim=\<persistentvolumeclaim-name\>| 

For more information about the supported metrics, see `VolumeUsage` within the CSI spec documentation for the [NodeGetVolumeStats](https://github.com/container-storage-interface/spec/blob/master/spec.md#nodegetvolumestats) RPC call.

For more information about metrics in Kubernetes, see the [Metrics For Kubernetes System Components](https://kubernetes.io/docs/concepts/cluster-administration/system-metrics/#metrics-in-kubernetes) documentation.

## CSI Operations Metrics

The `csi_operations_seconds metrics` reports a latency histogram of kubelet-initiated CSI gRPC calls by gRPC status code.

To manually scrape Kubelet metrics: 
```sh
$ kubectl proxy
$ kubectl get --raw /api/v1/nodes/<insert_node_name>/proxy/metrics
```
