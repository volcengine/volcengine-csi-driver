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

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/consts"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"
	"github.com/volcengine/volcengine-csi-driver/pkg/util/inflight"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/klog/v2"
)

const (
	defaultCapacity       = 20 * types.GB
	volumeTypeKey         = "type"
	volumeTypeDefault     = "ESSD_PL0"
	zoneIdKey             = "zone"
	filesystemLosePercent = 0.9
	nodeVolumesNumLimit   = 15
)

type ControllerSvc struct {
	d        *Driver
	cloud    Cloud
	inFlight *inflight.InFlight
}

func NewControllerSvc(driver *Driver, cloud Cloud) *ControllerSvc {
	return &ControllerSvc{
		d:        driver,
		cloud:    cloud,
		inFlight: inflight.NewInFlight(),
	}
}

func (cs *ControllerSvc) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	volName := req.GetName()
	if len(volName) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume name missiong in request")
	}
	if len(req.GetVolumeCapabilities()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume capabilities missing in request")
	}
	// check if a request is already in-flight
	if ok := cs.inFlight.Insert(volName); !ok {
		msg := fmt.Sprintf("Create volume request for %s is already in progress", volName)
		return nil, status.Error(codes.Aborted, msg)
	}
	defer cs.inFlight.Delete(volName)

	capacity := defaultCapacity
	if cr := req.GetCapacityRange(); cr != nil && cr.GetRequiredBytes() != 0 {
		capacity = cr.GetRequiredBytes()
	}

	zoneId := getZoneId(req)
	if zoneId == "" {
		klog.Errorf("CreateVolume %s: Can't get zoneId, please check your setup or set zone ID in storage class. Use zone from Meta service", req.Name)
		zoneId = cs.cloud.Zone()
		if zoneId == "" {
			klog.Errorf("CreateVolume %s: get zoneId from metadata service failed", req.Name)
			return nil, status.Error(codes.Internal, "get zoneId from metadata service failed")
		}
	}
	volumeType := getVolumeTypeWithDefault(req)

	snapshotID := ""
	volumeSource := req.GetVolumeContentSource()
	if volumeSource != nil {
		if _, ok := volumeSource.GetType().(*csi.VolumeContentSource_Snapshot); !ok {
			return nil, status.Error(codes.InvalidArgument, "Unsupported volumeContentSource type")
		}
		sourceSnapshot := volumeSource.GetSnapshot()
		if sourceSnapshot == nil {
			return nil, status.Error(codes.InvalidArgument, "Error retrieving snapshot from the volumeContentSource")
		}
		snapshotID = sourceSnapshot.GetSnapshotId()
	}
	volumeContext := map[string]string{}
	volumeContext[consts.SnapshotID] = snapshotID

	vol, err := cs.cloud.VolumeByName(ctx, req.GetName())
	if err != nil {
		klog.Error("get volume by name %s error: %v", req.Name, err)
		return nil, status.Errorf(codes.FailedPrecondition, "get volume by name error: %v", err)
	}
	if vol != nil {
		if !vol.Available() {
			klog.Errorf("volume %s status is not available", req.Name)
			return nil, status.Errorf(codes.FailedPrecondition, "the volume which get by name isn't available")
		}
		if vol.VolumeType != volumeType {
			klog.Errorf("volume %s existed with different type, expect: %s, existed: %s ", req.Name, volumeType, vol.ZoneId)
			return nil, status.Error(codes.AlreadyExists, "already exists volume with different type")
		}
		if vol.Capacity < capacity {
			klog.Errorf("volume %s existed with smaller capacity, existed %v < acquire %v", req.Name, vol.Capacity, capacity)
			return nil, status.Error(codes.AlreadyExists, "already exists volume with smaller capacity")
		}
		if vol.ZoneId != zoneId {
			klog.Errorf("volume %s existed with different zoneId, expect: %s, existed: %s ", req.Name, zoneId, vol.ZoneId)
			return nil, status.Error(codes.AlreadyExists, "already exists volume with different zoneId")
		}
		resp := &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				VolumeId:      vol.Id,
				CapacityBytes: vol.Capacity,
				VolumeContext: volumeContext,
				ContentSource: req.GetVolumeContentSource(),
				AccessibleTopology: []*csi.Topology{
					{
						Segments: map[string]string{
							consts.TopologyRegionKey: cs.cloud.Region(),
							consts.TopologyZoneKey:   vol.ZoneId,
						},
					},
				},
			},
		}
		return resp, nil
	}

	// if volume is not already existed
	volID, err := cs.cloud.CreateVolume(ctx, req.GetName(), volumeType, zoneId, snapshotID, capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume %v: %w", req.GetName(), err)
	}
	klog.Infof("create volume %s success, volID: %s", req.Name, volID)

	csiVol := &csi.Volume{
		VolumeId:      volID,
		CapacityBytes: capacity,
		VolumeContext: volumeContext,
		ContentSource: req.GetVolumeContentSource(),
		AccessibleTopology: []*csi.Topology{
			{
				Segments: map[string]string{
					consts.TopologyRegionKey: cs.cloud.Region(),
					consts.TopologyZoneKey:   zoneId,
				},
			},
		},
	}

	return &csi.CreateVolumeResponse{Volume: csiVol}, nil
}

