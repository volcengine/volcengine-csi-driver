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
	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"
	"github.com/volcengine/volcengine-csi-driver/pkg/openapi"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"k8s.io/klog/v2"
)

const (
	DefaultDriverName = "nas.csi.volcengine.com"

	// Address of the NFS server
	paramServer = "server"

	// Base directory of the NFS server to create volumes under.
	paramPath = "path"

	// TempMntPath used for create nas sub directory
	TempMntPath = "/mnt/ecs_mnt/k8s_nas/temp"

	// VolumeAs subpath or filesystem
	VolumeAs = "volumeAs"

	// MntRootPath used for create/delete volume sub directory
	MntRootPath = "/csi-persistentvolumes"

	// Nas Server
	SERVER = "server"

	// Nas FsID
	FSID = "fsId"

	// Nas SubPath
	SUBPATH = "subPath"

	// ArchiveOnDelete is a parameter in StorageClass which determines whether to archive
	ArchiveOnDelete = "archiveOnDelete"

	VCIInstanceRoleForVKE = "VCIInstanceRoleForVKE"
)

type Driver struct {
	*csicommon.CSIDriver
	*openapi.Config
}

// NewDriver create the identity/node/controller server and disk driver
func NewDriver(name, version, nodeId string, config *openapi.Config) *Driver {
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
		csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
		csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})
	csiDriver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	})
	csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		csi.NodeServiceCapability_RPC_UNKNOWN,
	})

	return &Driver{
		CSIDriver: csiDriver,
		Config:    config,
	}
}

func (d *Driver) Run(endpoint string) {
	klog.Infof("Starting csi-plugin Driver: %v version: %v", d.Name, d.Version)
	s := csicommon.NewNonBlockingGRPCServer()

	s.Start(
		endpoint,
		NewIdentityServer(d),
		NewControllerServer(d),
		NewNodeServer(d),
		false,
	)
	s.Wait()
}
