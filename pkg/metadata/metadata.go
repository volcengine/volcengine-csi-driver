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

package metadata

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/volcengine/volcengine-csi-driver/pkg/util"
	"k8s.io/klog/v2"
)

type MetadataService interface {
	NodeId() string
	Region() string
	Zone() string
	Credential() (accessKeyId, secretAccessKey, securityToken string)
	Active() bool
}

func NewECSMetadataService(metadataURL string) MetadataService {
	return &ecsMetadataService{
		metadataURL: metadataURL,
	}
}

type ecsMetadataService struct {
	metadataURL string
}

func (e *ecsMetadataService) NodeId() string {
	nodeId, err := util.HttpGet(fmt.Sprintf("%s/%s", e.metadataURL, "instance_id"))
	if err != nil {
		klog.Errorf("Get NodeId from ecs metadata server failed: %v", err)
		return ""
	}
	return nodeId
}

func (e *ecsMetadataService) Region() string {
	regionId, err := util.HttpGet(fmt.Sprintf("%s/%s", e.metadataURL, "region_id"))
	if err != nil {
		klog.Errorf("Get regionId from ecs metadata server failed: %v", err)
		return ""
	}
	return regionId
}

func (e *ecsMetadataService) Zone() string {
	zone, err := util.HttpGet(fmt.Sprintf("%s/%s", e.metadataURL, "availability_zone"))
	if err != nil {
		klog.Errorf("Get zone from ecs metadata server failed: %v", err)
		return ""
	}
	return zone
}

func (e *ecsMetadataService) Credential() (accessKeyId, secretAccessKey, securityToken string) {
	// get from sts
	return "", "", ""
}

func (e *ecsMetadataService) Active() bool {
	_url, err := url.Parse(e.metadataURL)
	if err != nil {
		klog.Errorf("Parse metadataURL failed: %v", err)
		return false
	}
	addr := fmt.Sprintf("%s:%s", _url.Host, "http")
	_, err = net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		klog.Errorf("Dial metadata host failed: %v", err)
		return false
	}
	return true
}
