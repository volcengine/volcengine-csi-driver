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

package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

// GetDeviceByVolumeID First try to find the device by serial
// If cannot find the device using the serial number, get device by volumeID, link file should be like:
// /dev/disk/by-id/virtio-163696622650261511 -> ../../vdc
func GetDeviceByVolumeID(volumeID string) (device string, err error) {
	device = getDeviceSerial(strings.TrimPrefix(volumeID, "vol-"))
	if device != "" {
		klog.Infof("Use the serial to find device, got %q, volumeID: %s", device, volumeID)
		return device, nil
	}

	byIDPath := "/dev/disk/by-id/"
	volumeLinkName := strings.Replace(volumeID, "vol-", "virtio-", -1)
	volumeLinPath := filepath.Join(byIDPath, volumeLinkName)

	stat, err := os.Lstat(volumeLinPath)
	if err != nil {
		if os.IsNotExist(err) {
			// in some os, link file is not begin with virtio-,
			// but diskPart will always be part of link file.
			isSearched := false
			files, _ := ioutil.ReadDir(byIDPath)
			diskPart := strings.Replace(volumeID, "vol-", "", -1)
			for _, f := range files {
				if strings.Contains(f.Name(), diskPart) {
					volumeLinPath = filepath.Join(byIDPath, f.Name())
					stat, _ = os.Lstat(volumeLinPath)
					isSearched = true
					break
				}
			}
			if !isSearched {
				klog.Warningf("volumeID link path %q not found", volumeLinPath)
				return "", fmt.Errorf("volumeID link path %q not found", volumeLinPath)
			}
		} else {
			return "", fmt.Errorf("error getting stat of %q: %v", volumeLinPath, err)
		}
	}

	if stat.Mode()&os.ModeSymlink != os.ModeSymlink {
		klog.Warningf("volumeID link file %q found, but was not a symlink", volumeLinPath)
		return "", fmt.Errorf("volumeID link file %q found, but was not a symlink", volumeLinPath)
	}
	// Find the target, resolving to an absolute path
	// For example, /dev/disk/by-id/virtio-wz9cu3ctp6aj1iagco4h -> ../../vdc
	resolved, err := filepath.EvalSymlinks(volumeLinPath)
	if err != nil {
		return "", fmt.Errorf("error reading target of symlink %q: %v", volumeLinPath, err)
	}
	if !strings.HasPrefix(resolved, "/dev") {
		return "", fmt.Errorf("resolved symlink for %q was unexpected: %q", volumeLinPath, resolved)
	}
	klog.Infof("Device Link Info: %s link to %s", volumeLinPath, resolved)
	return resolved, nil
}

func getDeviceSerial(serial string) (device string) {
	serialFiles, err := filepath.Glob("/sys/block/*/serial")
	if err != nil {
		klog.Errorf("List device serial failed: %v", err)
		return ""
	}

	for _, serialFile := range serialFiles {
		body, err := ioutil.ReadFile(serialFile)
		if err != nil {
			klog.Errorf("Read serial(%s): %v", serialFile, err)
			continue
		}
		if strings.TrimSpace(string(body)) == serial {
			return filepath.Join("/dev", filepath.Base(filepath.Dir(serialFile)))
		}
	}
	return ""
}

func run(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Failed to run cmd: %s, with out: %s, error: %s ", cmd, out, err.Error())
	}
	return string(out), nil
}

// IsFileExisting check file exist in volume driver or not
func IsFileExisting(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// CreateDest create de destination dir
func CreateDest(dest string) error {
	fi, err := os.Lstat(dest)

	if os.IsNotExist(err) {
		return os.MkdirAll(dest, 0777)
	} else if err != nil {
		return err
	}

	if fi != nil && !fi.IsDir() {
		return fmt.Errorf("%v already exist but it's not a directory", dest)
	}
	return nil
}

// FsInfo linux returns (available bytes, byte capacity, byte usage, total inodes, inodes free, inode usage, error)
// for the filesystem that path resides upon.
func FsInfo(path string) (int64, int64, int64, int64, int64, int64, error) {
	statfs := &unix.Statfs_t{}
	err := unix.Statfs(path, statfs)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	// Available is blocks available * fragment size
	available := int64(statfs.Bavail) * int64(statfs.Bsize)

	// Capacity is total block count * fragment size
	capacity := int64(statfs.Blocks) * int64(statfs.Bsize)

	// Usage is block being used * fragment size (aka block size).
	usage := (int64(statfs.Blocks) - int64(statfs.Bfree)) * int64(statfs.Bsize)

	inodes := int64(statfs.Files)
	inodesFree := int64(statfs.Ffree)
	inodesUsed := inodes - inodesFree

	return available, capacity, usage, inodes, inodesFree, inodesUsed, nil
}
