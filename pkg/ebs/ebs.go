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
	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

const (
	DefaultDriverName = "ebs.csi.volcengine.com"
)

type Driver struct {
	*csicommon.CSIDriver
	maxVolumesPerNode     int64
	reserveVolumesPerNode int64
}

// NewDriver create the identity/node/controller server and disk driver
func NewDriver(name, version, nodeId string, maxVolumesPerNode, reserveVolumesPerNode int64) *Driver {
	csiDriver := &csicommon.CSIDriver{}
	csiDriver.Name = DefaultDriverName
	if name != "" {
		csiDriver.Name = name
	}

	csiDriver.Version = version
	csiDriver.NodeID = nodeId
	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		csi.VolumeCapability_AccessMode_SINGLE_NODE_READER_ONLY,
	})
	csiDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
	})
	csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	})

	return &Driver{
		CSIDriver:             csiDriver,
		maxVolumesPerNode:     maxVolumesPerNode,
		reserveVolumesPerNode: reserveVolumesPerNode,
	}
}

func (d *Driver) Run(endpoint string, cloud Cloud) {
	klog.Infof("Starting csi-plugin Driver: %v version: %v", d.Name, d.Version)
	s := csicommon.NewNonBlockingGRPCServer()

	s.Start(
		endpoint,
		NewIdentitySvc(d),
		NewControllerSvc(d, cloud),
		NewNodeSvc(
			d,
			newNodeMounter(),
			cloud),
		false,
	)
	s.Wait()
}
