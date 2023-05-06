/*
Copyright 2017 The Kubernetes Authors.
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

package csicommon

import (
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"k8s.io/klog/v2"
)

type CSIDriver struct {
	Name    string
	NodeID  string
	Version string
	Cap     []*csi.VolumeCapability_AccessMode
	CSCap   []*csi.ControllerServiceCapability
	NSCap   []*csi.NodeServiceCapability
}

// AddVolumeCapabilityAccessModes add VolumeCapability_AccessMode_Mode
func (d *CSIDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		klog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, &csi.VolumeCapability_AccessMode{Mode: c})
	}
	d.Cap = vca
	return vca
}

// AddControllerServiceCapabilities add ControllerServiceCapability_RPC_Type
func (d *CSIDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability

	for _, c := range cl {
		klog.Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}

	d.CSCap = csc
}

// AddNodeServiceCapabilities add NodeServiceCapability_RPC_Type
func (d *CSIDriver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) {
	var nsc []*csi.NodeServiceCapability
	for _, n := range nl {
		klog.Infof("Enabling node service capability: %v", n.String())
		nsc = append(nsc, NewNodeServiceCapability(n))
	}
	d.NSCap = nsc
}
