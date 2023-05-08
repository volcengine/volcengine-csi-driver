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

package main

import (
	"context"
	"math"
	"os"

	"github.com/volcengine/volcengine-csi-driver/pkg/ebs"
	"github.com/volcengine/volcengine-csi-driver/pkg/metadata"
	"github.com/volcengine/volcengine-csi-driver/pkg/openapi"
	"github.com/volcengine/volcengine-csi-driver/pkg/sts"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

var (
	name                 string
	endpoint             string
	nodeId               string
	openApiCfgFile       string
	metadataURL          string
	version              string // Set by the build process
	showVersion          = false
	reserveVolumesFactor float64
)

const (
	defaultMaxVolumesPerNode int64 = 15
)

func loadOpenapiConfig() *openapi.Config {
	loaders := []openapi.CfgLoader{openapi.EnvLoader()}
	if info, err := os.Stat(openApiCfgFile); err == nil && info.Mode().IsRegular() {
		loaders = append(loaders, openapi.FileLoader(openApiCfgFile))
	}
	if metadataService := metadata.NewECSMetadataService(metadataURL); metadataService.Active() {
		loaders = append(loaders, openapi.ServerLoader(metadataService))
	}

	return openapi.ConfigVia(loaders...)
}

func run(cmd *cobra.Command, args []string) {
	if showVersion {
		klog.Infof("Driver name: %s, version: %s.", name, version)
		os.Exit(0)
	}

	metadataService := metadata.NewECSMetadataService(metadataURL)
	if nodeId == "" {
		klog.Info("node id is empty, trying to get node id from metadata server...")
		// get instance id from metadata server
		if nodeId = metadataService.NodeId(); nodeId == "" {
			klog.Error("get empty node id from metadata server")
			return
		}
	}

	config := loadOpenapiConfig()
	klog.V(5).Infof("Load openapi config %+v", config)

	serviceClients, err := sts.NewServiceClients(config)
	if err != nil {
		klog.Errorf("create service clients error: %v", err)
		return
	}

	cloud := ebs.NewVolcEngin(serviceClients, config.Region, config.Zone)

	maxVolumesPerNode := defaultMaxVolumesPerNode
	if instanceTypeName := metadataService.InstanceType(); instanceTypeName != "" {
		if instanceType, err := cloud.DescribeInstanceTypes(context.TODO(), instanceTypeName); err == nil {
			maxVolumesPerNode = int64(*instanceType.Volume.MaximumCount)
		} else {
			klog.Errorf("DescribeInstanceTypes to get maxVolumesPerNode fail, use defaultMaxVolumesPerNode %d, err: %s", defaultMaxVolumesPerNode, err)
		}
	} else {
		klog.Errorf("get instanceType from metadata server to get maxVolumesPerNode fail, use defaultMaxVolumesPerNode %d", defaultMaxVolumesPerNode)
	}
	reserveVolumesPerNode := int64(math.Floor(float64(maxVolumesPerNode) * reserveVolumesFactor))
	klog.Infof("maxVolumesPerNode: %d, reserveVolumesPerNode: %d", maxVolumesPerNode, reserveVolumesPerNode)

	driver := ebs.NewDriver(name, version, nodeId, maxVolumesPerNode, reserveVolumesPerNode)
	driver.Run(endpoint, cloud)
}

func main() {
	cmd := &cobra.Command{
		Use:   "ebsplugin",
		Short: "run ebs csi plugin",
		Run:   run,
	}

	cmd.Flags().StringVar(&name, "name", "ebs.csi.volcengine.com", "csi driver name")
	cmd.Flags().StringVar(&endpoint, "endpoint", "unix:///tmp/csi.sock", "csi endpoint")
	cmd.Flags().StringVar(&nodeId, "node-id", "", "node id")
	cmd.Flags().StringVar(&openApiCfgFile, "openapi-file", "/etc/csi/config/volc.yaml", "openapi config file path")
	cmd.Flags().StringVar(&metadataURL, "metadata-url", "http://100.96.0.96/volcstack/latest", "ecs metadata service url")
	cmd.Flags().BoolVar(&showVersion, "version", false, "Show version.")
	cmd.Flags().Float64Var(&reserveVolumesFactor, "reserve-volumes-factor", 0.3, "volume attach reserve factor per node, Rounded down. default 0.3")

	if err := cmd.Execute(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
