/*
Copyright 2017 The Kubernetes Authors.

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

package metrics

import (
	"sync"

	"github.com/volcengine/volcengine-go-sdk/volcengine/custom"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

var (
	ebsAPIMetric = metrics.NewHistogramVec(
		&metrics.HistogramOpts{
			Name:           "volc_api_request_duration_seconds",
			Help:           "Latency of VOLC API calls",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"action", "method", "version"})

	ebsAPIErrorMetric = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "volc_api_request_errors",
			Help:           "VOLC API errors",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"action", "method", "version"})

	ebsAPIThrottlesMetric = metrics.NewCounterVec(
		&metrics.CounterOpts{
			Name:           "volc_api_throttled_requests_total",
			Help:           "VOLC API throttled requests",
			StabilityLevel: metrics.ALPHA,
		},
		[]string{"action", "method", "version"})
)

func RecordEBSMetric(action, method, version string, timeTaken float64, err error) {
	if err != nil {
		ebsAPIErrorMetric.With(metrics.Labels{"action": action, "method": method, "version": version}).Inc()
	} else {
		ebsAPIMetric.With(metrics.Labels{"action": action, "method": method, "version": version}).Observe(timeTaken)
	}
}

func RecordEBSThrottlesMetric(action, method, version string) {
	ebsAPIThrottlesMetric.With(metrics.Labels{"action": action, "method": method, "version": version}).Inc()
}

// IsErrorThrottle returns whether the error is to be throttled based on its
// code. Returns false if the request has no Error set.
//
// Alias for the utility function IsErrorThrottle
func IsErrorThrottle(r custom.RequestInfo) bool {
	if r.Response != nil {
		switch r.Response.StatusCode {
		case
			429, // error caused due to too many requests
			502, // Bad Gateway error should be throttled
			503, // caused when service is unavailable
			504: // error occurred due to gateway timeout
			return true
		}
	}

	return request.IsErrorThrottle(r.Error)
}

var registerOnce sync.Once

func RegisterMetrics() {
	registerOnce.Do(func() {
		legacyregistry.MustRegister(ebsAPIMetric)
		legacyregistry.MustRegister(ebsAPIErrorMetric)
		legacyregistry.MustRegister(ebsAPIThrottlesMetric)
	})
}
