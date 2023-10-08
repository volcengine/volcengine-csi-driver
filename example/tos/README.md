# CSI Driver Example

After the TOS CSI Driver is deployed in your cluster, you can follow this documentation to quickly deploy some examples.

You can use TOS CSI Driver to provision Persistent Volumes statically or dynamically. Please read [Kubernetes Persistent Volumes documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) for more information about Static and Dynamic provisioning.


## Prerequisite

We recommend [Volcengine Kubernetes Engine（VKE](https://www.volcengine.com/product/vke), which will deploy CSI drivers for you automatically.

Alternatively, you can try to deploy it manually on [ECS](https://www.volcengine.com/product/ecs), which is covered by the follow document. We do not provide commercial support for such deployment, however.

- [Install TOS CSI Driver](../../deploy/tos/README.md)

## PV/PVC Usage (Static Provisioning)

- Create `PersistentVolume` and `PersistentVolumeClaim` statically.

```bash
# Create tos secret
kubectl create -f ./secret.yaml

# Create PV
kubectl create -f ./tos-pv.yaml

# Create PVC
kubectl create -f ./tos-pvc.yaml
```


- Check the status of `PersistentVolume` and `PersistentVolumeClaim`.

```bash
# kubectl get pvc
NAME          STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
csi-tos-pvc   Bound    csi-tos-pv                                 1Gi      RWX                           13s

# kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS     CLAIM                 STORAGECLASS   REASON   AGE
pv-tos                                        1Gi      RWX            Retain           Bound   default/csi-tos-pvc                           83d
```

## Pod Usage

- Create a `Pod` use the `PersistentVolumeClaim`.

```bash
# Create Pod
kubectl create -f ./csi-pod.yaml
```

- Check the status of `Pod`.

```bash
# kubectl get pod | grep csi
csi-pod          1/1     Running   0          1m
```

## Cleanup

- Cleanup resources.

```bash
# Cleanup
kubectl delete -f ./
```