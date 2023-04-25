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
	goflag "flag"
	"os"

	"github.com/volcengine/volcengine-csi-driver/pkg/metadata"
	"github.com/volcengine/volcengine-csi-driver/pkg/tos"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

var (
	name        string
	endpoint    string
	nodeId      string
	metadataURL string
	version     string // Set by the build process
	showVersion = false

	rootCmd = &cobra.Command{
		Use:   "tosplugin rootCmd",
		Short: "run tos csi plugin",
		Long:  "run tos csi plugin",
	}
)

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.AddCommand(RunCmd())

	klog.InitFlags(nil)
	goflag.Parse()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
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
		nodeId = metadataService.NodeId()
		if nodeId == "" {
			klog.Error("get empty node id from metadata server")
			return
		}
	}

	tos.NewDriver(name, version, nodeId).Run(endpoint)
}

func RunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "run tos csi plugin",
		Run:   run,
	}

	runCmd.Flags().SortFlags = false
	runCmd.Flags().StringVar(&name, "name", tos.DefaultDriverName, "csi driver name")
	runCmd.Flags().StringVar(&endpoint, "endpoint", "unix:///tmp/csi.sock", "csi endpoint")
	runCmd.Flags().StringVar(&nodeId, "node-id", "", "node id")
	runCmd.Flags().StringVar(&metadataURL, "metadata-url", "http://100.96.0.96/volcstack/latest", "ecs metadata service url")
	runCmd.Flags().BoolVar(&showVersion, "version", false, "Show version.")
	return runCmd
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
