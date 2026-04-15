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

package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// TestRegisterTestSdkServer - test to verify
func TestRegisterTestSdkServer(t *testing.T) {
	t.Parallel()
	ctlConf := parseEnvFlags()
	grpcServer := grpc.NewServer()
	_, err := registerTestSdkServer(grpcServer, ctlConf)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx.Done()
	ctlConf.LocalFile = "@@"
	_, err = registerLocal(grpcServer, ctlConf)
	assert.Error(t, err, "Wrong file name should produce an error")
}

func TestHealthCheckWrapper(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		body     io.Reader
		expected int
	}{
		{"empty body", nil, http.StatusBadRequest},
		{"non-empty body", bytes.NewBuffer([]byte(`{}`)), http.StatusOK},
	}

	testWrapper := healthCheckWrapper(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testResponse := httptest.NewRecorder()
			testWrapper.ServeHTTP(testResponse, httptest.NewRequest("POST", "http://testServer/health", tc.body))
			assert.Equal(t, tc.expected, testResponse.Code)
		})
	}
}
