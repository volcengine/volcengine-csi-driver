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

package ebs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/consts"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"
	"github.com/volcengine/volcengine-csi-driver/pkg/util"
	"github.com/volcengine/volcengine-csi-driver/pkg/util/inflight"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

type NodeSvc struct {
	d        *Driver
	mounter  Mounter
	cloud    Cloud
	inFlight *inflight.InFlight
}

var (
	// BLOCKVOLUMEPREFIX block volume mount prefix
	BLOCKVOLUMEPREFIX = filepath.Join("/var/lib/kubelet", "/plugins/kubernetes.io/csi/volumeDevices/publish")
)

func NewNodeSvc(driver *Driver, mounter Mounter, cloud Cloud) *NodeSvc {
	return &NodeSvc{
		d:        driver,
		mounter:  mounter,
		cloud:    cloud,
		inFlight: inflight.NewInFlight(),
	}
}

func (ns *NodeSvc) NodeStageVolume(_ context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	// check arguments
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	if ok := ns.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodeStageVolume: volume operation finished", "volumeID", volumeID)
		ns.inFlight.Delete(volumeID)
	}()

	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}
	if !ns.isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	// If the access type is block, do nothing for stage
	switch volCap.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		return &csi.NodeStageVolumeResponse{}, nil
	}

	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "staging target volumePath missing in request")
	}

	devicePath := ns.cloud.DevicePathByVolId(volumeID)
	if devicePath == "" {
		return nil, status.Error(codes.NotFound, "device cannot be found")
	}

	if err := util.CheckDeviceAvailable(devicePath); err != nil {
		klog.Errorf("volume %s check device failed: %s", volumeID, err)
		return nil, status.Errorf(codes.FailedPrecondition, "check device failed: %s", err)
	}

	opts := req.GetVolumeCapability().GetMount().GetMountFlags()

	if err := ns.createTargetMountPoint(targetPath, false); err != nil {
		return nil, status.Errorf(codes.Internal, "createTargetMountPoint error: %s", err.Error())
	}

	// Check if a device is mounted in target directory
	device, _, err := ns.mounter.GetDeviceNameFromMount(targetPath)
	if err != nil {
		msg := fmt.Sprintf("failed to check if volume is already mounted: %v", err)
		return nil, status.Error(codes.Internal, msg)
	}
	// This operation (NodeStageVolume) MUST be idempotent.
	// If the volume corresponding to the volume_id is already staged to the staging_target_path,
	// and is identical to the specified volume_capability the Plugin MUST reply 0 OK.
	if device == devicePath {
		klog.InfoS("NodeStageVolume: volume already staged", "volumeID", volumeID)
		return &csi.NodeStageVolumeResponse{}, nil
	}

	fsType := "ext4"
	if req.GetVolumeCapability().GetMount().FsType != "" {
		fsType = req.GetVolumeCapability().GetMount().FsType
	}
	err = ns.mounter.FormatAndMount(devicePath, targetPath, fsType, opts)
	if err != nil {
		errMsg := fmt.Sprintf("failed to mount device path (%s) to staging path (%s) for volume (%s), error: %s",
			devicePath,
			targetPath,
			volumeID,
			err)
		klog.Error(errMsg)
		return nil, status.Error(codes.Internal, errMsg)
	}

	if req.GetVolumeContext()[consts.SnapshotID] != "" {
		klog.Infof("NodeStageVolume: pv %s is created from snapshot, add resizefs check", volumeID)
		r := mount.NewResizeFs(exec.New())
		needResize, err := r.NeedResize(devicePath, targetPath)
		if err != nil {
			klog.Infof("NodeStageVolume: Could not determine if volume %s need to be resized: %v", volumeID, err)
			return &csi.NodeStageVolumeResponse{}, nil
		}
		if needResize {
			klog.Infof("NodeStageVolume: Resizing volume %s created from a snapshot", volumeID)
			if _, err := r.Resize(devicePath, targetPath); err != nil {
				klog.Errorf("NodeStageVolume: Resizing volume %s created from a snapshot failed: %s", volumeID, err)
				return nil, status.Errorf(codes.Internal, "Resizing volume %s created from a snapshot failed: %s", volumeID, err)
			}
		}
	}

	klog.Infof("stage volume %s success", volumeID)

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *NodeSvc) NodeUnstageVolume(_ context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	// check arguments
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	if ok := ns.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodeStageVolume: volume operation finished", "volumeID", volumeID)
		ns.inFlight.Delete(volumeID)
	}()
	targetPath := req.GetStagingTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "unstaging target volumePath missing in request")
	}
	// check target is mounted
	isNotMnt, err := mount.IsNotMountPoint(ns.mounter, targetPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, status.Errorf(codes.Internal, "check targetPath IsNotMountPoint err: %s", err)
	}
	if !isNotMnt {
		if err = ns.mounter.Unmount(targetPath); err != nil {
			klog.Errorf("volume %s failed to unmount targetPath %s with error: %v", req.VolumeId, targetPath, err)
			return nil, status.Errorf(codes.Internal, "failed to unmount targetPath %s with error: %v", targetPath, err)
		}
	}
	klog.Infof("successfully unmounted volume (%s) from staging path (%s)", req.GetVolumeId(), targetPath)

	if err = os.Remove(targetPath); err != nil {
		if !os.IsNotExist(err) {
			klog.Errorf("volume %s failed to remove staging target path (%s): (%v)", req.VolumeId, targetPath, err)
			return nil, status.Errorf(codes.Internal, "failed to remove staging target path (%s): (%v)", targetPath, err)
		}
	}
	klog.Infof("successfully remove staging target path (%s)", targetPath)

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *NodeSvc) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	// check arguments
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	if ok := ns.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodeStageVolume: volume operation finished", "volumeID", volumeID)
		ns.inFlight.Delete(volumeID)
	}()
	source := req.GetStagingTargetPath()
	if len(source) == 0 {
		return nil, status.Error(codes.InvalidArgument, "staging target missing in request")
	}
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target volumePath missing in request")
	}
	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}
	if !ns.isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	isBlock := volCap.GetBlock() != nil
	if err := ns.createTargetMountPoint(targetPath, isBlock); err != nil {
		return nil, status.Errorf(codes.Internal, "createTargetMountPoint error: %s", err.Error())
	}

	// check if targetPath mounted
	isNotMnt, err := ns.mounter.IsLikelyNotMountPoint(targetPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check targetPath IsNotMountPoint err: %s", err)
	}
	if !isNotMnt {
		klog.InfoS("NodePublishVolume: volume already published", "volumeID", volumeID)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	opts := volCap.GetMount().GetMountFlags()
	if req.GetReadonly() {
		opts = append(opts, "ro")
	}
	opts = append(opts, "bind")

	if isBlock {
		devicePath := ns.cloud.DevicePathByVolId(volumeID)
		if devicePath == "" {
			return nil, status.Error(codes.NotFound, "device cannot be found")
		}
		if err := ns.mounter.Mount(devicePath, targetPath, "", opts); err != nil {
			klog.Errorf("publish volume (%s) error: %v", volumeID, err)
			return nil, status.Error(codes.Internal, "publish volume failed")
		}
	} else {
		// check if source mounted
		isNotMnt, err = mount.IsNotMountPoint(ns.mounter, source)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "check source %s IsNotMountPoint err: %s", source, err)
		}
		if isNotMnt {
			return nil, status.Errorf(codes.Internal, "source %s not mounted", source)
		}
		fsType := "ext4"
		if fs := volCap.GetMount().FsType; fs != "" {
			fsType = fs
		}
		if err := ns.mounter.Mount(source, req.GetTargetPath(), fsType, opts); err != nil {
			klog.Errorf("publish volume (%s) error: %v", volumeID, err)
			return nil, status.Error(codes.Internal, "publish volume failed")
		}
	}
	klog.Infof("publish volume %s success", volumeID)
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *NodeSvc) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id is missing")
	}
	if ok := ns.inFlight.Insert(volumeID); !ok {
		return nil, status.Errorf(codes.Aborted, "An operation with the given Volume %s already exists", volumeID)
	}
	defer func() {
		klog.V(4).InfoS("NodeStageVolume: volume operation finished", "volumeID", volumeID)
		ns.inFlight.Delete(volumeID)
	}()
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "target volumePath is missing")
	}

	if err := mount.CleanupMountPoint(targetPath, ns.mounter, true); err != nil {
		klog.Errorf("failed to cleanup targetPath: %s with error: %v", targetPath, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	klog.Infof("unpublish volume %s success", volumeID)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *NodeSvc) NodeGetVolumeStats(_ context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	if len(req.VolumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume ID was empty")
	}
	if len(req.VolumePath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeGetVolumeStats volume path was empty")
	}

	exists, err := mount.PathExists(req.VolumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unknown error when stat on %s: %v", req.VolumePath, err)
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "path %s does not exist", req.VolumePath)
	}

	return util.GetMetrics(req.VolumePath)
}

