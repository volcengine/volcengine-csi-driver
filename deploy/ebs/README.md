# Install EBS CSI driver

## Compiling and Package
the driver can be compiled in a form of a container.

To build a container:
```
$  make container CMDS=ebsplugin
```

## Install with kubectl

- Step1: Create [CSIDriver](https://kubernetes-csi.github.io/docs/csi-driver-object.html)

```bash
# Create CSIDriver
kubectl create -f ./csi-ebs-driverinfo.yaml
```

- Step2: Deploy Controller Plugin

```bash
# Create ServiceAccount
kubectl create -f ./rbac-csi-ebs-controller.yaml

# Create Controller Plugin
kubectl create -f ./csi-ebs-controller.yaml
```

> Note: You need to config AKSK and server info to plugin; Set VOLC_ACCESSKEY, VOLC_SECRETKEY, VOLC_HOST to environment in csi-ebs-driver container.

> VOLC_ACCESSKEY: Access Key ID of VOLC engine IAM uesr, using for invoking volc engine iaas ebs api.   

> VOLC_SECRETKEY: Secret Access Key of VOLC engine IAM uesr, using for invoking volc engine iaas ebs api.

> VOLC_HOST: Host of VOLC engine api server. e.g.,  open.volcengineapi.com

- Step3: Deploy Node Plugin

```bash
# Create Node Plugin
kubectl create -f ./csi-ebs-node.yaml
```

> Note: You need to config AKSK and server info to plugin too; Set VOLC_ACCESSKEY, VOLC_SECRETKEY, VOLC_HOST to environment in csi-ebs-driver container;

- Step4: Check Status of CSI plugin

```bash
# kubectl get pods -A |grep csi-ebs
kube-system   csi-ebs-controller-fb84c647d-mq8k9         4/4     Running   0          1h
kube-system   csi-ebs-node-kbgkm                         3/3     Running   0          1h
```

- Step5: Create StorageClass && PVC

```bash
# Create SC
kubectl create -f ./../../example/ebs/csi-storageclass.yaml

# Create PVC
kubectl create -f ./../../example/ebs/csi-pvc.yaml
```
  
> Please refer to [CSI Driver Example](../../example/ebs/README.md) for more example.

