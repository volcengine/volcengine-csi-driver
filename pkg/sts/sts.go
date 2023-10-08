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

package sts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/volcengine/volcengine-csi-driver/pkg/ebs/metrics"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/volcengine/volcengine-csi-driver/pkg/openapi"

	"github.com/volcengine/volcengine-go-sdk/service/storageebs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/client"
	"github.com/volcengine/volcengine-go-sdk/volcengine/credentials"
	"github.com/volcengine/volcengine-go-sdk/volcengine/custom"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
	"github.com/volcengine/volcengine-go-sdk/volcengine/universal"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

var (
	assumeRoleName          = "KubernetesNodeRoleForECS"
	tokenRefreshPeriod      = 5 * time.Minute
	tokenExpiredWindow      = 30 * time.Minute
	openHostAddress         = "open.volcengineapi.com"
	credentialServerAddress = "http://100.96.0.96/volcstack/latest/iam/security_credentials/"
)

type token struct {
	lock   sync.RWMutex
	auth   *Credential
	active bool
}

type Credential struct {
	ExpiredTime     time.Time `json:"ExpiredTime"`
	CurrentTime     time.Time `json:"CurrentTime"`
	AccessKeyId     string    `json:"AccessKeyId"`
	AccessKeySecret string    `json:"SecretAccessKey"`
	SecurityToken   string    `json:"SessionToken"`
}

type ServiceClients struct {
	config          *openapi.Config
	token           *token
	UniversalClient *universal.Universal
	EbsClient       *storageebs.STORAGEEBS
}

func NewServiceClients(config *openapi.Config) (*ServiceClients, error) {
	t := &token{
		auth: &Credential{
			AccessKeyId:     config.AccessKeyId,
			AccessKeySecret: config.SecretAccessKey,
		},
		active: false,
	}
	serviceClients := &ServiceClients{
		config: config,
		token:  t,
	}
	var universalClient *universal.Universal
	var ebsClient *storageebs.STORAGEEBS
	var volcConfig *volcengine.Config
	if config.Host != "" {
		openHostAddress = config.Host
	}
	if config.SecretAccessKey != "" && config.AccessKeyId != "" {
		klog.Info("NewServiceClients with user ak/sk.")
		volcConfig = volcengine.NewConfig().
			WithRegion(config.Region).
			WithCredentials(credentials.NewStaticCredentials(config.AccessKeyId, config.SecretAccessKey, "")).
			WithEndpoint(openHostAddress).
			WithMaxRetries(0).
			WithDisableSSL(true).
			WithLogger(newVolcLogger()).
			WithLogLevel(volcengine.LogInfoWithInputAndOutput)
	} else {
		//
		klog.Info("NewServiceClients with sts.")
		if config.AssumeRoleName != "" {
			assumeRoleName = config.AssumeRoleName
		}
		url := credentialServerAddress + assumeRoleName
		role, err := getRoleToken(url)
		if err != nil {
			return nil, err
		}
		t.auth = role
		t.active = true
		// init all client with token method.
		volcConfig = volcengine.NewConfig().
			WithRegion(config.Region).
			WithCredentials(credentials.NewStaticCredentials(role.AccessKeyId, role.AccessKeySecret, role.SecurityToken)).
			WithEndpoint(openHostAddress).
			WithMaxRetries(0).
			WithDisableSSL(true).
			WithLogger(newVolcLogger()).
			WithLogLevel(volcengine.LogInfoWithInputAndOutput).AddInterceptor(custom.SdkInterceptor{
			Before: func(info custom.RequestInfo) interface{} {
				return time.Now()
			},
			After: func(info custom.RequestInfo, i interface{}) {
				sendTime, ok := i.(time.Time)
				if !ok {
					klog.Errorf("parse sendTime error")
				}
				metrics.RecordEBSMetric(info.Name, info.Method, info.ClientInfo.APIVersion, time.Since(sendTime).Seconds(), info.Error)
				if metrics.IsErrorThrottle(info) {
					metrics.RecordEBSThrottlesMetric(info.Name, info.Method, info.ClientInfo.APIVersion)
					klog.InfoS("Got RequestLimitExceeded error on EBS request", "request", info.ClientInfo.ServiceName+"::"+info.Name)
				}
			},
		})
	}

	sess, err := session.NewSession(volcConfig)
	if err != nil {
		panic(err)
	}
	ebsClient = storageebs.New(sess)
	universalClient = universal.New(sess)
	if t.active {
		// refresh client periodically.
		go wait.Until(func() {
			t.refreshToken(serviceClients)
		}, tokenRefreshPeriod, nil)
	}
	serviceClients.UniversalClient = universalClient
	serviceClients.EbsClient = ebsClient

	return serviceClients, nil
}