func (ns *NodeSvc) NodeExpandVolume(_ context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	klog.Infof("NodeExpandVolume: node expand volume: %v", req)
	volumeId := req.GetVolumeId()
	if len(volumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID is empty")
	}
	volumePath := req.GetVolumePath()
	if len(volumePath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume path is empty")
	}
	devicePath := ns.cloud.DevicePathByVolId(volumeId)
	if devicePath == "" {
		klog.Errorf("NodeExpandVolume: can't get devicePath: %s", volumeId)
		return nil, status.Errorf(codes.NotFound, "can't get devicePath for %s", volumeId)
	}
	volExpandBytes := req.GetCapacityRange().GetRequiredBytes()
	requestGB := float64((volExpandBytes + types.GB - 1) / types.GB)

	volumeCapability := req.GetVolumeCapability()
	if volumeCapability != nil {
		if blk := volumeCapability.GetBlock(); blk != nil {
			// Noop for Block NodeExpandVolume
			klog.V(4).Infof("NodeExpandVolume called for %v at %s. Since it is a block device, ignoring...", volumeId, volumePath)
			return &csi.NodeExpandVolumeResponse{}, nil
		}
	}

	if strings.Contains(volumePath, BLOCKVOLUMEPREFIX) {
		klog.Infof("NodeExpandVolume: Block Volume not Expand FS, volumeId: %s, volumePath: %s", volumeId, volumePath)
		return &csi.NodeExpandVolumeResponse{}, nil
	}

	isBlock, err := ns.isBlockDevice(volumePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to determine if volumePath [%v] is a block device: %v", volumePath, err)
	}
	if isBlock {
		klog.Infof("NodeExpandVolume: Block Volume not Expand FS, volumeId: %s, volumePath: %s", volumeId, volumePath)
		return &csi.NodeExpandVolumeResponse{}, nil
	}

	// use resizer to expand volume filesystem
	resizer := mount.NewResizeFs(exec.New())
	if _, err := resizer.Resize(devicePath, volumePath); err != nil {
		klog.Errorf("NodeExpandVolume: Resize Error, volumeId: %s, devicePath: %s, volumePath: %s, err: %s", volumeId, devicePath, volumePath, err.Error())
		return nil, status.Errorf(codes.Internal, "Could not resize volume %s: %v", req.VolumeId, err)
	}

	diskCapacity, err := getDiskCapacity(volumePath)
	if err != nil {
		klog.Errorf("NodeExpandVolume: volume %s get diskCapacity of %s error %v", volumeId, volumePath, err)
		return nil, status.Errorf(codes.Internal, "get diskCapacity of %s error %v", volumePath, err)
	}
	if diskCapacity >= requestGB*filesystemLosePercent {
		klog.Infof("NodeExpandVolume: resizefs successful volumeId: %s, devicePath: %s, volumePath: %s", volumeId, devicePath, volumePath)
		return &csi.NodeExpandVolumeResponse{}, nil
	}
	return nil, status.Errorf(codes.Internal, "requestGB: %v, diskCapacity: %v not in range", requestGB, diskCapacity)
}

