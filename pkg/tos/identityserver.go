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
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// IdentityServer driver
type IdentityServer struct {
	d *Driver
}

// NewIdentityServer new IdentityServer
func NewIdentityServer(driver *Driver) *IdentityServer {
	return &IdentityServer{driver}
}

// GetPluginInfo return info of the plugin
func (ids *IdentityServer) GetPluginInfo(_ context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
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

// Probe check whether the plugin is running or not.
func (ids *IdentityServer) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	return &csi.ProbeResponse{Ready: &wrapperspb.BoolValue{Value: true}}, nil
}

// GetPluginCapabilities return the capabilities of the plugin
func (ids *IdentityServer) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_UNKNOWN,
					},
				},
			},
		},
	}, nil
}
