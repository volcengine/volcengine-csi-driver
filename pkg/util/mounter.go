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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
	"k8s.io/mount-utils"
)

// Mounter is responsible for formatting and mounting volumes
type Mounter interface {
	mount.Interface

	// ForceUnmount the given target
	ForceUnmount(target string) error
}

var DefaultMounter = NewMounter()

type mounter struct {
	mount.Interface
}

// NewMounter returns a new mounter instance
func NewMounter() Mounter {
	return &mounter{Interface: mount.New("")}
}

func (m *mounter) ForceUnmount(target string) error {
	umountCmd := "umount"
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	umountArgs := []string{"-f", target}

	klog.Infof("ForceUnmount %s, the command is %s %v", target, umountCmd, umountArgs)

	out, err := exec.Command(umountCmd, umountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: '%s -f %s' output: %q",
			err, umountCmd, target, string(out))
	}

	return nil
}

func CheckDeviceAvailable(devicePath string) error {
	if devicePath == "" {
		return status.Error(codes.Internal, "devicePath is empty, cannot used for Volume")
	}

	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return err
	}

	// check the device is used for system
	if devicePath == "/dev/vda" || devicePath == "/dev/vda1" {
		return fmt.Errorf("devicePath(%s) is system device, cannot used for Volume", devicePath)
	}

	checkCmd := fmt.Sprintf("mount | grep \"%s on /var/lib/kubelet type\" | wc -l", devicePath)
	if out, err := run(checkCmd); err != nil {
		return fmt.Errorf("devicePath(%s) is used to kubelet", devicePath)
	} else if strings.TrimSpace(out) != "0" {
		return fmt.Errorf("devicePath(%s) is used as DataDisk for kubelet, cannot used fo Volume", devicePath)
	}
	return nil
}
