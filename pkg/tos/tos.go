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
	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"k8s.io/klog/v2"
)

const (
	DefaultDriverName = "tos.csi.volcengine.com"

	// Address of the tos server
	paramURL = "url"
	// Base directory of the tos to create volumes under.
	paramPath = "path"
	// Bucket of tos
	paramBucket = "bucket"
	// Additional Args
	paramAdditionalArgs = "additional_args"
	// Debug level
	paramDbgLevel = "dbglevel"

	defaultDBGLevel          = "err"
	tosPasswordFileDirectory = "/tmp/"
	socketPath               = "/tmp/tosfs.sock"
	credentialID             = "akId"
	credentialKey            = "akSecret"

	// tempMntPath used for create tos sub directory
	tempMntPath = "/tmp/tos_mnt/"
)

type Driver struct {
	*csicommon.CSIDriver
}

// NewDriver create the identity/node/controller server and disk driver
func NewDriver(name, version, nodeId string) *Driver {
	csiDriver := &csicommon.CSIDriver{}
	csiDriver.Name = DefaultDriverName
	if name != "" {
		csiDriver.Name = name
	}

	csiDriver.Version = version
	csiDriver.NodeID = nodeId
	csiDriver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{
		csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
	})
	csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_UNKNOWN,
	})

	return &Driver{
		CSIDriver: csiDriver,
	}
}

func (d *Driver) Run(endpoint string) {
	klog.Infof("Starting csi-plugin Driver: %v version: %v", d.Name, d.Version)

	s := csicommon.NewNonBlockingGRPCServer()

	s.Start(
		endpoint,
		NewIdentityServer(d),
		nil,
		NewNodeServer(d),
		false,
	)
	s.Wait()
}
