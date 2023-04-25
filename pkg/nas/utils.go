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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"

	"k8s.io/klog/v2"
)

func getNfsPathDetail(nfsPath string) (fsId, subPath string) {
	nfsPathList := strings.Split(nfsPath, "/")
	if len(nfsPathList) == 2 {
		fsId = nfsPathList[1]
		subPath = ""
	} else if len(nfsPathList) >= 3 {
		fsId = nfsPathList[1]
		subPath = "/" + strings.Join(nfsPathList[2:], "/")
	}
	return
}

// isNfsPathMounted check whether the given nfs path was mounted
func isNfsPathMounted(mountPoint, path string) bool {
	findmntCmd := "grep"
	findmntArgs := []string{mountPoint, "/proc/mounts"}
	out, err := exec.Command(findmntCmd, findmntArgs...).CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return false
	}
	if strings.Contains(outStr, path) {
		return true
	}
	return false
}

// doNfsMount execute the mount command for nas dir
func doNfsMount(nfsServer, fsId, subPath, nfsVers, mountOptions, mountPoint, volumeID string) error {
	if !util.IsFileExisting(mountPoint) {
		if err := os.MkdirAll(mountPoint, 0777); err != nil {
			klog.Errorf("doNfsMount: Create mountPoint directory error: %s", err.Error())
			return err
		}
	}
	nfsPath := filepath.Join("/"+fsId, subPath)

	if isNfsPathMounted(mountPoint, nfsPath) {
		klog.Infof("doNfsMount: nfs server already mounted: %s, %s", nfsServer, nfsPath)
		return nil
	}

	varOption := fmt.Sprintf("vers=%s", nfsVers)
	options := []string{varOption}
	if mountOptions != "" {
		options = append(options, strings.Split(mountOptions, ",")...)
	}
	source := fmt.Sprintf("%s:%s", nfsServer, nfsPath)
	err := util.DefaultMounter.Mount(source, mountPoint, "nfs", options)
	if err != nil && nfsPath != "/" {
		if err := createNasSubDir(nfsServer, fsId, subPath, nfsVers, mountOptions, volumeID); err != nil {
			klog.Errorf("doNfsMount: Create SubPath error: %s", err.Error())
			return err
		}
		if err := util.DefaultMounter.Mount(source, mountPoint, "nfs", options); err != nil {
			klog.Errorf("doNfsMount, Mount Nfs sub directory fail: %s", err.Error())
			return err
		}
	} else if err != nil {
		return err
	}
	klog.Infof("doNfsMount: mount nfs successful with source: %s", source)
	return nil
}

func createNasSubDir(nfsServer, fsId, subPath, nfsVers, nfsOptions string, volumeID string) error {
	// step 1: create mount path
	nasTmpPath := filepath.Join(TempMntPath, volumeID)
	if err := util.CreateDest(nasTmpPath); err != nil {
		klog.Infof("Create Nas temp Directory err: " + err.Error())
		return err
	}
	if notMnt, _ := util.DefaultMounter.IsLikelyNotMountPoint(nasTmpPath); !notMnt {
		err := util.DefaultMounter.Unmount(nasTmpPath)
		if err != nil {
			klog.Errorf("Nas, unmount directory nasTmpPath fail, nasTmpPath:%s, err:%s", nasTmpPath, err.Error())
			return err
		}
	}

	// step 2: do mount, and create subpath
	varOption := fmt.Sprintf("vers=%s", nfsVers)
	options := []string{varOption}
	if nfsOptions != "" {
		options = append(options, strings.Split(nfsOptions, ",")...)
	}
	rootPath := "/" + fsId
	source := fmt.Sprintf("%s:%s", nfsServer, rootPath)
	err := util.DefaultMounter.Mount(source, nasTmpPath, "nfs", options)
	if err != nil {
		klog.Errorf("Nas, mount nas rootPath fail, source:%s, err:%s", source, err.Error())
		return err
	}

	destPath := path.Join(nasTmpPath, subPath)
	if err := util.CreateDest(destPath); err != nil {
		klog.Infof("Nas, Create Sub Directory fail, subPath:%s, err: " + err.Error())
		return err
	}

	// step 3: umount after create
	err = util.DefaultMounter.Unmount(nasTmpPath)
	if err != nil {
		klog.Errorf("Nas, unmount directory nasTmpPath fail, nasTmpPath:%s, err:%s", nasTmpPath, err.Error())
		return err
	}
	klog.Infof("Create Sub Directory successful, fsId: %s, subPath: %s", fsId, subPath)
	return nil
}

// parseMountOptions parse mountOptions
func parseMountOptions(mntOptions []string) (string, string) {
	if len(mntOptions) > 0 {
		mntOptionsStr := strings.Join(mntOptions, ",")
		// mntOptions should re-split, as some like ["a,b,c", "d"]
		mntOptionsList := strings.Split(mntOptionsStr, ",")
		tmpOptionsList := []string{}
		nfsVers := ""
		validVersion := map[string]bool{"3": true, "4": true, "4.0": true, "4.1": true, "4.2": true}
		for _, tmpOptions := range mntOptionsList {
			if strings.HasPrefix(tmpOptions, "vers=") {
				nfsVers = tmpOptions[5:]
			} else {
				tmpOptionsList = append(tmpOptionsList, tmpOptions)
			}
		}
		if !validVersion[nfsVers] {
			nfsVers = "3"
		}
		return nfsVers, strings.Join(tmpOptionsList, ",")

	}
	return "", ""
}

// nolint
func getNasVolumeArgs(volParameters map[string]string) (*nasVolumeArgs, error) {
	var ok bool
	nasVolArgs := &nasVolumeArgs{}
	if nasVolArgs.VolumeAs, ok = volParameters[VolumeAs]; !ok {
		nasVolArgs.VolumeAs = "subpath"
	} else if nasVolArgs.VolumeAs != "subpath" {
		return nil, fmt.Errorf("required parameter [parameter.volumeAs] must be [subpath]")
	}

	if nasVolArgs.ArchiveOnDelete, ok = volParameters[ArchiveOnDelete]; !ok {
		nasVolArgs.ArchiveOnDelete = "true"
	}
	if nasVolArgs.VolumeAs == "subpath" {
		nasVolArgs.Server = volParameters[SERVER]
		if len(nasVolArgs.Server) == 0 {
			return nil, fmt.Errorf("server can not be empty")
		}
		nasVolArgs.FsID = volParameters[FSID]
		if strings.Contains(nasVolArgs.FsID, "/") {
			return nil, fmt.Errorf("fsId can not contain /")
		}
		nasVolArgs.SubPath = volParameters[SUBPATH]
		if nasVolArgs.SubPath != "" && !strings.HasPrefix(nasVolArgs.SubPath, "/") {
			nasVolArgs.SubPath = "/" + nasVolArgs.SubPath
		}
		// remove / if subPath end with /;
		if strings.HasSuffix(nasVolArgs.SubPath, "/") {
			nasVolArgs.SubPath = nasVolArgs.SubPath[0 : len(nasVolArgs.SubPath)-1]
		}
	} else if nasVolArgs.VolumeAs == "filesystem" {
		// TODO: enrich other option when volumeAs filesystem
		klog.Error("not support filesystem type")
	}

	return nasVolArgs, nil
}
