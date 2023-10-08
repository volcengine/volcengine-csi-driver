# CSI Driver Example

After the NAS CSI Driver is deployed in your cluster, you can follow this documentation to quickly deploy some examples.

You can use NAS CSI Driver to provision Persistent Volumes statically or dynamically. Please read [Kubernetes Persistent Volumes documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) for more information about Static and Dynamic provisioning.


## Prerequisite

We recommend [Volcengine Kubernetes Engine（VKE](https://www.volcengine.com/product/vke), which will deploy CSI drivers for you automatically.

Alternatively, you can try to deploy it manually on [ECS](https://www.volcengine.com/product/ecs), which is covered by the follow document. We do not provide commercial support for such deployment, however.

- [Install NAS CSI Driver](../../deploy/nas/README.md)

## Storage Class Usage (Dynamic Provisioning)

- Create a `StorageClass`, and then `PersistentVolume` and `PersistentVolumeClaim` dynamically.

```bash
# Create StorageClass
kubectl create -f ./csi-storageclass.yaml

# Create PVC 
kubectl create -f ./csi-pvc.yaml
```

- Check the status of `PersistentVolume` and `PersistentVolumeClaim`.

```bash
# kubectl get sc
NAME             PROVISIONER              RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
volcengine-nas   nas.csi.volcengine.com   Delete          Immediate           false                  5m

# kubectl get pvc
NAME                 STATUS      VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS     AGE
csi-nas-pvc          Bound       pvc-44704213-5e21-4532-b644-b2e7e2a2c76b   10Gi       RWX            volcengine-nas   3m

# kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                        STORAGECLASS     REASON   AGE
pvc-44704213-5e21-4532-b644-b2e7e2a2c76b   10Gi       RWX            Delete           Bound    default/csi-nas-pvc          volcengine-nas            2m
```

## PV/PVC Usage (Static Provisioning)

- Create `PersistentVolume` and `PersistentVolumeClaim` statically.

```bash
# Create PV
kubectl create -f ./csi-pv.yaml

# Create PVC
kubectl create -f ./csi-pvc-static.yaml
```

- Check the status of `PersistentVolume` and `PersistentVolumeClaim`.

```bash
# kubectl get pvc
NAME                 STATUS        VOLUME                      CAPACITY   ACCESS MODES   STORAGECLASS     AGE
csi-nas-pvc-static   Bound         csi-nas-pv-static           10Gi       RWX                             5m

# kubectl get pv
NAME                 CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                        STORAGECLASS     REASON   AGE
csi-nas-pv-static    10Gi       RWX            Retain           Bound    default/csi-nas-pvc-static                             4m
```

## Pod Usage

- Create a `Pod` use the `PersistentVolumeClaim`.

```bash
# Create Pod
kubectl create -f ./csi-pod.yaml
```

- Check the status of `Pod`.

```bash
# kubectl get pod |grep csi
csi-pod          1/1     Running   0          10m

# kubectl exec -it csi-pod -- sh
# ls -lah /data/
total 8.0K
drwxrwxrwx  2   root    root    22    Jul 21 07:25    .
drwxr-xr-x  1   root    root    4.0K  Jul 21 08:32    ..
-rw-r--r--  1   root    root    30    Jul 21 07:25    testfile
# mount |grep /data
100.65.230.38:/fs-039e9a16 on /data type nfs (rw,relatime,vers=3,rsize=1048576,wsize=1048576,namlen=255,hard,nolock,noresvport,proto=tcp,timeo=600,retrans=2,sec=sys,mountaddr=100.65.230.38,mountvers=3,mountport=892,mountproto=tcp,local_lock=all,addr=100.65.230.38)
```

## Cleanup

- Cleanup resources.

```bash
# Cleanup
kubectl delete -f ./
```