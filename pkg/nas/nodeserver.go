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
	"fmt"
	"os"
	"strings"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NodeServer driver
type NodeServer struct {
	d *Driver
}

// NewNodeServer new NodeServer
func NewNodeServer(driver *Driver) *NodeServer {
	return &NodeServer{
		d: driver,
	}
}

// NodeStageVolume stage volume
func (ns *NodeServer) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume unstage volume
func (ns *NodeServer) NodeUnstageVolume(_ context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodePublishVolume mount the volume
func (ns *NodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// Check parameters
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target volumePath missing in request")
	}
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}

	// volume is already mount ?
	notMnt, err := util.DefaultMounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Errorf(codes.Internal, "mkdir targetPath err %s", err.Error())
			}
			notMnt = true
		} else {
			return nil, status.Errorf(codes.Internal, "check targetPath mounted err: %s", err.Error())
		}
	}
	if !notMnt {
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// parse mount options
	mountOptions := req.GetVolumeCapability().GetMount().GetMountFlags()
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}
	nfsVersion, nfsOptionsStr := parseMountOptions(mountOptions)
	if nfsVersion == "" {
		nfsVersion = "3"
	}
	server := req.GetVolumeContext()[paramServer]
	nfsPath := req.GetVolumeContext()[paramPath]

	// if nfsPath is empty, use root directory
	if nfsPath == "" {
		return nil, status.Error(codes.InvalidArgument, "path can not be empty")
	}
	// remove / if nfsPath end with /;
	if nfsPath != "/" && strings.HasSuffix(nfsPath, "/") {
		nfsPath = nfsPath[0 : len(nfsPath)-1]
	}

	var fsId, subPath string
	if req.GetVolumeContext()[FSID] != "" {
		fsId, subPath = getNfsPathDetail(nfsPath)
	} else {
		subPath = nfsPath
	}
	if err := doNfsMount(server, fsId, subPath, nfsVersion, nfsOptionsStr, targetPath, req.GetVolumeId()); err != nil {
		klog.Errorf("NodePublishVolume: %s, Mount server: %s, nfsPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.GetVolumeId(), server, nfsPath, nfsVersion, nfsOptionsStr, targetPath, err.Error())
		return nil, status.Errorf(codes.Internal, "NodePublishVolume: %s, Mount server: %s, nfsPath: %s, nfsVersion: %s, nfsOptions: %s, mountPoint: %s, with error: %s", req.GetVolumeId(), server, nfsPath, nfsVersion, nfsOptionsStr, targetPath, err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume
func (ns *NodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if err := util.DefaultMounter.ForceUnmount(targetPath); err != nil {
		if strings.Contains(err.Error(), "not mounted") || strings.Contains(err.Error(), "no mount point specified") {
			klog.Infof("mountpoint not mounted, skipping: %s", targetPath)
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		klog.Errorf("umount nas %s fail: %v", targetPath, err)
		return nil, status.Errorf(codes.Internal, "umount nas fail: %s", err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeGetVolumeStats get volume stats
func (ns *NodeServer) NodeGetVolumeStats(_ context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	var err error
	targetPath := req.GetVolumePath()
	if targetPath == "" {
		err = fmt.Errorf("NodeGetVolumeStats targetpath %v is empty", targetPath)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return util.GetMetrics(targetPath)
}

// NodeExpandVolume node expand volume
func (ns *NodeServer) NodeExpandVolume(_ context.Context, _ *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeGetCapabilities return the capabilities of the Node plugin
func (ns *NodeServer) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.d.NSCap,
	}, nil
}

// NodeGetInfo return info of the node on which this plugin is running
func (ns *NodeServer) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.d.NodeID,
	}, nil
}
