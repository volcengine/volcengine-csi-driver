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

package openapi

import (
	"io/ioutil"
	"os"

	"github.com/volcengine/volcengine-csi-driver/pkg/metadata"

	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

func ConfigVia(loaders ...CfgLoader) *Config {
	config := &Config{}
	for _, l := range loaders {
		_ = mergo.Merge(config, l.Load())
	}
	return config
}

type Credential struct {
	AccessKeyId     string `yaml:"AccessKeyId"`
	SecretAccessKey string `yaml:"SecretAccessKey"`
	SessionToken    string `yaml:"SessionToken"`
	AssumeRoleName  string `yaml:"AssumeRoleName"`
}

type Topology struct {
	Region string `yaml:"Region"`
	Zone   string `yaml:"Zone"`
}

type OpenApi struct {
	Host string `yaml:"Host"`
}

type Config struct {
	Credential `yaml:",inline"`
	Topology   `yaml:",inline"`
	OpenApi    `yaml:",inline"`
}

type CfgLoader interface {
	Load() (config *Config)
}

func EnvLoader() CfgLoader {
	return &envLoader{}
}

func FileLoader(filename string) CfgLoader {
	return &fileLoader{filename: filename}
}

func ServerLoader(metadataService metadata.MetadataService) CfgLoader {
	return &serverLoader{
		metadataService: metadataService,
	}
}

type envLoader struct{}

func (el *envLoader) Load() (config *Config) {
	credential := Credential{
		AccessKeyId:     os.Getenv("VOLC_ACCESSKEYID"),
		SecretAccessKey: os.Getenv("VOLC_SECRETACCESSKEY"),
		AssumeRoleName:  os.Getenv("VOLC_ASSUMEROLENAME"),
	}

	topology := Topology{
		Region: os.Getenv("VOLC_REGION"),
		Zone:   os.Getenv("VOLC_ZONE"),
	}

	openapi := OpenApi{
		Host: os.Getenv("VOLC_HOST"),
	}

	config = &Config{
		Credential: credential,
		Topology:   topology,
		OpenApi:    openapi,
	}

	klog.V(5).Infof("Config load from env, config = %+v", config)
	return config
}

type fileLoader struct {
	filename string
}

func (fl *fileLoader) Load() (config *Config) {
	data, err := ioutil.ReadFile(fl.filename)
	if err != nil {
		klog.Errorf("Read data from file %s failed: %+v", fl.filename, err)
		return
	}

	config = &Config{}
	if err = yaml.Unmarshal(data, config); err != nil {
		klog.Errorf("Unmarshal from file %s failed: %v", fl.filename, err)
		return
	}

	klog.V(5).Infof("Config load from file, config = %+v", config)
	return config
}

type serverLoader struct {
	metadataService metadata.MetadataService
}

func (sl *serverLoader) Load() (config *Config) {
	accessKeyId, secretAccessKey, sessionToken := sl.metadataService.Credential()
	credential := Credential{
		AccessKeyId:     accessKeyId,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}

	region := sl.metadataService.Region()
	zone := sl.metadataService.Zone()
	topology := Topology{
		Region: region,
		Zone:   zone,
	}

	openapi := OpenApi{
		Host: "",
	}

	config = &Config{
		Credential: credential,
		Topology:   topology,
		OpenApi:    openapi,
	}

	klog.V(5).Infof("Config load from server, config %+v", config)
	return config
}