func (cs *ControllerSvc) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	// check arguments
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	// check if a request is already in-flight
	if ok := cs.inFlight.Insert(volumeID); !ok {
		msg := fmt.Sprintf("An operation with the given Volume %s already exists", volumeID)
		return nil, status.Error(codes.Aborted, msg)
	}
	defer cs.inFlight.Delete(volumeID)

	vol, err := cs.cloud.VolumeById(ctx, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "test volume existed error %v", err)
	}
	if vol == nil {
		return &csi.DeleteVolumeResponse{}, nil
	}

	// cs.e.deleteVolume return nil error when p response http.StatusNotFound
	if err = cs.cloud.DeleteVolume(ctx, volumeID); err != nil {
		klog.Errorf("failed to delete volume %v: %w", volumeID, err)
		return nil, fmt.Errorf("failed to delete volume %v: %w", volumeID, err)
	}
	klog.Infof("volume %s delete success", volumeID)

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerSvc) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	if len(req.GetNodeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "node id missing in request")
	}
	volCap := req.GetVolumeCapability()
	if volCap == nil {
		return nil, status.Error(codes.InvalidArgument, "volume capability missing in request")
	}
	if !cs.isValidVolumeCapabilities([]*csi.VolumeCapability{volCap}) {
		return nil, status.Error(codes.InvalidArgument, "Volume capability not supported")
	}

	vol, err := cs.cloud.VolumeById(ctx, req.GetVolumeId())
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "get volume by id failed: %v", err)
	}
	if vol == nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	if vol.Attached() {
		if vol.NodeId == req.GetNodeId() {
			resp := &csi.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{},
			}
			return resp, nil
		} else {
			klog.Errorf("volume %s is attached to another node", req.VolumeId)
			return nil, status.Error(codes.FailedPrecondition, "volume is attached to another node")
		}
	}

	if !vol.Available() {
		klog.Errorf("volume %s status is not available", req.VolumeId)
		return nil, status.Error(codes.FailedPrecondition, "volume status is not available")
	}

	nodeInfo, err := cs.cloud.NodeById(ctx, req.GetNodeId())
	if err != nil {
		klog.Errorf("get node %s by id for volume %s failed: %s", req.NodeId, req.VolumeId, err)
		return nil, status.Errorf(codes.NotFound, "get node by id %s failed", req.NodeId)
	}
	if len(nodeInfo.Volumes) >= nodeVolumesNumLimit {
		return nil, status.Errorf(codes.FailedPrecondition, "volume attached to the node exceed limit %v", nodeVolumesNumLimit)
	}
	// check volumeType
	volumeTypeMatch := false
	for _, volumeType := range nodeInfo.InstanceType.VolumeTypes {
		if *volumeType == vol.VolumeType {
			volumeTypeMatch = true
			break
		}
	}
	if !volumeTypeMatch {
		klog.Errorf("volume: %s, node: %s, volumeType misMatch, volume type: %s, node support types: %+v", req.VolumeId, req.NodeId, vol.VolumeType, nodeInfo.InstanceType.VolumeTypes)
		return nil, status.Errorf(codes.FailedPrecondition, "volumeType misMatch, volume type: %s, node support types: %+v", vol.VolumeType, nodeInfo.InstanceType.VolumeTypes)
	}

	err = cs.cloud.AttachVolume(ctx, req.GetNodeId(), req.GetVolumeId())
	if err != nil {
		klog.Errorf("attach volume %s to node %s failed: %s", req.VolumeId, req.NodeId, err)
		return nil, status.Errorf(codes.Internal, "attach volume failed: %v", err)
	}

	resp := &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{},
	}
	return resp, nil
}