func (t *token) expiredAt() time.Time {
	return t.auth.ExpiredTime.UTC().Add(-tokenExpiredWindow)
}

func (t *token) isTokenExpired() bool {
	return time.Now().UTC().After(t.expiredAt())
}

func (t *token) refreshToken(serviceClients *ServiceClients) {
	t.lock.Lock()
	defer t.lock.Unlock()
	// expired token, refresh it.
	if t.isTokenExpired() {
		klog.V(2).Infof("token is expired, now: %+v, current: %+v, expired: %+v, at: %+v", time.Now(), t.auth.CurrentTime, t.auth.ExpiredTime, t.expiredAt())
		url := credentialServerAddress + assumeRoleName
		role, err := getRoleToken(url)
		if err != nil {
			klog.Errorf("get role token error", err)
			return
		}
		t.auth = role
		// refresh client.
		// init all client with token method.
		config := volcengine.NewConfig().
			WithRegion(serviceClients.config.Region).
			WithCredentials(credentials.NewStaticCredentials(role.AccessKeyId, role.AccessKeySecret, role.SecurityToken)).
			WithEndpoint(openHostAddress).
			WithMaxRetries(0).WithDisableSSL(true).
			WithLogger(newVolcLogger()).
			WithLogLevel(volcengine.LogInfoWithInputAndOutput).AddInterceptor(custom.SdkInterceptor{
			Before: func(info custom.RequestInfo) interface{} {
				return time.Now()
			},
			After: func(info custom.RequestInfo, i interface{}) {
				sendTime, ok := i.(time.Time)
				if !ok {
					klog.Errorf("parse sendTime error")
				}
				metrics.RecordEBSMetric(info.Name, info.Method, info.ClientInfo.APIVersion, time.Since(sendTime).Seconds(), info.Error)
				if metrics.IsErrorThrottle(info) {
					metrics.RecordEBSThrottlesMetric(info.Name, info.Method, info.ClientInfo.APIVersion)
					klog.InfoS("Got RequestLimitExceeded error on EBS request", "request", info.ClientInfo.ServiceName+"::"+info.Name)
				}
			},
		})
		sess, err := session.NewSession(config)
		if err != nil {
			panic(err)
		}
		ebsClient := storageebs.New(sess)
		universalClient := universal.New(sess)
		serviceClients.UniversalClient = universalClient
		serviceClients.EbsClient = ebsClient
	}
}

func getRoleToken(url string) (*Credential, error) {
	resp, err := http.Get(url)
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	credential := &Credential{}
	err = json.Unmarshal(data, &credential)
	if err != nil {
		return nil, fmt.Errorf("ausumeRole unmarshal err: %s, raw: %s", err, string(data))
	}

	return credential, nil
}

func getGID() string {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return fmt.Sprintf("(t-%v)", n)
}

func newVolcLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

type Logger struct {
	logger *log.Logger
}

func (l *Logger) Log(fields ...interface{}) {
	if len(fields) > 0 {
		if v, ok := fields[0].(client.LogStruct); ok {
			logStruct := v
			switch logStruct.Type {
			case "Request":
				b, _ := json.Marshal(logStruct.Request)
				l.logger.Println(logStruct.Level, getGID(), logStruct.OperationName, "request", string(b), logStruct.Context)
			case "Response":
				b, _ := json.Marshal(logStruct.Response)
				l.logger.Println(logStruct.Level, getGID(), logStruct.OperationName, "response", string(b), logStruct.Context)
			}
		}
	}
}
