# CSI-Driver

[![GoReportCard Widget]][GoReportCardResult]

English | [简体中文](./README-zh_CN.md)

## Introduction
Volcengine CSI plugins implement the lifecycle management of storage volume of Volcengine in Kubernetes. It allows dynamically provision Disk
volumes and attach it to workloads.

Current implementation of CSI plugins has been tested in Kubernetes environment (requires Kubernetes v1.20+).

Current Support: ***EBS, NAS, TOS***;

### EBS CSI Plugin

EBS CSI plugin supports dynamic creation and mounting of elastic block storage volumes. Elastic Block Storage (EBS) is a block storage device with high availability, high reliability, high performance, and elastic scalability provided by the volcano engine. It can be used as an expandable disk for cloud servers and elastic container services.

More detail information please refer to [EBS](./example/ebs/README.md).

### NAS CSI Plugin

NAS CSI Plugin supports the mounting and use of NAS storage volumes for workloads, as well as the dynamic creation of NAS volumes. NAS file storage is a file storage service for volcanic engine elastic computing, container services, and AI intelligent applications. It provides a cost-effective cloud storage service with high-performance shared access, continuous online access, elastic expansion, and cross-region access for business applications.

More detail information please refer to [NAS](./example/nas/README.md).

### TOS CSI Plugin

TOS CSI Plugin support TOS bucket mount, but does not support provision volume. Tinder Object Storage (TOS) is a massive, secure, low-cost, easy-to-use, highly reliable, and highly available distributed cloud storage service provided by Volcano Engine.

More detail information pls refer to [TOS](./example/tos/README.md).

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](https://kubernetes.io/community/).

You can reach the maintainers of this project at the [Cloud Provider SIG](https://github.com/kubernetes/community/tree/master/sig-cloud-provider).

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

Please submit an issue at: [Issues](https://github.com/volcengine/volcengine-csi-driver/issues)
