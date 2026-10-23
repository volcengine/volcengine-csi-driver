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
	"fmt"

	"k8s.io/mount-utils"
	"k8s.io/utils/exec"
)

// Mounter is the interface implemented by NodeMounter.
type Mounter interface {
	mount.Interface

	FormatAndMountWithOptions(source string, target string, fstype string, options, formatOptions []string) error
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

// FormatAndMountWithOptions applies formatOptions only to an unformatted device.
// Existing filesystems and partition tables retain the upstream mount behavior.
func (m NodeMounter) FormatAndMountWithOptions(source, target, fstype string, options, formatOptions []string) error {
	if len(formatOptions) == 0 {
		return m.FormatAndMount(source, target, fstype, options)
	}
	if fstype == "" {
		fstype = "ext4"
	}
	existingFormat, err := m.GetDiskFormat(source)
	if err != nil {
		return mount.NewMountError(mount.GetDiskFormatFailed, "failed to probe device %s: %v", source, err)
	}
	if existingFormat != "" {
		return m.FormatAndMount(source, target, fstype, options)
	}
	for _, option := range options {
		if option == "ro" {
			return mount.NewMountError(mount.UnformattedReadOnly, "cannot format read-only device %s", source)
		}
	}
	var args []string
	switch fstype {
	case "ext3", "ext4":
		args = []string{"-F", "-m0"}
	case "xfs":
		args = []string{"-f"}
	}
	args = append(args, formatOptions...)
	args = append(args, source)
	output, err := m.Exec.Command("mkfs."+fstype, args...).CombinedOutput()
	if err != nil {
		return mount.NewMountError(mount.FormatFailed, "mkfs.%s failed for %s: %v, output: %s", fstype, source, err, output)
	}
	// Verify that mkfs created a filesystem whose detected type matches the
	// requested fstype before proceeding to mount.
	existingFormat, err = m.GetDiskFormat(source)
	if err != nil {
		return fmt.Errorf("failed to verify filesystem after mkfs on %s: %w", source, err)
	}
	if existingFormat != fstype {
		return mount.NewMountError(mount.FormatFailed, "mkfs on %s produced filesystem %q, expected %q", source, existingFormat, fstype)
	}
	return m.FormatAndMount(source, target, fstype, options)
}
