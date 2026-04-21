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

package sdkserver

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"agones.dev/agones/pkg/sdk"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/util/wait"
)

func testHTTPHealth(t *testing.T, url string, expectedResponse string, expectedStatus int) {
	// do a poll, because this code could run before the health check becomes live
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, 20*time.Second, true, func(_ context.Context) (done bool, err error) {
		resp, err := http.Get(url)
		if err != nil {
			logrus.WithError(err).Error("Error connecting to ", url)
			return false, nil
		}

		assert.NotNil(t, resp)
		if resp != nil {
			defer resp.Body.Close() // nolint: errcheck
			body, err := io.ReadAll(resp.Body)
			assert.Nil(t, err, "(%s) read response error should be nil: %v", url, err)
			assert.Equal(t, expectedStatus, resp.StatusCode, "url: %s", url)
			assert.Equal(t, []byte(expectedResponse), body, "(%s) response body should be '%s'", url, expectedResponse)
		}

		return true, nil
	})
	assert.Nil(t, err, "Timeout on %s health check, %v", url, err)
}

// emptyMockStream is the mock of the SDK_HealthServer for streaming
type emptyMockStream struct {
	msgs chan *sdk.Empty
}

func newEmptyMockStream() *emptyMockStream {
	return &emptyMockStream{msgs: make(chan *sdk.Empty)}
}

func (m *emptyMockStream) SendAndClose(*sdk.Empty) error {
	return nil
}

func (m *emptyMockStream) Recv() (*sdk.Empty, error) {
	empty, ok := <-m.msgs
	if ok {
		return empty, nil
	}
	return empty, io.EOF
}

func (m *emptyMockStream) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (m *emptyMockStream) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (m *emptyMockStream) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (m *emptyMockStream) Context() context.Context {
	panic("implement me")
}

func (m *emptyMockStream) SendMsg(_ interface{}) error {
	panic("implement me")
}

func (m *emptyMockStream) RecvMsg(_ interface{}) error {
	panic("implement me")
}

type gameServerMockStream struct {
	ctx  context.Context
	msgs chan *sdk.GameServer
}

// newGameServerMockStream implements SDK_WatchGameServerServer for testing
func newGameServerMockStream() *gameServerMockStream {
	return &gameServerMockStream{
		ctx:  context.Background(),
		msgs: make(chan *sdk.GameServer, 10),
	}
}

func (m *gameServerMockStream) Send(gs *sdk.GameServer) error {
	m.msgs <- gs
	return nil
}

func (*gameServerMockStream) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (*gameServerMockStream) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (*gameServerMockStream) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (m *gameServerMockStream) Context() context.Context {
	return m.ctx
}

func (*gameServerMockStream) SendMsg(_ interface{}) error {
	panic("implement me")
}

func (*gameServerMockStream) RecvMsg(_ interface{}) error {
	panic("implement me")
}