func (ns *NodeSvc) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.d.NSCap,
	}, nil
}

func (ns *NodeSvc) NodeGetInfo(_ context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	resp := &csi.NodeGetInfoResponse{
		NodeId:             ns.d.NodeID,
		MaxVolumesPerNode:  ns.d.maxVolumesPerNode,
		AccessibleTopology: &csi.Topology{Segments: ns.cloud.Topology()},
	}
	return resp, nil
}

func (ns *NodeSvc) createTargetMountPoint(mountPath string, isBlock bool) error {
	if isBlock {
		fi, err := os.Lstat(mountPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && fi.IsDir() {
			klog.Warningf("remove %s", mountPath)
			os.Remove(mountPath)
		}
		pathFile, err := os.OpenFile(mountPath, os.O_CREATE|os.O_RDWR, 0777)
		klog.Warningf("create %s", mountPath)
		if err != nil {
			klog.Errorf("failed to create mountPath:%s with error: %v", mountPath, err)
			return status.Error(codes.Internal, err.Error())
		}
		if err = pathFile.Close(); err != nil {
			klog.Errorf("failed to close mountPath:%s with error: %v", mountPath, err)
			return status.Error(codes.Internal, err.Error())
		}

		return nil
	}

	return util.CreateDest(mountPath)
}

// IsBlock checks if the given path is a block device
func (ns *NodeSvc) isBlockDevice(fullPath string) (bool, error) {
	var st unix.Stat_t
	err := unix.Stat(fullPath, &st)
	if err != nil {
		return false, err
	}

	return (st.Mode & unix.S_IFMT) == unix.S_IFBLK, nil
}

func (ns *NodeSvc) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range ns.d.Cap {
			if c.GetMode() == cap.AccessMode.GetMode() {
				return true
			}
		}
		return false
	}

	foundAll := true
	for _, c := range volCaps {
		if !hasSupport(c) {
			foundAll = false
			break
		}
	}
	return foundAll
}