func (cs *ControllerSvc) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id missing in request")
	}
	if len(req.GetNodeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "node id missing in request")
	}

	vol, err := cs.cloud.VolumeById(ctx, req.GetVolumeId())
	if err != nil {
		klog.Errorf("get volume %s by id failed: %s", req.VolumeId, err)
		return nil, status.Errorf(codes.FailedPrecondition, "get volume by id failed: %v", err)
	}

	// that means volume is not exist or detach success
	if vol == nil || vol.Available() || vol.Deleted() || vol.NodeId != req.GetNodeId() {
		return &csi.ControllerUnpublishVolumeResponse{}, nil
	}

	if vol.Detaching() {
		klog.Errorf("volume %s is detaching", req.VolumeId)
		return nil, status.Error(codes.Internal, "volume is detaching")
	}

	err = cs.cloud.DetachVolume(ctx, req.GetNodeId(), req.GetVolumeId())
	if err != nil {
		klog.Error("detach volume %s from node %s failed: %s", req.VolumeId, req.NodeId, err)
		return nil, status.Errorf(codes.Internal, "detach volume failed: %v", err)
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (cs *ControllerSvc) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	// check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id cannot be empty")
	}
	volCaps := req.GetVolumeCapabilities()
	if len(volCaps) == 0 {
		return nil, status.Error(codes.InvalidArgument, "capabilities of request cannot be empty")
	}

	vol, err := cs.cloud.VolumeById(ctx, req.GetVolumeId())
	if err != nil {
		klog.Errorf("get volume %s by id failed: %s", req.VolumeId, err)
		return nil, status.Errorf(codes.FailedPrecondition, "get volume by id failed: %v", err)
	}
	if vol == nil {
		return nil, status.Error(codes.NotFound, "volume not found")
	}

	if !cs.isValidVolumeCapabilities(volCaps) {
		return &csi.ValidateVolumeCapabilitiesResponse{}, nil
	}

	resp := &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext:      req.GetVolumeContext(),
			VolumeCapabilities: req.GetVolumeCapabilities(),
			Parameters:         req.GetParameters(),
		},
	}
	return resp, nil
}

func (cs *ControllerSvc) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerSvc) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerSvc) ControllerGetCapabilities(_ context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.d.CSCap,
	}, nil
}

func (cs *ControllerSvc) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	klog.Infof("CreateSnapshot: called with args %+v", req)
	if err := validateCreateSnapshotRequest(req); err != nil {
		return nil, err
	}

	snapshotName := req.GetName()
	volumeID := req.GetSourceVolumeId()

	// check if a request is already in-flight
	if ok := cs.inFlight.Insert(snapshotName); !ok {
		msg := fmt.Sprintf(inflight.VolumeOperationAlreadyExistsErrorMsg, snapshotName)
		return nil, status.Error(codes.Aborted, msg)
	}
	defer cs.inFlight.Delete(snapshotName)

	snapshot, err := cs.cloud.GetSnapshotByName(ctx, snapshotName)
	if err != nil {
		klog.Errorf("Error looking for the snapshot %s: %v", snapshotName, err)
		return nil, err
	}
	if snapshot != nil {
		if snapshot.SourceVolumeID != volumeID {
			return nil, status.Errorf(codes.AlreadyExists, "Snapshot %s already exists for different volume (%s)", snapshotName, snapshot.SourceVolumeID)
		}
		klog.V(4).Infof("Snapshot %s of volume %s already exists; nothing to do", snapshotName, volumeID)
		return newCreateSnapshotResponse(snapshot), nil
	}

	vol, err := cs.cloud.VolumeById(ctx, volumeID)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "get source volume by id failed: %v", err)
	}
	if vol == nil {
		return nil, status.Error(codes.NotFound, "source volume not found")
	}

	snapshot, err = cs.cloud.CreateSnapshot(ctx, volumeID, snapshotName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create snapshot %q: %v", snapshotName, err)
	}
	snapshot.Size = vol.Capacity
	snapshot.SourceVolumeID = volumeID
	return newCreateSnapshotResponse(snapshot), nil
}

func validateCreateSnapshotRequest(req *csi.CreateSnapshotRequest) error {
	if len(req.GetName()) == 0 {
		return status.Error(codes.InvalidArgument, "Snapshot name not provided")
	}

	if len(req.GetSourceVolumeId()) == 0 {
		return status.Error(codes.InvalidArgument, "Snapshot volume source ID not provided")
	}
	return nil
}

func (cs *ControllerSvc) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	klog.Infof("DeleteSnapshot: called with args %+v", req)
	if err := validateDeleteSnapshotRequest(req); err != nil {
		return nil, err
	}
	snapshotID := req.GetSnapshotId()
	// check if a request is already in-flight
	if ok := cs.inFlight.Insert(snapshotID); !ok {
		msg := fmt.Sprintf("DeleteSnapshot for Snapshot %s is already in progress", snapshotID)
		return nil, status.Error(codes.Aborted, msg)
	}
	defer cs.inFlight.Delete(snapshotID)

	if err := cs.cloud.DeleteSnapshot(ctx, snapshotID); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not delete snapshot ID %q: %v", snapshotID, err)
	}
	return &csi.DeleteSnapshotResponse{}, nil
}

func validateDeleteSnapshotRequest(req *csi.DeleteSnapshotRequest) error {
	if len(req.GetSnapshotId()) == 0 {
		return status.Error(codes.InvalidArgument, "Snapshot ID not provided")
	}
	return nil
}

