/*
Copyright 2023 Beijing Volcano Engine Technology Ltd.

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

package tos

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"
	"github.com/volcengine/volcengine-csi-driver/pkg/util/inflight"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// NodeServer driver
type NodeServer struct {
	d        *Driver
	inFlight *inflight.InFlight
}

// NewNodeServer new NodeServer
func NewNodeServer(driver *Driver) *NodeServer {
	return &NodeServer{
		d:        driver,
		inFlight: inflight.NewInFlight(),
	}
}

type tosfsOptions struct {
	URL             string
	Bucket          string
	Path            string
	DbgLevel        string
	AdditionalArgs  string
	NotsupCompatDir bool
}

// NodePublishVolume mount the volume
func (ns *NodeServer) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	if err := validateNodePublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	volID := req.GetVolumeId()
	if ok := ns.inFlight.Insert(volID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volID)
	}
	defer func() {
		klog.V(4).InfoS("NodePublishVolume: volume operation finished", "volumeID", volID)
		ns.inFlight.Delete(volID)
	}()
	targetPath := req.GetTargetPath()

	options, err := parseTosfsOptions(req.GetVolumeAttributes())
	if err != nil {
		klog.Errorf("parse options from VolumeAttributes for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.InvalidArgument, "parse options failed: %v", err)
	}
	subPath := options.Path
	options.Path = ""
	options.NotsupCompatDir = true

	// create the tmp credential info from NodePublishSecrets
	credFilePath, err := createCredentialFile(volID, options.Bucket, req.GetNodePublishSecrets())
	if err != nil {
		return nil, err
	}

	// create tos subPath if not exist
	tosTmpPath := filepath.Join(tempMntPath, volID)
	if err = os.MkdirAll(tosTmpPath, 0750); err != nil {
		klog.Errorf("create tosTmpPath for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "create tosTmpPath for %s failed: %v", volID, err)
	}
	notMnt, err := util.DefaultMounter.IsLikelyNotMountPoint(tosTmpPath)
	if err != nil {
		klog.Errorf("check tosTmpPath IsLikelyNotMountPoint for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "check tosTmpPath IsLikelyNotMountPoint for %s failed: %v", volID, err)
	}
	defer func() {
		if err != nil {
			util.DefaultMounter.Unmount(tosTmpPath)
			util.DefaultMounter.Unmount(targetPath)
		}
	}()
	if notMnt {
		if err = mount(options, tosTmpPath, credFilePath); err != nil {
			klog.Errorf("Mount %s to %s failed: %v", volID, tosTmpPath, err)
			return nil, status.Errorf(codes.Internal, "mount failed: %v", err)
		}
	}
	err = checkTosMounted(tosTmpPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check tosTmpPath mounted fail: %s, please check tos bucket existed and ak/sk is correct", err)
	}

	destPath := filepath.Join(tosTmpPath, subPath)
	if err = os.MkdirAll(destPath, 0750); err != nil {
		klog.Errorf("Tos Create Sub Directory fail, path: %s, err: %v", destPath, err)
		return nil, status.Errorf(codes.Internal, "create subPath error: %v", err)
	}

	// umount tmp path
	if err = util.DefaultMounter.Unmount(tosTmpPath); err != nil {
		klog.Errorf("Failed to umount tosTmpPath %s for volume %s: %v", tosTmpPath, volID, err)
		return nil, status.Errorf(codes.Internal, "umount failed: %v", err)
	}
	if err = os.Remove(tosTmpPath); err != nil {
		klog.Errorf("Failed to remove tosTmpPath %s for volume %s: %v", tosTmpPath, volID, err)
		return nil, status.Errorf(codes.Internal, "remove tosTmpPath failed: %v", err)
	}

	// mount targetPath
	options.Path = subPath
	options.NotsupCompatDir = false
	if err = os.MkdirAll(targetPath, 0750); err != nil {
		klog.Errorf("create targetPath for %s failed: %v", volID, err)
		return nil, status.Errorf(codes.Internal, "create targetPath for %s failed: %v", volID, err)
	}
	notMnt, err = util.DefaultMounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		klog.Errorf("isMountPoint for %s failed: %v", volID, err)
		return nil, err
	}
	if !notMnt {
		klog.Infof("Volume %s is already mounted to %s, skipping", volID, targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}
	if err = mount(options, targetPath, credFilePath); err != nil {
		klog.Errorf("Mount %s to %s failed: %v", volID, targetPath, err)
		return nil, status.Errorf(codes.Internal, "mount failed: %v", err)
	}
	err = checkTosMounted(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check targetPath mounted fail: %s, please check tos bucket existed and ak/sk is correct", err)
	}

	klog.Infof("successfully mounted volume %s to %s", volID, targetPath)

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unmount the volume
func (ns *NodeServer) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	if err := validateNodeUnpublishVolumeRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	volID := req.GetVolumeId()
	if ok := ns.inFlight.Insert(volID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volID)
	}
	defer func() {
		klog.V(4).InfoS("NodeUnpublishVolume: volume operation finished", "volumeID", volID)
		ns.inFlight.Delete(volID)
	}()
	targetPath := req.GetTargetPath()

	if err := util.DefaultMounter.Unmount(targetPath); err != nil {
		if strings.Contains(err.Error(), "not mounted") || strings.Contains(err.Error(), "no mount point specified") {
			klog.Infof("mountpoint not mounted, skipping: %s", targetPath)
			return &csi.NodeUnpublishVolumeResponse{}, nil
		}
		klog.Errorf("failed to umount point %s for volume %s: %v", targetPath, volID, err)
		return nil, status.Errorf(codes.Internal, "umount tos failed: %v", err)
	}

	klog.Infof("Successfully unmounted volume %s from %s", volID, targetPath)

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeStageVolume stage volume
func (ns *NodeServer) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume unstage volume
func (ns *NodeServer) NodeUnstageVolume(_ context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
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

// NodeGetId return id of the node on which this plugin is running
func (ns *NodeServer) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{
		NodeId: ns.d.NodeID,
	}, nil
}
