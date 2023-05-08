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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/storageebs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/universal"
	"k8s.io/klog/v2"

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/consts"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"
	"github.com/volcengine/volcengine-csi-driver/pkg/sts"
	"github.com/volcengine/volcengine-csi-driver/pkg/util"
)

type VolcEngin struct {
	svcClients *sts.ServiceClients
	region     string
	zone       string
}

func NewVolcEngin(svcClients *sts.ServiceClients, region, zone string) *VolcEngin {
	return &VolcEngin{
		svcClients: svcClients,
		region:     region,
		zone:       zone,
	}
}

var _ Cloud = &VolcEngin{}

func (volc *VolcEngin) CreateVolume(ctx context.Context, name, volumeType, zoneId, snapshotId string, capacity int64) (id string, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "CreateVolume",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	if capacity%types.GB != 0 {
		klog.Warningf("request capacity cannot be divied by GB: %d %% %d != 0 ", capacity, types.GB)
	}
	sizeGB := (capacity + types.GB - 1) / types.GB // round up
	req := &types.CreateVolumeInput{
		VolumeName: &name,
		VolumeType: &volumeType,
		Kind:       volcengine.String("data"),
		Size:       volcengine.JsonNumber(strconv.FormatInt(sizeGB, 10)),
		ZoneId:     &zoneId,
		SnapshotId: &snapshotId,
	}
	resp := &types.CreateVolumeOutput{}
	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return "", fmt.Errorf("create volume error: %w, req: %v", err, req)
	}

	if resp.VolumeId == nil {
		return "", errors.New("created volume failed from ebs with nil VolumeId")
	}

	if resp.Metadata.Error != nil {
		return "", fmt.Errorf("created volume error: %v, req: %v", resp.Metadata.Error, req)
	}

	if err = volc.WaitVolumeBeCreated(ctx, *resp.VolumeId); err != nil {
		return "", fmt.Errorf("poll volume to be created timeout: %w", err)
	}

	if err = volc.WaitVolumeBeStatus(ctx, *resp.VolumeId, types.EBSStatusAvailable); err != nil {
		return "", fmt.Errorf("poll volume status to be available failed: %w", err)
	}
	return *resp.VolumeId, nil
}

func (volc *VolcEngin) ExtendVolume(ctx context.Context, id string, newSize int64) (err error) {
	if newSize%types.GB != 0 {
		klog.Warningf("request capacity cannot be divied by GB: %d %% %d != 0 ", newSize, types.GB)
	}
	sizeGB := (newSize + types.GB - 1) / types.GB // round up

	req := &storageebs.ExtendVolumeInput{
		VolumeId: &id,
		NewSize:  volcengine.JsonNumber(strconv.FormatInt(sizeGB, 10)),
	}
	resp, err := volc.svcClients.EbsClient.ExtendVolumeWithContext(ctx, req)

	if err != nil {
		return fmt.Errorf("extend volume error: %w, req: %v", err, req)
	}

	if resp == nil {
		return errors.New("extend volume failed from ebs with nil resp")
	}

	if resp.Metadata.Error != nil {
		return fmt.Errorf("extend volume error: %v, req: %v", resp.Metadata.Error, req)
	}

	err = volc.WaitVolumeBeExtended(ctx, id, sizeGB*types.GB)
	if err != nil {
		return fmt.Errorf("check extend volume failed: %w", err)
	}

	return
}

func (volc *VolcEngin) DeleteVolume(ctx context.Context, id string) error {
	resp, err := volc.svcClients.EbsClient.DeleteVolumeWithContext(ctx, &storageebs.DeleteVolumeInput{
		VolumeId: &id,
	})

	if err != nil {
		if resp != nil && resp.Metadata != nil && resp.Metadata.HTTPCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("delete volume by id %s failed: %w", id, err)
	}

	if resp != nil && resp.Metadata.Error != nil {
		return fmt.Errorf("delete volume error: %v, id: %v", resp.Metadata.Error, id)
	}

	if err = volc.WaitVolumeBeStatus(ctx, id, types.EBSStatusDeleted); err != nil {
		return fmt.Errorf("poll volume status to be deleted failed: %w", err)
	}

	return nil
}

