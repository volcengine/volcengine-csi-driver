/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

​http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebs

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type IdentitySvc struct {
	d *Driver
}

func NewIdentitySvc(driver *Driver) *IdentitySvc {
	return &IdentitySvc{driver}
}

func (ids *IdentitySvc) GetPluginInfo(_ context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {

	if ids.d.Name == "" {
		return nil, status.Error(codes.Unavailable, "Plugin name is not configured")
	}
	if ids.d.Version == "" {
		return nil, status.Error(codes.Unavailable, "Plugin version is not configured")
	}

	resp := &csi.GetPluginInfoResponse{
		Name:          ids.d.Name,
		VendorVersion: ids.d.Version,
	}

	return resp, nil
}

func (ids *IdentitySvc) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	resp := &csi.ProbeResponse{
		Ready: wrapperspb.Bool(true),
	}
	return resp, nil
}

func (ids *IdentitySvc) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {

	resp := &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
					},
				},
			},
		},
	}
	return resp, nil
}
