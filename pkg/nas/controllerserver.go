/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nas

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// ControllerServer controller server setting
type ControllerServer struct {
	d      *Driver
	client kubernetes.Interface
}

// nas volume parameters
type nasVolumeArgs struct {
	VolumeAs        string `json:"volumeAs"`
	Server          string `json:"server"`
	FsID            string `json:"fsId"`
	SubPath         string `json:"subPath"`
	ArchiveOnDelete string `json:"archiveOnDelete"`
}

// NewControllerServer new ControllerServer
func NewControllerServer(driver *Driver) *ControllerServer {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("NewControllerServer: Failed to create config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("NewControllerServer: Failed to create client: %v", err)
	}
	return &ControllerServer{driver, clientset}
}

func (cs *ControllerServer) CreateVolume(_ context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	pvName := req.Name
	nfsOptions := []string{}
	for _, volCap := range req.VolumeCapabilities {
		volCapMount := ((*volCap).AccessType).(*csi.VolumeCapability_Mount)
		nfsOptions = append(nfsOptions, volCapMount.Mount.MountFlags...)
	}
	nfsVersion, nfsOptionsStr := parseMountOptions(nfsOptions)
	if nfsVersion == "" {
		nfsVersion = "3"
	}

	volumeContext := map[string]string{}
	var csiTargetVol *csi.Volume

	nasVolArgs, err := getNasVolumeArgs(req.GetParameters())
	if err != nil {
		klog.Errorf("Invalid parameters from input: %v, with error: %v", req.Name, err)
		return nil, status.Errorf(codes.InvalidArgument, "Invalid parameters from input: %v, with error: %v", req.Name, err)
	}

	if nasVolArgs.VolumeAs == "subpath" {
		nfsServer := nasVolArgs.Server
		fsId := nasVolArgs.FsID
		subPath := nasVolArgs.SubPath
		nfsPath := filepath.Join("/"+fsId, subPath)

		// vci instance does not have mount privilege
		if cs.d.AssumeRoleName != VCIInstanceRoleForVKE {
			klog.Infof("Create Volume %s with Exist Nfs Server: %s,  Path: %s, Options: %s, Version: %s", req.Name, nfsServer, nfsPath, nfsOptions, nfsVersion)
			// create tmp mountpoint
			mountPoint := filepath.Join(MntRootPath, pvName)
			// Mount nfs server to tmp mountpoint, this will create the subdirectory if not exiest
			if err := doNfsMount(nfsServer, fsId, subPath, nfsVersion, nfsOptionsStr, mountPoint, req.Name); err != nil {
				klog.Errorf("CreateVolume: %s, Mount server: %s, fsId: %s, subPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.Name, nfsServer, fsId, subPath, nfsVersion, nfsOptionsStr, mountPoint, err.Error())
				return nil, status.Errorf(codes.Internal, "CreateVolume: %s, Mount server: %s, fsId: %s, subPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.Name, nfsServer, fsId, subPath, nfsVersion, nfsOptionsStr, mountPoint, err.Error())
			}

			// create volume subpath
			fullPath := filepath.Join(mountPoint, pvName)
			if err := os.MkdirAll(fullPath, 0777); err != nil {
				util.DefaultMounter.Unmount(mountPoint)
				klog.Errorf("Provision: %s, creating path: %s, with error: %s", req.Name, fullPath, err.Error())
				return nil, status.Errorf(codes.Internal, "Provision: %s, creating path: %s, with error: %s", req.Name, fullPath, err.Error())
			}
			os.Chmod(fullPath, 0777)

			// Unmount nfs server
			if err := util.DefaultMounter.Unmount(mountPoint); err != nil {
				klog.Errorf("Provision: %s, unmount nfs mountpoint %s failed with error %v", req.Name, mountPoint, err)
				return nil, status.Errorf(codes.Internal, "Provision: %s, unmount nfs mountpoint %s failed with error %v", req.Name, mountPoint, err)
			}
		}

		volumeContext["archiveOnDelete"] = nasVolArgs.ArchiveOnDelete
		volumeContext["volumeAs"] = nasVolArgs.VolumeAs
		volumeContext["path"] = filepath.Join(nfsPath, pvName)
		volumeContext["server"] = nfsServer
		volumeContext[FSID] = fsId

		csiTargetVol = &csi.Volume{
			VolumeId:      req.Name,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			VolumeContext: volumeContext,
		}
	} else {
		// TODO: volumeAs filesystem
		klog.Errorf("CreateVolume %s: volumeAs should be set as subpath: %s", req.Name, nasVolArgs.VolumeAs)
		return nil, status.Errorf(codes.Unimplemented, "CreateVolume: volumeAs should be set as subpath: %s", nasVolArgs.VolumeAs)
	}

	klog.Infof("Provision volume %s Successful with PV: %v", req.Name, csiTargetVol)

	return &csi.CreateVolumeResponse{Volume: csiTargetVol}, nil
}

func (cs *ControllerServer) DeleteVolume(_ context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if cs.d.AssumeRoleName == VCIInstanceRoleForVKE {
		klog.Warningln("not support DeleteVolume in vci")
		return &csi.DeleteVolumeResponse{}, nil
	}

	pvInfo, err := cs.client.CoreV1().PersistentVolumes().Get(context.Background(), req.VolumeId, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("DeleteVolume: get pv %s from cluster error: %s", req.VolumeId, err.Error())
		return nil, status.Errorf(codes.FailedPrecondition, "get pv %s from cluster error: %s", req.VolumeId, err.Error())
	}
	var volumeAs, pvPath, nfsPath, nfsServer, nfsOptions string
	nfsOptions = strings.Join(pvInfo.Spec.MountOptions, ",")
	if pvInfo.Spec.CSI == nil {
		klog.Errorf("DeleteVolume: CSI in volume %s is nil, pv: %v", req.VolumeId, pvInfo)
		return nil, status.Errorf(codes.FailedPrecondition, "CSI in volume %s is nil, pv: %v", req.VolumeId, pvInfo)
	}
	if value, ok := pvInfo.Spec.CSI.VolumeAttributes["volumeAs"]; !ok {
		volumeAs = "subpath"
	} else {
		volumeAs = value
	}
	if value, ok := pvInfo.Spec.CSI.VolumeAttributes["server"]; ok {
		nfsServer = value
	} else {
		klog.Errorf("DeleteVolume: nfs server in volume %s is empty, CSI: %v", req.VolumeId, pvInfo.Spec.CSI)
		return nil, status.Errorf(codes.FailedPrecondition, "nfs server in volume %s is empty, CSI: %v", req.VolumeId, pvInfo.Spec.CSI)
	}
	if value, ok := pvInfo.Spec.CSI.VolumeAttributes["path"]; ok {
		pvPath = value
	} else {
		klog.Errorf("DeleteVolume: path in volume %s is empty, CSI: %v", req.VolumeId, pvInfo.Spec.CSI)
		return nil, status.Errorf(codes.FailedPrecondition, "nfs server in volume %s is empty, CSI: %v", req.VolumeId, pvInfo.Spec.CSI)
	}
	if pvInfo.Spec.StorageClassName == "" {
		klog.Errorf("DeleteVolume: storageclass in volume %s is empty, Spec: %v", req.VolumeId, pvInfo.Spec)
		return nil, status.Errorf(codes.FailedPrecondition, "storageclass in volume %s is empty, Spec: %v", req.VolumeId, pvInfo.Spec)
	}

	if volumeAs == "subpath" {
		nfsVersion := "3"
		if pvPath == "/" || pvPath == "" {
			klog.Errorf("DeleteVolume volume %s err: pvPath cannot be / or empty in subpath mode", req.GetVolumeId())
			return nil, status.Error(codes.FailedPrecondition, "pvPath cannot be / or empty in subpath mode")
		}
		pvPath = strings.TrimSuffix(pvPath, "/")
		pvName := filepath.Base(pvPath)
		pos := strings.LastIndex(pvPath, "/")
		nfsPath = pvPath[0:pos]
		if nfsPath == "" {
			nfsPath = "/"
		}
		fsId, subPath := getNfsPathDetail(nfsPath)

		// create the tmp mountpoint
		mountPoint := filepath.Join(MntRootPath, req.VolumeId+"-delete")
		if err := doNfsMount(nfsServer, fsId, subPath, nfsVersion, nfsOptions, mountPoint, req.VolumeId); err != nil {
			klog.Errorf("DeleteVolume %s error, Mount server: %s, nfsPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.VolumeId, nfsServer, nfsPath, nfsVersion, nfsOptions, mountPoint, err.Error())
			return nil, status.Errorf(codes.Internal, "DeleteVolume: %s, Mount server: %s, nfsPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.VolumeId, nfsServer, nfsPath, nfsVersion, nfsOptions, mountPoint, err.Error())
		}
		defer util.DefaultMounter.Unmount(mountPoint)

		deletePath := filepath.Join(mountPoint, pvName)
		if _, err := os.Stat(deletePath); os.IsNotExist(err) {
			klog.Infof("Delete Volume %s Path %s does not exist, deletion skipped", req.VolumeId, deletePath)
			return &csi.DeleteVolumeResponse{}, nil
		}

		// If archiveOnDelete exists and has a false value, delete the directory. Otherwise, archive it.
		archiveOnDelete, exists := pvInfo.Spec.CSI.VolumeAttributes["archiveOnDelete"]
		if exists {
			archiveBool, err := strconv.ParseBool(archiveOnDelete)
			if err != nil {
				klog.Errorf("volume %s archiveOnDelete %s format error: %s", req.VolumeId, archiveOnDelete, err.Error())
				return nil, status.Errorf(codes.Internal, "archiveOnDelete %s format error: %s", archiveOnDelete, err.Error())
			}
			if !archiveBool {
				if err := os.RemoveAll(deletePath); err != nil {
					return nil, status.Errorf(codes.Internal, "volume %s remove path %s error: %s", req.GetVolumeId(), deletePath, err.Error())
				}
				klog.Infof("Delete volume %s Successful: Removed path %s", req.VolumeId, deletePath)
				return &csi.DeleteVolumeResponse{}, nil
			}
		}

		archivePath := filepath.Join(mountPoint, "archived-"+pvName+time.Now().Format(".2006-01-02-15:04:05"))
		if err := os.Rename(deletePath, archivePath); err != nil {
			klog.Errorf("Delete Failed: Volume %s, archiving path %s to %s with error: %s", req.VolumeId, deletePath, archivePath, err.Error())
			return nil, status.Errorf(codes.Internal, "Delete Failed: Volume %s, archiving path %s to %s with error: %s", req.VolumeId, deletePath, archivePath, err.Error())
		}
		klog.Infof("Delete volume %s Successful: Archiving path %s to %s", req.VolumeId, deletePath, archivePath)
	}

	if volumeAs == "filesystem" {
		// TODO: delete when filesystem type
		klog.Error("not support filesystem type")
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerPublishVolume(_ context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerUnpublishVolume(_ context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ValidateVolumeCapabilities(_ context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id cannot be empty")
	}

	if len(req.VolumeCapabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "capabilities of request cannot be empty")
	}

	for _, capability := range req.GetVolumeCapabilities() {
		if capability.GetAccessMode().GetMode() != csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{}, nil
		}
	}

	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters(),
		},
	}, nil
}

func (cs *ControllerServer) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerGetCapabilities(_ context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.d.CSCap,
	}, nil
}

func (cs *ControllerServer) CreateSnapshot(_ context.Context, _ *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) DeleteSnapshot(_ context.Context, _ *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ListSnapshots(_ context.Context, _ *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerExpandVolume(_ context.Context, _ *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	// TODO: @xueshengjie
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerGetVolume(_ context.Context, _ *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}
