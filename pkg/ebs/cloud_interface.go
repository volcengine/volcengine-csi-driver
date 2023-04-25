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

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"
)

type Cloud interface {
	CreateVolume(ctx context.Context, name, volumeType, zoneId, snapshotId string, capacity int64) (id string, err error)
	ExtendVolume(ctx context.Context, id string, newSize int64) (err error)
	DeleteVolume(ctx context.Context, id string) error
	DevicePathByVolId(volId string) string
	AttachVolume(ctx context.Context, nodeId, volId string) error
	DetachVolume(ctx context.Context, nodeId, volId string) error
	NodeById(ctx context.Context, id string) (*types.InstanceForDescribeInstancesOutput, error)
	VolumeById(ctx context.Context, id string) (vol *types.Volume, err error)
	VolumeByName(ctx context.Context, name string) (vol *types.Volume, err error)
	CreateSnapshot(ctx context.Context, volumeID, snapshotName string) (snapshot *types.Snapshot, err error)
	DeleteSnapshot(ctx context.Context, snapshotID string) error
	GetSnapshotByName(ctx context.Context, name string) (snapshot *types.Snapshot, err error)
	GetSnapshotByID(ctx context.Context, snapshotID string) (snapshot *types.Snapshot, err error)
	Region() string
	Zone() string
	Topology() map[string]string
}
