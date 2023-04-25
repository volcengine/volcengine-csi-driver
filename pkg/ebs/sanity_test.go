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
	"math/rand"
	"os"
	"testing"
	"time"

	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/consts"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"

	"github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
	"k8s.io/mount-utils"
)

func TestSanity(t *testing.T) {
	// let openapi volumePath to be empty, load openapi from env
	d := NewDriver("mock.csi.volcengine.com", "v0.0.1.ut", "mock-node", 10)
	fc := newFakeCloud()

	go func() {
		s := csicommon.NewNonBlockingGRPCServer()
		s.Start(
			"unix:///tmp/mock-csi.sock",
			NewIdentitySvc(d),
			NewControllerSvc(d, fc),
			NewNodeSvc(d, newFakeMounter(), fc),
			false,
		)
		s.Wait()
	}()

	time.Sleep(1 * time.Second)

	config := sanity.NewTestConfig()
	config.Address = "unix:///tmp/mock-csi.sock"
	sanity.Test(t, config)
}

type fakeCloud struct {
	volumes   map[string]*types.Volume
	attached  map[string]string
	snapshots map[string]*types.Snapshot
}

func (c *fakeCloud) CreateSnapshot(ctx context.Context, volumeID, snapshotName string) (snapshot *types.Snapshot, err error) {
	snapshot = &types.Snapshot{
		SnapshotID:     volumeID,
		SourceVolumeID: volumeID,
		Size:           0,
		CreationTime:   time.Time{},
		ReadyToUse:     true,
	}
	c.snapshots[snapshotName] = snapshot
	return snapshot, nil
}

func (c *fakeCloud) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	for name, snapshot := range c.snapshots {
		if snapshot.SnapshotID == snapshotID {
			delete(c.snapshots, name)
		}
	}
	return nil
}

func (c *fakeCloud) GetSnapshotByName(ctx context.Context, name string) (snapshot *types.Snapshot, err error) {
	if _, ok := c.snapshots[name]; ok {
		return c.snapshots[name], nil
	}
	return nil, nil
}

func (c *fakeCloud) GetSnapshotByID(ctx context.Context, snapshotID string) (snapshot *types.Snapshot, err error) {
	for _, f := range c.snapshots {
		if f.SnapshotID == snapshotID {
			return f, nil
		}
	}
	return nil, nil
}

func newFakeCloud() Cloud {
	return &fakeCloud{
		volumes:   make(map[string]*types.Volume),
		attached:  make(map[string]string),
		snapshots: make(map[string]*types.Snapshot),
	}
}

func (c *fakeCloud) CreateVolume(ctx context.Context, name, volumeType, zoneId, snapshotId string, capacity int64) (id string, err error) {
	r1 := rand.New(rand.NewSource(time.Now().UnixNano()))
	v := &types.Volume{
		Id:         fmt.Sprintf("vol-%d", r1.Uint64()),
		Name:       name,
		Status:     types.StatusAvailable,
		Capacity:   capacity,
		NodeId:     "",
		ZoneId:     zoneId,
		VolumeType: volumeType,
	}
	c.volumes[name] = v
	return v.Id, nil
}

func (c *fakeCloud) DeleteVolume(ctx context.Context, id string) error {
	for volName, f := range c.volumes {
		if f.Id == id {
			delete(c.volumes, volName)
		}
	}
	return nil
}

func (c *fakeCloud) ExtendVolume(ctx context.Context, id string, newSize int64) (err error) {
	for volName, f := range c.volumes {
		if f.Id == id {
			c.volumes[volName].Capacity = newSize
		}
	}
	return nil
}

func (c *fakeCloud) AttachVolume(ctx context.Context, nodeId, volId string) error {
	if nid, ok := c.attached[volId]; ok {
		if nid == nodeId {
			return nil
		}
		return fmt.Errorf("volume attached by other node")
	}
	c.attached[volId] = nodeId
	for _, f := range c.volumes {
		if f.Id == volId {
			f.Status = types.StatusAttached
			f.NodeId = nodeId
		}
	}
	return nil
}

func (c *fakeCloud) DetachVolume(ctx context.Context, nodeId, volId string) error {
	for vid, nid := range c.attached {
		if vid == volId && nid == nodeId {
			delete(c.attached, volId)
		}
	}
	for _, f := range c.volumes {
		if f.Id == volId {
			f.Status = types.StatusAvailable
			f.NodeId = ""
		}
	}
	return nil
}

func (c *fakeCloud) NodeById(ctx context.Context, id string) (*types.InstanceForDescribeInstancesOutput, error) {
	volumeType := volumeTypeDefault
	if id == "mock-node" {
		return &types.InstanceForDescribeInstancesOutput{Id: &id, InstanceType: &types.InstanceTypeForDescribeInstancesOutput{VolumeTypes: []*string{&volumeType}}}, nil
	}
	return nil, errors.New("not found")
}

func (c *fakeCloud) VolumeById(ctx context.Context, id string) (vol *types.Volume, err error) {
	for _, f := range c.volumes {
		if f.Id == id {
			return f, nil
		}
	}
	return nil, nil
}

func (c *fakeCloud) VolumeByName(ctx context.Context, name string) (vol *types.Volume, err error) {
	if _, ok := c.volumes[name]; ok {
		return c.volumes[name], nil
	}
	return nil, nil
}

func (c *fakeCloud) DevicePathByVolId(volId string) string {
	path := "/tmp/testvolume"
	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return ""
	}
	for _, f := range c.volumes {
		if f.Id == volId {
			return path
		}
	}

	return ""
}

func (c *fakeCloud) Region() string {
	return "region"
}

func (c *fakeCloud) Zone() string {
	return "zone"
}

func (c *fakeCloud) Topology() map[string]string {
	return map[string]string{
		consts.TopologyRegionKey: c.Region(),
		consts.TopologyZoneKey:   c.Zone(),
	}
}

type fakeMounter struct {
	*mount.FakeMounter
}

func newFakeMounter() *fakeMounter {
	return &fakeMounter{
		mount.NewFakeMounter([]mount.MountPoint{}),
	}
}

func (f *fakeMounter) FormatAndMount(source string, target string, fstype string, options []string) error {
	// formats the given disk, do nothing

	if err := f.Mount(source, target, fstype, options); err != nil {
		return err
	}
	return nil
}

func (f *fakeMounter) GetDeviceNameFromMount(mountPath string) (string, int, error) {
	return "/tmp/testvolume", 1, nil
}
