/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

​http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ebs

import (
	"fmt"

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/types"
	"github.com/volcengine/volcengine-csi-driver/pkg/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
)

func getDiskCapacity(devicePath string) (float64, error) {
	_, capacity, _, _, _, _, err := util.FsInfo(devicePath)
	if err != nil {
		klog.Errorf("getDiskCapacity:: get device error: %+v", err)
		return 0, fmt.Errorf("getDiskCapacity:: get device error: %+v", err)
	}
	capacity, ok := (*(resource.NewQuantity(capacity, resource.BinarySI))).AsInt64()
	if !ok {
		klog.Errorf("getDiskCapacity:: failed to fetch capacity bytes for target: %s", devicePath)
		return 0, status.Error(codes.Unknown, "failed to fetch capacity bytes")
	}
	return float64(capacity) / float64(types.GB), nil
}

func int32ToPtr(i int32) *int32 {
	return &i
}