func (volc *VolcEngin) DevicePathByVolId(volId string) string {
	devicePath, err := util.GetDeviceByVolumeID(volId)
	if err != nil {
		klog.Errorf("get device path by volume id %s failed: %v", volId, err)
		return ""
	}
	return devicePath
}

func (volc *VolcEngin) AttachVolume(ctx context.Context, nodeId, volId string) error {
	resp, err := volc.svcClients.EbsClient.AttachVolumeWithContext(ctx, &storageebs.AttachVolumeInput{
		InstanceId: &nodeId,
		VolumeId:   &volId,
	})
	if err != nil {
		return fmt.Errorf("attach volume %v to instance %v error: %w", volId, nodeId, err)
	}

	if resp != nil && resp.Metadata.Error != nil {
		return fmt.Errorf("attach volume %v to instance %v error: %v", volId, nodeId, resp.Metadata.Error)
	}

	if err = volc.WaitVolumeBeStatus(ctx, volId, types.EBSStatusAttached); err != nil {
		return fmt.Errorf("poll volume status to be attached failed: %w", err)
	}

	return nil
}

func (volc *VolcEngin) DetachVolume(ctx context.Context, nodeId, volId string) error {
	resp, err := volc.svcClients.EbsClient.DetachVolumeWithContext(ctx, &storageebs.DetachVolumeInput{
		InstanceId: &nodeId,
		VolumeId:   &volId,
	})
	if err != nil {
		return fmt.Errorf("detach volume %v to instance %v error: %w", volId, nodeId, err)
	}

	if resp != nil && resp.Metadata.Error != nil {
		return fmt.Errorf("detach volume %v to instance %v error: %v", volId, nodeId, resp.Metadata.Error)
	}

	err = volc.WaitVolumeBeStatus(ctx, volId, types.EBSStatusAvailable)
	if err != nil {
		return fmt.Errorf("poll volume status to be available failed: %w", err)
	}
	return nil
}

func (volc *VolcEngin) VolumeById(ctx context.Context, id string) (vol *types.Volume, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "DescribeVolumes",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeVolumesInput{
		VolumeIds: []*string{&id},
	}
	resp := &types.DescribeVolumesOutput{}
	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("get volume by id %s failed: %w", id, err)
	}

	if resp.Metadata == nil {
		return nil, errors.New("response from volc stack is nil")
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("get volume by id %s error: %v", id, resp.Metadata.Error)
	}

	if len(resp.Volumes) == 0 {
		return nil, nil
	}
	ebsVolume := resp.Volumes[0]
	if *ebsVolume.ErrorDetail != "" {
		return nil, fmt.Errorf("volume create failed with id %s: %v", id, ebsVolume.ErrorDetail)
	}

	sizeGB, err := ebsVolume.Size.Int64()
	if err != nil {
		return nil, fmt.Errorf("parse ebs volume size to int failed: %v", err)
	}

	vol = &types.Volume{
		Id:         *ebsVolume.VolumeId,
		Name:       *ebsVolume.VolumeName,
		Status:     types.StatusFromString(*ebsVolume.Status),
		Capacity:   sizeGB * types.GB,
		NodeId:     *ebsVolume.InstanceId,
		ZoneId:     *ebsVolume.ZoneId,
		VolumeType: *ebsVolume.VolumeType,
	}
	return vol, nil
}

