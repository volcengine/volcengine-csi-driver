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
)

func TestGetNfsPathDetail(t *testing.T) {
	type args struct {
		nfsPath string
	}
	tests := []struct {
		name        string
		args        args
		wantFsId    string
		wantSubPath string
	}{
		{"testSingleLayer",
			args{"/fsid/layer"},
			"fsid",
			"/layer",
		},
		{"testMultiLayer",
			args{"/fsid/layer1/layer2"},
			"fsid",
			"/layer1/layer2",
		},
		{"testEmptyLayer",
			args{"/fsid"},
			"fsid",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFsId, gotSubPath := getNfsPathDetail(tt.args.nfsPath)
			if gotFsId != tt.wantFsId {
				t.Errorf("getNfsPathDetail() gotFsId = %v, want %v", gotFsId, tt.wantFsId)
			}
			if gotSubPath != tt.wantSubPath {
				t.Errorf("getNfsPathDetail() gotSubPath = %v, want %v", gotSubPath, tt.wantSubPath)
			}
		})
	}
}

func TestParseMountFlags(t *testing.T) {

	mntOptions1 := []string{"mnt=/test", "vers=3.0"}

	ver, result := parseMountOptions(mntOptions1)

	assert.Equal(t, "3", ver)
	assert.Equal(t, "mnt=/test", result)

	mntOptions2 := []string{"mnt=/test", "vers=3"}

	ver, result = parseMountOptions(mntOptions2)

	assert.Equal(t, "3", ver)
	assert.Equal(t, "mnt=/test", result)

	mntOptions3 := []string{"mnt=/test", "vers=4.0"}

	ver, result = parseMountOptions(mntOptions3)

	assert.Equal(t, "4.0", ver)
	assert.Equal(t, "mnt=/test", result)

	mntOptions4 := []string{"mnt=/test", "vers=4.1"}

	ver, result = parseMountOptions(mntOptions4)

	assert.Equal(t, "4.1", ver)
	assert.Equal(t, "mnt=/test", result)
}
