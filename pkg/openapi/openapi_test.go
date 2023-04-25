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
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/imdario/mergo"

	"github.com/stretchr/testify/assert"
)

const (
	gAccesskeyid     = "MyTestAccessKey"
	gSecretaccesskey = "MyTestSecretAccessKey"
	gSecuritytoken   = "MyTestSessionToken"
	gRegion          = "MyTestRegion"
	gZone            = "MyTestZone"
	gHost            = "MyTestOpenApiHost"
)

func TestEnvLoader_Load(t *testing.T) {
	assert.Empty(t, os.Setenv("VOLC_ACCESSKEYID", gAccesskeyid))
	assert.Empty(t, os.Setenv("VOLC_SECRETACCESSKEY", gSecretaccesskey))
	assert.Empty(t, os.Setenv("VOLC_REGION", gRegion))
	assert.Empty(t, os.Setenv("VOLC_ZONE", gZone))
	assert.Empty(t, os.Setenv("VOLC_HOST", gHost))

	t.Cleanup(func() {
		assert.Empty(t, os.Unsetenv("VOLC_ACCESSKEYID"))
		assert.Empty(t, os.Unsetenv("VOLC_SECRETACCESSKEY"))
		assert.Empty(t, os.Unsetenv("VOLC_REGION"))
		assert.Empty(t, os.Unsetenv("VOLC_ZONE"))
		assert.Empty(t, os.Unsetenv("VOLC_HOST"))
	})

	envLoader := EnvLoader()
	config := envLoader.Load()
	assert.Equal(t, gAccesskeyid, config.AccessKeyId)
	assert.Equal(t, gSecretaccesskey, config.SecretAccessKey)
	assert.Empty(t, config.SessionToken)
	assert.Equal(t, gRegion, config.Region)
	assert.Equal(t, gHost, config.Host)
}

var openApiCfgContent = fmt.Sprintf(`
AccessKeyId: %s
SecretAccessKey: %s
Region: %s
Zone: %s
Host: %s
`, gAccesskeyid, gSecretaccesskey, gRegion, gZone, gHost)

func TestFileLoader_Load(t *testing.T) {
	const yamlFileName = "/tmp/csi-volc-test-loader.yaml"
	err := ioutil.WriteFile(yamlFileName, []byte(openApiCfgContent), 0644)
	assert.Empty(t, err)

	fileLoader := FileLoader(yamlFileName)
	config := fileLoader.Load()
	assert.Equal(t, gAccesskeyid, config.AccessKeyId)
	assert.Equal(t, gSecretaccesskey, config.SecretAccessKey)
	assert.Empty(t, config.SessionToken)
	assert.Equal(t, gRegion, config.Region)
	assert.Equal(t, gHost, config.Host)

	err = os.Remove(yamlFileName)
	assert.Empty(t, err)
}

type fakeMetadataService struct{}

func (f *fakeMetadataService) NodeId() string {
	return "fake-node-id"
}

func (f *fakeMetadataService) Region() string {
	return gRegion
}

func (f *fakeMetadataService) Zone() string {
	return gZone
}

func (f *fakeMetadataService) Credential() (accessKeyId, secretAccessKey, sessionToken string) {
	return gAccesskeyid, gSecretaccesskey, gSecuritytoken
}

func (f *fakeMetadataService) Active() bool {
	return true
}

func TestServerLoader_Load(t *testing.T) {

	serverLoader := ServerLoader(&fakeMetadataService{})
	config := serverLoader.Load()
	assert.Equal(t, gAccesskeyid, config.AccessKeyId)
	assert.Equal(t, gSecretaccesskey, config.SecretAccessKey)
	assert.Equal(t, gSecuritytoken, config.SessionToken)
	assert.Equal(t, gRegion, config.Region)
	assert.Empty(t, config.Host)

}

func TestEnvFileLoader(t *testing.T) {
	assert.Empty(t, os.Setenv("VOLC_ACCESSKEYID", gAccesskeyid))
	assert.Empty(t, os.Setenv("VOLC_SECRETACCESSKEY", gSecretaccesskey))
	assert.Empty(t, os.Setenv("VOLC_ZONE", gZone))
	assert.Empty(t, os.Setenv("VOLC_HOST", gHost))

	t.Cleanup(func() {
		assert.Empty(t, os.Unsetenv("VOLC_ACCESSKEYID"))
		assert.Empty(t, os.Unsetenv("VOLC_SECRETACCESSKEY"))
		assert.Empty(t, os.Unsetenv("VOLC_REGION"))
		assert.Empty(t, os.Unsetenv("VOLC_ZONE"))
		assert.Empty(t, os.Unsetenv("VOLC_HOST"))
	})

	yamlContent := fmt.Sprintf(`
AccessKeyId: %s
Host: %s
`, "XXXXXXX", gHost)

	const yamlFileName = "/tmp/csi-volc-test-loader.yaml"
	err := ioutil.WriteFile(yamlFileName, []byte(yamlContent), 0644)
	assert.Empty(t, err)

	envLoader := EnvLoader()
	fileLoader := FileLoader(yamlFileName)

	config := &Config{}
	for _, l := range []CfgLoader{envLoader, fileLoader} {
		assert.Empty(t, mergo.Merge(config, l.Load()))
	}

	assert.Equal(t, gAccesskeyid, config.AccessKeyId)
	assert.Equal(t, gSecretaccesskey, config.SecretAccessKey)
	assert.Empty(t, config.SessionToken)
	assert.Empty(t, config.Region)
	assert.Equal(t, gHost, config.Host)

	err = os.Remove(yamlFileName)
	assert.Empty(t, err)
}

func TestAllLoader(t *testing.T) {
	assert.Empty(t, os.Setenv("VOLC_ACCESSKEYID", gAccesskeyid))
	t.Cleanup(func() {
		assert.Empty(t, os.Unsetenv("VOLC_ACCESSKEYID"))
		assert.Empty(t, os.Unsetenv("VOLC_SECRETACCESSKEY"))
		assert.Empty(t, os.Unsetenv("VOLC_REGION"))
		assert.Empty(t, os.Unsetenv("VOLC_ZONE"))
		assert.Empty(t, os.Unsetenv("VOLC_HOST"))
	})

	yamlContent := fmt.Sprintf(`
Host: %s
`, gHost)

	const yamlFileName = "/tmp/csi-volc-test-loader.yaml"
	err := ioutil.WriteFile(yamlFileName, []byte(yamlContent), 0644)
	assert.Empty(t, err)

	envLoader := EnvLoader()
	fileLoader := FileLoader(yamlFileName)
	serverLoader := ServerLoader(&fakeMetadataService{})

	config := &Config{}
	for _, l := range []CfgLoader{envLoader, fileLoader, serverLoader} {
		assert.Empty(t, mergo.Merge(config, l.Load()))
	}

	assert.Equal(t, gAccesskeyid, config.AccessKeyId)
	assert.Equal(t, gSecretaccesskey, config.SecretAccessKey)
	assert.Equal(t, gSecuritytoken, config.SessionToken)
	assert.Equal(t, gRegion, config.Region)
	assert.Equal(t, gHost, config.Host)

	err = os.Remove(yamlFileName)
	assert.Empty(t, err)
}
