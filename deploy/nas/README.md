# Install NAS CSI driver

## Install with kubectl

- Step1: Create [CSIDriver](https://kubernetes-csi.github.io/docs/csi-driver-object.html)

```bash
# Create CSIDriver
kubectl create -f ./csi-nas-driverinfo.yaml
```

- Step2: Deploy Controller Plugin

```bash
# Create ServiceAccount
kubectl create -f ./rbac-csi-nas-controller.yaml

# Create Controller Plugin
kubectl create -f ./csi-nas-controller.yaml
```

> Note: You need to config AKSK info to plugin; Set VOLC_ACCESSKEY, VOLC_SECRETKEY, VOLC_REGION and VOLC_HOST to environment;

- Step3: Deploy Node Plugin

```bash
# Create Node Plugin
kubectl create -f ./csi-nas-node.yaml
```

> Note: You need to config AKSK info to plugin; Set VOLC_ACCESSKEY, VOLC_SECRETKEY, VOLC_REGION and VOLC_HOST to environment;

- Step4: Check Status of CSI plugin

```bash
# kubectl get pods -A |grep csi-nas
kube-system   csi-nas-controller-fb84c647d-mq8k9         4/4     Running   0          1h
kube-system   csi-nas-node-kbgkm                         3/3     Running   0          1h
```

- Step5: Create StorageClass && PVC

```bash
# Create SC
kubectl create -f ./../../example/nas/csi-storageclass.yaml

# Create PVC
kubectl create -f ./../../example/nas/csi-pvc.yaml
```

> Please refer to [CSI Driver Example](../../example/nas/README.md) for more example.

