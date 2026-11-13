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

package processor

import (
	"context"
	"io"
	"testing"

	allocationpb "agones.dev/agones/pkg/allocation/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientStreamReceiveBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []*allocationpb.ProcessorMessage
		wantBatchID string
		wantErr     error
	}{
		{
			name: "other messages are skipped",
			messages: []*allocationpb.ProcessorMessage{
				{ClientId: "client-1"},
				{
					ClientId: "client-1",
					Payload:  &allocationpb.ProcessorMessage_Pull{Pull: &allocationpb.PullRequest{}},
				},
				batchMessage("b-1"),
			},
			wantBatchID: "b-1",
		},
		{
			name:    "end of the stream is returned",
			wantErr: io.EOF,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stream := newMockServerStream(t.Context())
			for _, msg := range tc.messages {
				stream.recvChan <- msg
			}
			close(stream.recvChan)

			batch, err := newClientStream("client-1", stream).receiveBatch()
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantBatchID, batch.GetBatchId())
		})
	}
}

func TestClientStreamRequestPull(t *testing.T) {
	t.Parallel()

	client := newClientStream("client-1", newMockServerStream(t.Context()))

	// nothing takes the pull requests, as when the sender is held by a slow client: asking again must not block
	asked := make(chan struct{})
	go func() {
		defer close(asked)
		client.requestPull()
		client.requestPull()
	}()
	receive(t, asked)

	// both asks are merged into a single pending pull request
	assert.Len(t, client.pulls, 1)
}

func TestClientStreamSendPulls(t *testing.T) {
	t.Parallel()

	t.Run("stops when the context is done", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		client := newClientStream("client-1", newMockServerStream(t.Context()))
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			client.sendPulls(ctx)
		}()

		cancel()

		receive(t, stopped)
	})

	t.Run("stops when a pull request cannot be sent", func(t *testing.T) {
		t.Parallel()

		stream := &failingSendStream{mockServerStream: newMockServerStream(t.Context())}
		client := newClientStream("client-1", stream)
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			client.sendPulls(t.Context())
		}()

		client.requestPull()

		receive(t, stopped)
		assert.EqualValues(t, 1, stream.sends.Load())
	})
}

func TestClientStreamRespond(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		failSend bool
	}{
		{
			name: "response is sent to the client",
		},
		{
			name:     "send error is returned",
			failSend: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := newMockServerStream(t.Context())
			var stream allocationpb.Processor_StreamBatchesServer = mock
			if tc.failSend {
				stream = &failingSendStream{mockServerStream: mock}
			}
			client := newClientStream("client-1", stream)

			err := client.respond(&allocationpb.BatchResponse{BatchId: "b-1"})
			if tc.failSend {
				require.ErrorIs(t, err, errSendFailed)
				return
			}
			require.NoError(t, err)

			msg := receive(t, mock.sendChan)
			assert.Equal(t, "client-1", msg.GetClientId())
			assert.Equal(t, "b-1", msg.GetBatchResponse().GetBatchId())
		})
	}
}
