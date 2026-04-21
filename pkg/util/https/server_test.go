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

package https

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testServer struct {
	server *httptest.Server
}

func (ts *testServer) Shutdown(_ context.Context) error {
	ts.server.Close()
	return nil
}

// ListenAndServeTLS(certFile, keyFile string) error
func (ts *testServer) ListenAndServeTLS(_, _ string) error {
	ts.server.StartTLS()
	return nil
}

func TestServerRun(t *testing.T) {
	t.Parallel()

	s := NewServer("", "", "")
	ts := &testServer{server: httptest.NewUnstartedServer(s.Mux)}
	s.tls = ts

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := s.Run(ctx, 0)
	assert.Nil(t, err)

	client := ts.server.Client()
	resp, err := client.Get(ts.server.URL + "/test")
	assert.Nil(t, err)
	defer resp.Body.Close() // nolint: errcheck
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, err = client.Get(ts.server.URL + "/")
	assert.Nil(t, err)
	defer resp.Body.Close() // nolint: errcheck
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
