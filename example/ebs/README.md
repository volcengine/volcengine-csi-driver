# CSI Driver Example

After the EBS CSI Driver is deployed in your cluster, you can follow this documentation to quickly deploy some examples.

You can use EBS CSI Driver to provision Persistent Volumes statically or dynamically. Please read [Kubernetes Persistent Volumes documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) for more information about Static and Dynamic provisioning.

Please refer to [driver parameters](../../docs/csi-ebs-parameters.md) for more detailed usage.

## Prerequisite

- [Install EBS CSI Driver](../../deploy/ebs/README.md)

## Storage Class Usage (Dynamic Provisioning)

- Create a `StorageClass`, and then `PersistentVolume` and `PersistentVolumeClaim` dynamically.

```bash
# Create StorageClass
kubectl create -f ./csi-storageclass.yaml

# Create PVC 
kubectl create -f ./csi-pvc.yaml

# Create PVC (raw block mode)
kubectl create -f ./csi-pvc-block.yaml
```

- Check the status of `PersistentVolume` and `PersistentVolumeClaim`.

```bash
# kubectl get sc
NAME             PROVISIONER              RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
volcengine-ebs   ebs.csi.volcengine.com   Delete          Immediate           false                  5m

# kubectl get pvc
NAME                 STATUS      VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS     AGE
csi-ebs-pvc          Bound       pvc-44704213-5e21-4532-b644-b2e7e2a2c76b   10Gi       RWO            volcengine-ebs   3m
csi-ebs-pvc-raw      Bound       pvc-148f2dd1-0878-4c9a-87fa-5bdc94a27d6c   10Gi       RWO            volcengine-ebs   3m

# kubectl get pv
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                        STORAGECLASS     REASON   AGE
pvc-148f2dd1-0878-4c9a-87fa-5bdc94a27d6c   10Gi       RWO            Delete           Bound    default/csi-ebs-pvc-raw      volcengine-ebs            2m
pvc-44704213-5e21-4532-b644-b2e7e2a2c76b   10Gi       RWO            Delete           Bound    default/csi-ebs-pvc          volcengine-ebs            2m
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
csi-ebs-pvc-static   Bound         csi-ebs-pv-static           10Gi       RWO                             5m

# kubectl get pv
NAME                 CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                        STORAGECLASS     REASON   AGE
csi-ebs-pv-static    10Gi       RWO            Retain           Bound    default/csi-ebs-pvc-static                             4m
```

## Pod Usage

- Create a `Pod` use the `PersistentVolumeClaim`.

```bash
# Create Pod
kubectl create -f ./csi-pod.yaml

# Create Pod (raw block mode)
kubectl create -f ./csi-pod-block.yaml
```

- Check the status of `Pod`.

```bash
# kubectl get pod |grep csi
csi-pod          1/1     Running   0          10m
csi-pod-raw      1/1     Running   0          8m

# kubectl exec -it csi-pod -- sh
# ls -lah /data/
total 24K
drwxr-xr-x    3 root     root        4.0K Mar 28 08:37 .
drwxr-xr-x    1 root     root        4.0K Mar 28 07:00 ..
drwx------    2 root     root       16.0K Mar 28 07:00 lost+found
-rw-r--r--    1 root     root           0 Mar 28 08:37 testfile
# mount |grep /data
/dev/vdd on /data type ext4 (rw,relatime,data=ordered)

# kubectl exec -it csi-pod-raw -- sh
# ls -alh /dev/loop6
brw-rw-rw-    1 root     disk      253,  64 Mar 28 07:06 /dev/loop6
```

## Cleanup

- Cleanup resources.

```bash
# Cleanup
kubectl delete -f ./
```