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
	"io/ioutil"
	"net/http"

	"k8s.io/klog/v2"
)

func HttpGet(url string) (result string, err error) {
	resp, err := http.Get(url)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if err != nil {
		klog.Errorf("Get data from %s failed: %v", url, err)
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("Read data from response failed: %v", err)
		return "", err
	}
	return string(data), nil
}
