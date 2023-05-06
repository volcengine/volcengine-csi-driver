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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func validateNodePublishVolumeRequest(req *csi.NodePublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return errors.New("volume ID missing in request")
	}
	if req.GetTargetPath() == "" {
		return errors.New("target path missing in request")
	}
	if req.GetVolumeCapability() == nil {
		return errors.New("volume capability missing in request")
	}
	return nil
}

func validateNodeUnpublishVolumeRequest(req *csi.NodeUnpublishVolumeRequest) error {
	if req.GetVolumeId() == "" {
		return errors.New("volume ID missing in request")
	}
	if req.GetTargetPath() == "" {
		return errors.New("target path missing in request")
	}
	return nil
}

func parseTosfsOptions(attributes map[string]string) (*tosfsOptions, error) {
	options := &tosfsOptions{}
	for k, v := range attributes {
		switch strings.ToLower(k) {
		case paramURL:
			if !strings.Contains(v, "-s3") {
				idx := strings.Index(v, "tos")
				v = v[:idx+3] + "-s3" + v[idx+3:]
				klog.Warningf("tos url not support s3fs, change to %s", v)
			}
			options.URL = v
		case paramBucket:
			options.Bucket = v
		case paramPath:
			options.Path = v
		case paramAdditionalArgs:
			options.AdditionalArgs = v
		case paramDbgLevel:
			options.DbgLevel = v
		}
	}

	if options.DbgLevel == "" {
		options.DbgLevel = defaultDBGLevel
	}

	return options, validateTosfsOptions(options)
}

func validateTosfsOptions(options *tosfsOptions) error {
	if options.URL == "" {
		return errors.New("TOS service URL can't be empty")
	}
	if options.Bucket == "" {
		return errors.New("TOS bucket can't be empty")
	}
	return nil
}

func createCredentialFile(volID, bucket string, secrets map[string]string) (string, error) {
	credential, err := getSecretCredential(secrets)
	if err != nil {
		klog.Errorf("getSecretCredential info from NodeStageSecrets failed: %v", err)
		return "", status.Errorf(codes.InvalidArgument, "get credential failed: %v", err)
	}

	// compute sha256 and add on password file name
	credSHA := sha256.New()
	credSHA.Write([]byte(credential))
	shaString := hex.EncodeToString(credSHA.Sum(nil))
	passwdFilename := fmt.Sprintf("%s%s_%s", tosPasswordFileDirectory, bucket, shaString)

	klog.Infof("tosfs password file name is %s", passwdFilename)

	if _, err := os.Stat(passwdFilename); err != nil {
		if os.IsNotExist(err) {
			if err := ioutil.WriteFile(passwdFilename, []byte(credential), 0600); err != nil {
				klog.Errorf("create password file for volume %s failed: %v", volID, err)
				return "", status.Errorf(codes.Internal, "create tmp password file failed: %v", err)
			}
		} else {
			klog.Errorf("stat password file  %s failed: %v", passwdFilename, err)
			return "", status.Errorf(codes.Internal, "stat password file failed: %v", err)
		}
	} else {
		klog.Infof("password file %s is exist, and sha256 is same", passwdFilename)
	}

	return passwdFilename, nil
}

func getSecretCredential(secrets map[string]string) (string, error) {
	sid := strings.TrimSpace(secrets[credentialID])
	skey := strings.TrimSpace(secrets[credentialKey])
	if sid == "" || skey == "" {
		return "", fmt.Errorf("secret must contains %v and %v", credentialID, credentialKey)
	}
	return strings.Join([]string{sid, skey}, ":"), nil
}

func mount(options *tosfsOptions, mountPoint string, credentialFilePath string) error {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	bucketOrWithSubDir := options.Bucket
	if options.Path != "" {
		bucketOrWithSubDir = fmt.Sprintf("%s:%s", options.Bucket, options.Path)
	}
	args := []string{
		bucketOrWithSubDir,
		mountPoint,
		"-ourl=" + options.URL,
		"-odbglevel=" + options.DbgLevel,
		"-opasswd_file=" + credentialFilePath,
	}
	if options.AdditionalArgs != "" {
		args = append(args, options.AdditionalArgs)
	}
	if options.NotsupCompatDir {
		args = append(args, "-onotsup_compat_dir")
	}

	body := make(map[string]string)
	body["command"] = fmt.Sprintf("s3fs %s", strings.Join(args, " "))
	bodyJson, _ := json.Marshal(body)
	response, err := httpClient.Post("http://unix/launcher", "application/json", strings.NewReader(string(bodyJson)))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the response of launcher(action: s3fs) is: %v", string(respBody))
	}

	klog.Info("send s3fs command to launcher successfully")

	return nil
}

func checkTosMounted(mountPoint string) error {
	// Wait until TOS is successfully mounted.
	// Totally 4 seconds
	retryTimes := 20
	interval := time.Millisecond * 200
	notMnt := true
	var err error
	for i := 0; i < retryTimes; i++ {
		if notMnt, err = util.DefaultMounter.IsLikelyNotMountPoint(mountPoint); err == nil {
			if !notMnt {
				break
			} else {
				time.Sleep(interval)
			}
		} else {
			return err
		}
	}
	if notMnt {
		return errors.New("check tos mounted timeout")
	}
	return nil
}
