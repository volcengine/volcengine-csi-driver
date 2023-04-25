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
	"context"
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"

	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestGetPluginInfo(t *testing.T) {
	req := csi.GetPluginInfoRequest{}
	emptyNameDriver := NewFakeDriver("name")
	emptyVersionDriver := NewFakeDriver("version")
	tests := []struct {
		desc        string
		driver      *Driver
		expectedErr error
	}{
		{
			desc:        "Successful Request",
			driver:      NewFakeDriver(""),
			expectedErr: nil,
		},
		{
			desc:        "Driver name missing",
			driver:      emptyNameDriver,
			expectedErr: status.Error(codes.Unavailable, "Plugin name is not configured"),
		},
		{
			desc:        "Driver version missing",
			driver:      emptyVersionDriver,
			expectedErr: status.Error(codes.Unavailable, "Plugin version is not configured"),
		},
	}

	for _, test := range tests {
		fakeIdentityServer := IdentityServer{test.driver}
		_, err := fakeIdentityServer.GetPluginInfo(context.Background(), &req)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v\nExpected: %v", err, test.expectedErr)
		}
	}
}

func TestProbe(t *testing.T) {
	d := NewFakeDriver("")
	req := csi.ProbeRequest{}
	fakeIdentityServer := IdentityServer{d}
	resp, err := fakeIdentityServer.Probe(context.Background(), &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.XXX_sizecache, int32(0))
	assert.Equal(t, resp.Ready.Value, true)
}

func TestIdentityServer_GetPluginCapabilities(t *testing.T) {
	expectedCap := []*csi.PluginCapability{
		{
			Type: &csi.PluginCapability_Service_{
				Service: &csi.PluginCapability_Service{
					Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
				},
			},
		},
	}

	d := NewFakeDriver("")
	fakeIdentityServer := IdentityServer{d}
	req := csi.GetPluginCapabilitiesRequest{}
	resp, err := fakeIdentityServer.GetPluginCapabilities(context.Background(), &req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, resp.XXX_sizecache, int32(0))
	assert.Equal(t, resp.Capabilities, expectedCap)
}