func (cs *ControllerSvc) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	klog.Infof("ListSnapshots: called with args %+v", req)
	snapshotID := req.GetSnapshotId()
	if len(snapshotID) != 0 {
		snapshot, err := cs.cloud.GetSnapshotByID(ctx, snapshotID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get snapshot ID %q: %v", snapshotID, err)
		}
		if snapshot == nil {
			return &csi.ListSnapshotsResponse{}, nil
		}
		return &csi.ListSnapshotsResponse{
			Entries: []*csi.ListSnapshotsResponse_Entry{{Snapshot: newCSISnapshot(snapshot)}},
		}, nil
	}
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerSvc) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	klog.Infof("ControllerExpandVolume: Starting expand Volume with: %v", req)
	// check arguments
	volumeId := req.GetVolumeId()
	if len(volumeId) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	volSizeBytes := req.GetCapacityRange().GetRequiredBytes()

	// check resize conditions
	ebsVolume, err := cs.cloud.VolumeById(ctx, volumeId)
	if err != nil {
		klog.Error("ControllerExpandVolume: describe volume %s failed: %s", volumeId, err)
		return nil, status.Errorf(codes.Internal, "describe volume %s failed: %s", volumeId, err)
	}
	if ebsVolume == nil {
		klog.Errorf("ControllerExpandVolume: volume %s not exist", volumeId)
		return nil, status.Error(codes.Internal, "expand volume not exist")
	}
	requestGb := (volSizeBytes + types.GB - 1) / types.GB // round up
	if requestGb*types.GB == ebsVolume.Capacity {
		klog.Infof("ControllerExpandVolume %s: expect size is same with current size: %dGi", volumeId, requestGb)
		return &csi.ControllerExpandVolumeResponse{CapacityBytes: ebsVolume.Capacity, NodeExpansionRequired: true}, nil
	}
	if volSizeBytes < ebsVolume.Capacity {
		klog.Infof("ControllerExpandVolume %s: expect size is less than current size: %d, expect size: %d", volumeId, ebsVolume.Capacity, volSizeBytes)
		return &csi.ControllerExpandVolumeResponse{CapacityBytes: ebsVolume.Capacity, NodeExpansionRequired: true}, nil
	}

	// expend volume
	err = cs.cloud.ExtendVolume(ctx, volumeId, volSizeBytes)
	if err != nil {
		klog.Errorf("ControllerExpandVolume: expend volume %s failed: %s", volumeId, err)
		return nil, status.Errorf(codes.Internal, "expend volume failed: %s", err)
	}
	klog.Infof("expend volume %s success", volumeId)
	return &csi.ControllerExpandVolumeResponse{CapacityBytes: volSizeBytes, NodeExpansionRequired: true}, nil
}

func (cs *ControllerSvc) ControllerGetVolume(_ context.Context, _ *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func getVolumeTypeWithDefault(req *csi.CreateVolumeRequest) string {
	params := req.GetParameters()
	if params == nil {
		return volumeTypeDefault
	}

	if volumeType, ok := params[volumeTypeKey]; ok {
		return volumeType
	}
	return volumeTypeDefault
}

func getZoneId(req *csi.CreateVolumeRequest) string {
	params := req.GetParameters()
	if params != nil {
		if zoneId := params[zoneIdKey]; zoneId != "" {
			return zoneId
		}
	}

	return pickZone(req.GetAccessibilityRequirements())
}

// pickZone selects 1 zone given topology requirement.
// if not found, empty string is returned.
func pickZone(requirement *csi.TopologyRequirement) string {
	if requirement == nil {
		return ""
	}
	for _, topology := range requirement.GetPreferred() {
		zone, exists := topology.GetSegments()[consts.TopologyZoneKey]
		if exists {
			return zone
		}
	}
	for _, topology := range requirement.GetRequisite() {
		zone, exists := topology.GetSegments()[consts.TopologyZoneKey]
		if exists {
			return zone
		}
	}
	return ""
}

func newCreateSnapshotResponse(snapshot *types.Snapshot) *csi.CreateSnapshotResponse {
	return &csi.CreateSnapshotResponse{Snapshot: newCSISnapshot(snapshot)}
}
func newCSISnapshot(snapshot *types.Snapshot) *csi.Snapshot {
	return &csi.Snapshot{
		SnapshotId:     snapshot.SnapshotID,
		SourceVolumeId: snapshot.SourceVolumeID,
		SizeBytes:      snapshot.Size,
		CreationTime:   timestamppb.New(snapshot.CreationTime),
		ReadyToUse:     snapshot.ReadyToUse,
	}
}

func (cs *ControllerSvc) isValidVolumeCapabilities(volCaps []*csi.VolumeCapability) bool {
	hasSupport := func(cap *csi.VolumeCapability) bool {
		for _, c := range cs.d.Cap {
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
