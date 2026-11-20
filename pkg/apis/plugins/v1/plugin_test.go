// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"testing"
	"time"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admregv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestURLConfigurationValidateURLConfiguration(t *testing.T) {
	t.Parallel()

	url := "http://example.com/plugin.wasm"
	fldPath := field.NewPath("url")

	testCases := map[string]struct {
		config   *URLConfiguration
		expected []string
	}{
		"nil": {
			config:   nil,
			expected: []string{"url: Required value: url configuration is missing"},
		},
		"neither url or service": {
			config:   &URLConfiguration{},
			expected: []string{"url: Required value: url should be provided"},
		},
		"both url and service": {
			config:   &URLConfiguration{URL: &url, Service: &admregv1.ServiceReference{Name: "service", Namespace: "default"}},
			expected: []string{"url.url: Duplicate value: \"service and url cannot be used simultaneously\""},
		},
		"invalid caBundle": {
			config:   &URLConfiguration{URL: &url, CABundle: []byte("invalid")},
			expected: []string{"url.caBundle: Invalid value"},
		},
		"valid url": {
			config: &URLConfiguration{URL: &url},
		},
		"valid service": {
			config: &URLConfiguration{Service: &admregv1.ServiceReference{Name: "service", Namespace: "default"}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			allErrs := tc.config.ValidateURLConfiguration(fldPath)
			require.Len(t, allErrs, len(tc.expected))
			for i, e := range tc.expected {
				assert.Contains(t, allErrs[i].Error(), e)
			}
		})
	}
}

func TestWasmPolicyValidateWasmPolicy(t *testing.T) {
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	url := "http://example.com/plugin.wasm"
	fldPath := field.NewPath("wasm")

	testCases := map[string]struct {
		features string
		policy   *WasmPolicy
		expected []string
	}{
		"nil": {
			features: "WasmAllocation=true",
			policy:   nil,
			expected: []string{"wasm: Required value: wasm policy config params are missing"},
		},
		"feature gate disabled": {
			features: "WasmAllocation=false&WasmAutoscaler=true",
			policy:   &WasmPolicy{From: WasmFrom{URL: &URLConfiguration{URL: &url}}},
			expected: []string{"wasm: Forbidden: feature WasmAllocation must be enabled"},
		},
		"missing from": {
			features: "WasmAllocation=true",
			policy:   &WasmPolicy{},
			expected: []string{"wasm.from: Required value: wasm from configuration is missing"},
		},
		"invalid url configuration": {
			features: "WasmAllocation=true",
			policy:   &WasmPolicy{From: WasmFrom{URL: &URLConfiguration{}}},
			expected: []string{"wasm.from.url: Required value: url should be provided"},
		},
		"valid": {
			features: "WasmAllocation=true",
			policy:   &WasmPolicy{Function: "allocate", From: WasmFrom{URL: &URLConfiguration{URL: &url}}},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, runtime.ParseFeatures(tc.features))

			allErrs := tc.policy.ValidateWasmPolicy(fldPath, runtime.FeatureWasmAllocation)
			require.Len(t, allErrs, len(tc.expected))
			for i, e := range tc.expected {
				assert.Contains(t, allErrs[i].Error(), e)
			}
		})
	}
}

func TestURLConfigurationBuildURL(t *testing.T) {
	t.Parallel()

	validURL := "http://example.com/plugin.wasm"
	emptyURL := ""
	path := "plugin.wasm"
	port := int32(9090)

	testCases := map[string]struct {
		config   *URLConfiguration
		expected string
		err      string
	}{
		"both url and service": {
			config: &URLConfiguration{URL: &validURL, Service: &admregv1.ServiceReference{Name: "service"}},
			err:    "service and URL cannot be used simultaneously",
		},
		"empty url": {
			config: &URLConfiguration{URL: &emptyURL},
			err:    "URL was not provided",
		},
		"neither url or service": {
			config: &URLConfiguration{},
			err:    "service was not provided, either URL or Service must be provided",
		},
		"empty service name": {
			config: &URLConfiguration{Service: &admregv1.ServiceReference{Namespace: "default"}},
			err:    "service name was not provided",
		},
		"url": {
			config:   &URLConfiguration{URL: &validURL},
			expected: validURL,
		},
		"service with defaults": {
			config:   &URLConfiguration{Service: &admregv1.ServiceReference{Name: "service"}},
			expected: "http://service.default.svc:8000",
		},
		"service with path and port": {
			config:   &URLConfiguration{Service: &admregv1.ServiceReference{Name: "service", Namespace: "ns", Path: &path, Port: &port}},
			expected: "http://service.ns.svc:9090/plugin.wasm",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			u, err := tc.config.BuildURL()
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, u.String())
		})
	}
}

func TestURLConfigurationHTTPClient(t *testing.T) {
	t.Parallel()

	url := "http://example.com/plugin.wasm"

	client, err := (&URLConfiguration{URL: &url}).HTTPClient(time.Second)
	require.NoError(t, err)
	assert.Equal(t, time.Second, client.Timeout)

	_, err = (&URLConfiguration{URL: &url, CABundle: []byte("invalid")}).HTTPClient(time.Second)
	require.ErrorContains(t, err, "no certs were appended from caBundle")
}

func TestCreateURL(t *testing.T) {
	t.Parallel()
	var nonStandardPort int32 = 8888

	var testCases = []struct {
		description string
		scheme      string
		name        string
		namespace   string
		path        string
		port        *int32
		expected    string
	}{
		{
			description: "OK, path not empty",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "scale",
			expected:    "http://service1.default.svc:8000/scale",
		},
		{
			description: "OK, path not empty with slash",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "/scale",
			expected:    "http://service1.default.svc:8000/scale",
		},
		{
			description: "OK, path is empty",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "",
			expected:    "http://service1.default.svc:8000",
		},
		{
			description: "OK, port specified",
			scheme:      "http",
			name:        "service1",
			namespace:   "default",
			path:        "scale",
			port:        &nonStandardPort,
			expected:    "http://service1.default.svc:8888/scale",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			res := createURL(tc.scheme, tc.name, tc.namespace, tc.path, tc.port)

			if assert.NotNil(t, res) {
				assert.Equal(t, tc.expected, res.String())
			}
		})
	}
}