func (volc *VolcEngin) VolumeByName(ctx context.Context, name string) (vol *types.Volume, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "DescribeVolumes",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeVolumesInput{
		VolumeName: &name,
		PageNumber: int32ToPtr(1),
		PageSize:   int32ToPtr(10),
	}
	resp := &types.DescribeVolumesOutput{}
	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("get volume by name %s failed: %w", name, err)
	}

	if resp.Metadata == nil {
		return nil, errors.New("response from volc stack is nil")
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("get volume by name %s error: %v", name, resp.Metadata.Error)
	}

	if len(resp.Volumes) == 0 {
		return nil, nil
	}
	if len(resp.Volumes) != 1 {
		klog.Warningf("total count from list volume resp is expected 1, but got %d", len(resp.Volumes))
	}

	ebsVolume := resp.Volumes[0]
	if *ebsVolume.ErrorDetail != "" {
		return nil, fmt.Errorf("volume create failed with name %s: %v", name, ebsVolume.ErrorDetail)
	}
	sizeGB, err := ebsVolume.Size.Int64()
	if err != nil {
		return nil, fmt.Errorf("parse ebs volume size to int failed: %v", err)
	}

	vol = &types.Volume{
		Id:         *ebsVolume.VolumeId,
		Name:       *ebsVolume.VolumeName,
		Status:     types.StatusFromString(*resp.Volumes[0].Status),
		Capacity:   sizeGB * types.GB,
		NodeId:     *ebsVolume.InstanceId,
		ZoneId:     *ebsVolume.ZoneId,
		VolumeType: *ebsVolume.VolumeType,
	}

	return vol, nil
}

func (volc *VolcEngin) CreateSnapshot(ctx context.Context, volumeID, snapshotName string) (snapshot *types.Snapshot, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "CreateSnapshot",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.CreateSnapshotInput{
		VolumeId:     &volumeID,
		SnapshotName: &snapshotName,
	}
	resp := &types.CreateSnapshotOutput{}

	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)

	if err != nil {
		return nil, fmt.Errorf("CreateSnapshot error: %w, req: %v", err, req)
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("CreateSnapshot error: %v, req: %v", resp.Metadata.Error, req)
	}
	return &types.Snapshot{
		SnapshotID:     *resp.SnapshotId,
		SourceVolumeID: volumeID,
	}, nil
}

func (volc *VolcEngin) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "DeleteSnapshot",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DeleteSnapshotInput{SnapshotId: &snapshotID}
	resp := &types.DeleteSnapshotOutput{}
	err := volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)

	if err != nil {
		if resp != nil && resp.Metadata != nil && resp.Metadata.HTTPCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("DeleteSnapshot %s error: %s", snapshotID, err)
	}

	if resp.Metadata.Error != nil {
		return fmt.Errorf("DeleteSnapshot %s error: %s", snapshotID, err)
	}
	return nil
}

func (volc *VolcEngin) GetSnapshotByName(ctx context.Context, name string) (snapshot *types.Snapshot, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "DescribeSnapshots",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeSnapshotsInput{
		SnapshotName: &name,
		PageNumber:   int32ToPtr(1),
		PageSize:     int32ToPtr(10),
	}
	resp := &types.DescribeSnapshotsOutput{}
	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("GetSnapshotByName %s failed: %w", name, err)
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("GetSnapshotByName %s error: %v", name, resp.Metadata.Error)
	}

	if len(resp.Snapshots) == 0 {
		return nil, nil
	}
	if len(resp.Snapshots) != 1 {
		klog.Warningf("total count from list volume resp is expected 1, but got %d", len(resp.Snapshots))
	}

	ebsSnapshot := resp.Snapshots[0]
	return volc.ebsSnapshotResponseToStruct(ebsSnapshot)
}

// Helper method converting ebs snapshot type to the internal struct
func (volc *VolcEngin) ebsSnapshotResponseToStruct(ebsSnapshot *types.SnapshotForDescribeSnapshotsOutput) (*types.Snapshot, error) {
	if ebsSnapshot == nil {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339, *ebsSnapshot.CreationTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse snapshot creation time: %s", *ebsSnapshot.CreationTime)
	}

	snapshot := &types.Snapshot{
		SnapshotID:     *ebsSnapshot.SnapshotId,
		SourceVolumeID: *ebsSnapshot.VolumeId,
		Size:           *ebsSnapshot.VolumeSize * types.GB,
		CreationTime:   t,
	}
	if *ebsSnapshot.Status == "available" {
		snapshot.ReadyToUse = true
	} else {
		snapshot.ReadyToUse = false
	}
	return snapshot, nil
}

