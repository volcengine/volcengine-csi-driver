# Install TOS CSI driver

## Install with kubectl

- Step1: Create [CSIDriver](https://kubernetes-csi.github.io/docs/csi-driver-object.html)

```bash
# Create CSIDriver
kubectl create -f ./tos-driverinfo.yaml
```

- Step2: Deploy tos launcher

```bash
# Create launcher
kubectl create -f ./tos-launcher.yaml
```

- Step3: Deploy Node Plugin

```bash
# Create Node Plugin
kubectl create -f ./tos-node.yaml
```

- Step4: Check Status of CSI plugin

```bash
# kubectl get pods -n kube-system | grep tos
csi-tos-launcher-mqvst                                  1/1     Running   0          70s
csi-tos-node-s46x5                                      3/3     Running   0          70s
```

> Please refer to [CSI Driver Example](../../example/tos/README.md) for more example.
