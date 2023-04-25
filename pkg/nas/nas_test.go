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
	"testing"

	"github.com/stretchr/testify/assert"
	csicommon "github.com/volcengine/volcengine-csi-driver/pkg/csi-common"
)

const (
	fakeNodeID = "fakeNodeID"
)

func NewFakeDriver(field string) *Driver {
	csiDriver := &csicommon.CSIDriver{}
	switch field {
	case "version":
		csiDriver.Name = DefaultDriverName
		csiDriver.Version = ""
		csiDriver.NodeID = fakeNodeID
	case "name":
		csiDriver.Name = ""
		csiDriver.Version = "1.0.0"
		csiDriver.NodeID = fakeNodeID
	default:
		csiDriver.Name = DefaultDriverName
		csiDriver.Version = "1.0.0"
		csiDriver.NodeID = fakeNodeID
	}

	return &Driver{csiDriver, nil}
}

func TestNewFakeDriver(t *testing.T) {
	d := NewFakeDriver("version")
	assert.Empty(t, d.Version)

	d = NewFakeDriver("name")
	assert.Empty(t, d.Name)
}
