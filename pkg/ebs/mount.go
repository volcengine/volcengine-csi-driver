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

package ebs

import (
	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

// Mounter is the interface implemented by NodeMounter.
type Mounter interface {
	mount.Interface

	FormatAndMount(source string, target string, fstype string, options []string) error
	GetDeviceNameFromMount(mountPath string) (string, int, error)
}

// NodeMounter implements Mounter.
// A superstruct of SafeFormatAndMount.
type NodeMounter struct {
	*mount.SafeFormatAndMount
}

func newNodeMounter() Mounter {
	// mounter.NewSafeMounter returns a SafeFormatAndMount
	safeMounter := &mount.SafeFormatAndMount{
		Interface: mount.New(""),
		Exec:      exec.New(),
	}
	return &NodeMounter{safeMounter}
}

// GetDeviceNameFromMount returns the volume ID for a mount path.
func (m NodeMounter) GetDeviceNameFromMount(mountPath string) (string, int, error) {
	return mount.GetDeviceNameFromMount(m, mountPath)
}