func (volc *VolcEngin) GetSnapshotByID(ctx context.Context, snapshotID string) (snapshot *types.Snapshot, err error) {
	info := universal.RequestUniversal{
		ServiceName: "storage_ebs",
		Action:      "DescribeSnapshots",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeSnapshotsInput{
		SnapshotIds: []*string{&snapshotID},
	}
	resp := &types.DescribeSnapshotsOutput{}
	err = volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("GetSnapshotByID %s failed: %w", snapshotID, err)
	}

	if resp.Metadata.Error != nil {
		return nil, fmt.Errorf("GetSnapshotByID %s error: %v", snapshotID, resp.Metadata.Error)
	}

	if len(resp.Snapshots) == 0 {
		return nil, nil
	}
	ebsSnapshot := resp.Snapshots[0]
	return volc.ebsSnapshotResponseToStruct(ebsSnapshot)
}

func (volc *VolcEngin) WaitVolumeBeStatus(ctx context.Context, id, status string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			vol, err := volc.VolumeById(ctx, id)
			if err != nil {
				return err
			}
			if vol == nil {
				if status == "deleted" {
					return nil
				}
				return errors.New("volume cannot be found")
			}
			if types.StatusToString(vol.Status) == status {
				return nil
			}
			time.Sleep(500 * time.Millisecond) // TODO: change magic number
		}
	}
}

func (volc *VolcEngin) WaitVolumeBeCreated(ctx context.Context, id string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			vol, err := volc.VolumeById(ctx, id)
			if err == nil && vol != nil {
				// we should wait volume to be created, once volume was created, return.
				return nil
			}
			time.Sleep(500 * time.Millisecond) // TODO: change magic number
		}
	}
}

func (volc *VolcEngin) WaitVolumeBeExtended(ctx context.Context, id string, sizeBytes int64) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			vol, err := volc.VolumeById(ctx, id)
			if err == nil && vol.Capacity == sizeBytes {
				// we should wait volume to be extended to sizeBytes, once volume was created, return.
				return nil
			}
			time.Sleep(500 * time.Millisecond) // TODO: change magic number
		}
	}
}

func (volc *VolcEngin) NodeById(ctx context.Context, id string) (*types.InstanceForDescribeInstancesOutput, error) {
	info := universal.RequestUniversal{
		ServiceName: "ecs",
		Action:      "DescribeInstances",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeInstancesInput{
		InstanceIds: []*string{&id},
		PageNumber:  int32ToPtr(1),
		PageSize:    int32ToPtr(10),
	}
	resp := &types.DescribeInstancesOutput{}
	err := volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("get instance by id failed: %w", err)
	}

	if resp == nil || len(resp.Instances) == 0 {
		return nil, errors.New("not found")
	}
	return resp.Instances[0], nil
}

func (volc *VolcEngin) DescribeInstanceTypes(ctx context.Context, typeName string) (*types.InstanceTypeForDescribeInstanceTypesOutput, error) {
	info := universal.RequestUniversal{
		ServiceName: "ecs",
		Action:      "DescribeInstanceTypes",
		Version:     "2020-04-01",
		HttpMethod:  universal.GET,
	}
	req := &types.DescribeInstanceTypesInput{
		InstanceTypes: []*string{&typeName},
	}
	resp := &types.DescribeInstanceTypesOutput{}
	err := volc.svcClients.UniversalClient.DoCallWithType(info, req, resp)
	if err != nil {
		return nil, fmt.Errorf("get instance by id failed: %w", err)
	}

	if resp == nil || len(resp.InstanceTypes) == 0 {
		return nil, errors.New("not found")
	}

	if len(resp.InstanceTypes) != 1 {
		return nil, fmt.Errorf("total count from DescribeInstanceTypes resp is expected 1, but got %d", len(resp.InstanceTypes))
	}
	return resp.InstanceTypes[0], nil
}

func (volc *VolcEngin) Region() string {
	return volc.region
}

func (volc *VolcEngin) Zone() string {
	return volc.zone
}

func (volc *VolcEngin) Topology() map[string]string {
	return map[string]string{
		consts.TopologyRegionKey: volc.Region(),
		consts.TopologyZoneKey:   volc.Zone(),
	}
}
