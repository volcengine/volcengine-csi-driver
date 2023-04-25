# 火山引擎 Kubernetes CSI 插件
[![GoReportCard Widget]][GoReportCardResult]

[English](./README.md) | 简体中文

## 插件介绍

火山引擎 CSI 插件实现了在 Kubernetes 中对火山引擎中存储卷的生命周期管理，支持动态创建、挂载和使用存储数据卷。 当前的 CSI 实现基于 K8S v1.20 以上的版本；

支持的火山引擎存储服务：***EBS，NAS，TOS***

### EBS CSI 插件

EBS CSI 插件支持动态创建弹性块存储卷，挂载存储卷。弹性块存储 EBS（Elastic Block Storage）是火山引擎提供的高可用、高可靠、高性能、弹性扩展的块存储设备，可以作为云服务器和弹性容器服务的可扩展硬盘使用。

EBS CSI 插件更多详细说明请参考 [EBS](./example/ebs/README.md)。

### NAS CSI 插件

NAS CSI 插件支持为应用负载挂载和使用火山引擎 NAS 存储卷，也支持动态创建 NAS 卷。NAS 文件存储是面向火山引擎弹性计算、容器服务、AI 智能应用的文件存储服务，可为业务应用提供一种高性能共享访问、持续在线、弹性扩展、跨地域访问的高性价比云存储服务。

NAS CSI插件更多详细说明请参考 [NAS](./example/nas/README.md)。

### TOS CSI 插件

TOS CSI 插件支持火山引擎 TOS 存储桶的挂载，当前还不支持动态创建存储桶。火山引擎对象存储 TOS（Tinder Object Storage）是火山引擎提供的海量、安全、低成本、易用、高可靠、高可用的分布式云存储服务。

TOS CSI 插件更多详细说明请参考 [TOS](./example/tos/README.md)。

## 社区, 贡献, 讨论, 支持

可以到 [Kubernetes](https://kubernetes.io/community/) 社区学到如何获取支持；

可以到 [Cloud Provider SIG](https://github.com/kubernetes/community/tree/master/sig-cloud-provider) 联系到项目管理者；


### 行为准则

参与Kubernetes社区参考 [Kubernetes 行为准则](code-of-conduct.md)；

可以向社区提交 [Issue](https://github.com/volcengine/volcengine-csi-driver/issues)；
