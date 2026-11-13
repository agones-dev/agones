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
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	allocationpb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"
	testclocks "k8s.io/utils/clock/testing"
)

func TestAddClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		clientIDs []string
		wantLen   int
	}{
		{
			name:      "single client",
			clientIDs: []string{"client-1"},
			wantLen:   1,
		},
		{
			name:      "multiple clients",
			clientIDs: []string{"client-1", "client-2", "client-3"},
			wantLen:   3,
		},
		{
			name:      "duplicate client overwrites",
			clientIDs: []string{"client-1", "client-1"},
			wantLen:   1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := &Handler{}
			h.clients = make(map[string]*clientStream)
			stream := newClientStream("client-1", newMockServerStream(t.Context()))

			for _, id := range tc.clientIDs {
				h.addClient(id, stream)
			}

			h.mu.RLock()
			defer h.mu.RUnlock()
			assert.Len(t, h.clients, tc.wantLen)
		})
	}
}

func TestRemoveClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		initialIDs []string
		removeID   string
		wantLen    int
	}{
		{
			name:       "remove existing client",
			initialIDs: []string{"client-1", "client-2"},
			removeID:   "client-1",
			wantLen:    1,
		},
		{
			name:       "remove non-existing client is a no-op",
			initialIDs: []string{"client-1"},
			removeID:   "client-unknown",
			wantLen:    1,
		},
		{
			name:       "remove last client leaves empty map",
			initialIDs: []string{"client-1"},
			removeID:   "client-1",
			wantLen:    0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := &Handler{
				clients: make(map[string]*clientStream),
			}
			stream := newClientStream("client-1", newMockServerStream(t.Context()))
			for _, id := range tc.initialIDs {
				h.addClient(id, stream)
			}

			h.removeClient(tc.removeID, stream)

			h.mu.RLock()
			defer h.mu.RUnlock()
			assert.Len(t, h.clients, tc.wantLen)
		})
	}

	t.Run("removing a stale stream keeps the stream the client reconnected with", func(t *testing.T) {
		t.Parallel()

		h := &Handler{
			clients: make(map[string]*clientStream),
		}
		stale := newClientStream("client-1", newMockServerStream(t.Context()))
		current := newClientStream("client-1", newMockServerStream(t.Context()))
		h.addClient("client-1", stale)
		h.addClient("client-1", current)

		h.removeClient("client-1", stale)

		h.mu.RLock()
		defer h.mu.RUnlock()
		assert.Same(t, current, h.clients["client-1"])
	})
}

func TestProcessAllocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		allocateFunc  func(ctx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
		wantResponse  bool
		wantErrorCode codes.Code
		wantErrorMsg  string
	}{
		{
			name: "successful allocation",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-1",
						Address:        "192.168.1.1",
						NodeName:       "node-1",
						Ports: []agonesv1.GameServerStatusPort{
							{Name: "game", Port: 7777},
						},
					},
				}, nil
			},
			wantResponse: true,
		},
		{
			name: "unallocated state returns ResourceExhausted",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State: allocationv1.GameServerAllocationUnAllocated,
					},
				}, nil
			},
			wantErrorCode: codes.ResourceExhausted,
			wantErrorMsg:  "there is no available GameServer to allocate",
		},
		{
			name: "contention state returns Aborted",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State: allocationv1.GameServerAllocationContention,
					},
				}, nil
			},
			wantErrorCode: codes.Aborted,
			wantErrorMsg:  "too many concurrent requests have overwhelmed the system",
		},
		{
			name: "allocator returns error",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return nil, errors.New("allocation failed")
			},
			wantErrorCode: codes.Internal,
			wantErrorMsg:  "allocation failed",
		},
		{
			name: "allocator returns grpc status error",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return nil, status.Error(codes.NotFound, "no game servers available")
			},
			wantErrorCode: codes.NotFound,
			wantErrorMsg:  "no game servers available",
		},
		{
			name: "allocator returns metav1.Status (e.g. validation failure)",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "GameServerAllocation is invalid",
					Code:    http.StatusUnprocessableEntity,
				}, nil
			},
			wantErrorCode: codes.InvalidArgument,
			wantErrorMsg:  "GameServerAllocation is invalid",
		},
		{
			name: "allocator returns unexpected type",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &agonesv1.GameServer{}, nil
			},
			wantErrorCode: codes.Internal,
			wantErrorMsg:  "internal server error - Bad GSA format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			h := newTestHandler(ctx, tc.allocateFunc)

			req := &allocationpb.AllocationRequest{Namespace: "default"}
			result := h.processAllocation(ctx, req)

			if tc.wantResponse {
				assert.NotNil(t, result.response)
				assert.Nil(t, result.error)
			} else {
				assert.Nil(t, result.response)
				require.NotNil(t, result.error)
				assert.Equal(t, int32(tc.wantErrorCode), result.error.GetCode())
				assert.Contains(t, result.error.GetMessage(), tc.wantErrorMsg)
			}
		})
	}
}

func TestProcessAllocationsConcurrently(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		allocateFunc func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
		requestCount int
		wantErrors   []bool
	}{
		{
			name: "all requests succeed",
			allocateFunc: func(_ context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					ObjectMeta: metav1.ObjectMeta{Namespace: gsa.Namespace},
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-concurrent",
					},
				}, nil
			},
			requestCount: 5,
			wantErrors:   []bool{false, false, false, false, false},
		},
		{
			name: "all requests fail",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return nil, errors.New("no capacity")
			},
			requestCount: 3,
			wantErrors:   []bool{true, true, true},
		},
		{
			name:         "empty request list",
			allocateFunc: nil,
			requestCount: 0,
			wantErrors:   []bool{},
		},
		{
			name: "single request succeeds",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-single",
					},
				}, nil
			},
			requestCount: 1,
			wantErrors:   []bool{false},
		},
		{
			name: "mixed results based on namespace",
			allocateFunc: func(_ context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				if gsa.Namespace == "fail" {
					return nil, errors.New("allocation failed")
				}
				return &allocationv1.GameServerAllocation{
					ObjectMeta: metav1.ObjectMeta{Namespace: gsa.Namespace},
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-ok",
					},
				}, nil
			},
			requestCount: -1,
			wantErrors:   []bool{false, true, false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			allocFunc := tc.allocateFunc
			if allocFunc == nil {
				allocFunc = func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
					return nil, nil
				}
			}
			h := newTestHandler(ctx, allocFunc)

			var requestWrappers []*allocationpb.RequestWrapper
			if tc.requestCount == -1 {
				requestWrappers = []*allocationpb.RequestWrapper{
					{RequestId: "req-0", Request: &allocationpb.AllocationRequest{Namespace: "default"}},
					{RequestId: "req-1", Request: &allocationpb.AllocationRequest{Namespace: "fail"}},
					{RequestId: "req-2", Request: &allocationpb.AllocationRequest{Namespace: "other"}},
				}
			} else {
				requestWrappers = make([]*allocationpb.RequestWrapper, tc.requestCount)
				for i := range requestWrappers {
					requestWrappers[i] = &allocationpb.RequestWrapper{
						RequestId: fmt.Sprintf("req-%d", i),
						Request:   &allocationpb.AllocationRequest{Namespace: "default"},
					}
				}
			}

			results := h.processAllocationsConcurrently(ctx, requestWrappers)

			require.Len(t, results, len(tc.wantErrors))
			for i, result := range results {
				if tc.wantErrors[i] {
					assert.NotNil(t, result.error, "result %d should have an error", i)
					assert.Nil(t, result.response, "result %d should not have a response", i)
				} else {
					assert.Nil(t, result.error, "result %d should not have an error", i)
				}
			}
		})
	}
}

func TestSubmitBatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		allocateFunc func(ctx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
		requestCount int
		wantBatchID  string
		wantErrors   []bool
	}{
		{
			name: "all requests succeed",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-1",
					},
				}, nil
			},
			requestCount: 3,
			wantBatchID:  "batch-1",
			wantErrors:   []bool{false, false, false},
		},
		{
			name: "all requests fail",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return nil, errors.New("no capacity")
			},
			requestCount: 2,
			wantBatchID:  "batch-err",
			wantErrors:   []bool{true, true},
		},
		{
			name: "mixed success and failure based on namespace",
			allocateFunc: func(_ context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				if gsa.Namespace == "fail" {
					return nil, errors.New("no capacity")
				}
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-mixed",
					},
				}, nil
			},
			requestCount: -1,
			wantBatchID:  "batch-mixed",
			wantErrors:   []bool{false, true, false},
		},
		{
			name:         "empty batch",
			allocateFunc: nil,
			requestCount: 0,
			wantBatchID:  "batch-empty",
			wantErrors:   []bool{},
		},
		{
			name: "single request succeeds",
			allocateFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-solo",
					},
				}, nil
			},
			requestCount: 1,
			wantBatchID:  "batch-single",
			wantErrors:   []bool{false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			allocFunc := tc.allocateFunc
			if allocFunc == nil {
				allocFunc = func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
					return nil, nil
				}
			}
			h := newTestHandler(ctx, allocFunc)

			var requestWrappers []*allocationpb.RequestWrapper
			if tc.requestCount == -1 {
				requestWrappers = []*allocationpb.RequestWrapper{
					{RequestId: "req-0", Request: &allocationpb.AllocationRequest{Namespace: "default"}},
					{RequestId: "req-1", Request: &allocationpb.AllocationRequest{Namespace: "fail"}},
					{RequestId: "req-2", Request: &allocationpb.AllocationRequest{Namespace: "other"}},
				}
			} else {
				requestWrappers = make([]*allocationpb.RequestWrapper, tc.requestCount)
				for i := range requestWrappers {
					requestWrappers[i] = &allocationpb.RequestWrapper{
						RequestId: fmt.Sprintf("req-%d", i),
						Request:   &allocationpb.AllocationRequest{Namespace: "default"},
					}
				}
			}

			resp := h.submitBatch(ctx, tc.wantBatchID, requestWrappers)

			assert.Equal(t, tc.wantBatchID, resp.GetBatchId())
			require.Len(t, resp.GetResponses(), len(tc.wantErrors))

			for i, wrapper := range resp.GetResponses() {
				assert.Equal(t, fmt.Sprintf("req-%d", i), wrapper.GetRequestId())
				if tc.wantErrors[i] {
					assert.NotNil(t, wrapper.GetError(), "request %d should have error", i)
					assert.Nil(t, wrapper.GetResponse(), "request %d should not have response", i)
				} else {
					assert.Nil(t, wrapper.GetError(), "request %d should not have error", i)
				}
			}
		})
	}
}

func TestReceiveClientID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		messages     []*allocationpb.ProcessorMessage
		wantClientID string
		wantErr      error
	}{
		{
			name:         "client ID is taken from the first message",
			messages:     []*allocationpb.ProcessorMessage{{ClientId: "client-1"}, {ClientId: "client-2"}},
			wantClientID: "client-1",
		},
		{
			name:     "empty client ID is rejected",
			messages: []*allocationpb.ProcessorMessage{{}},
			wantErr:  status.Error(codes.InvalidArgument, "clientID is required"),
		},
		{
			name:    "receive error is returned",
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

			clientID, err := receiveClientID(stream)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantClientID, clientID)
		})
	}
}

func TestStreamBatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []*allocationpb.ProcessorMessage
		allocFunc   func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
		wantErr     bool
		wantClients int
		validate    func(t *testing.T, sent []*allocationpb.ProcessorMessage)
	}{
		{
			name:        "recv error on first message closes stream",
			wantErr:     true,
			wantClients: 0,
		},
		{
			name: "empty clientID closes stream with InvalidArgument error",
			messages: []*allocationpb.ProcessorMessage{
				{ClientId: ""},
			},
			wantErr:     true,
			wantClients: 0,
		},
		{
			name: "registers client then handles batch request",
			messages: []*allocationpb.ProcessorMessage{
				{ClientId: "client-1"},
				{
					ClientId: "client-1",
					Payload: &allocationpb.ProcessorMessage_BatchRequest{
						BatchRequest: &allocationpb.BatchRequest{
							BatchId: "b-1",
							Requests: []*allocationpb.RequestWrapper{
								{
									RequestId: "r-1",
									Request:   &allocationpb.AllocationRequest{Namespace: "default"},
								},
							},
						},
					},
				},
			},
			allocFunc: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
				return &allocationv1.GameServerAllocation{
					Status: allocationv1.GameServerAllocationStatus{
						State:          allocationv1.GameServerAllocationAllocated,
						GameServerName: "gs-1",
					},
				}, nil
			},
			wantErr:     true,
			wantClients: 0,
			validate: func(t *testing.T, sent []*allocationpb.ProcessorMessage) {
				require.Len(t, sent, 1)
				batchResp := sent[0].GetBatchResponse()
				require.NotNil(t, batchResp)
				assert.Equal(t, "b-1", batchResp.GetBatchId())
				require.Len(t, batchResp.GetResponses(), 1)
				assert.Equal(t, "r-1", batchResp.GetResponses()[0].GetRequestId())
				assert.NotNil(t, batchResp.GetResponses()[0].GetResponse())
			},
		},
		{
			name: "nil payload is skipped",
			messages: []*allocationpb.ProcessorMessage{
				{ClientId: "client-1"},
				{ClientId: "client-1", Payload: nil},
			},
			wantErr:     true,
			wantClients: 0,
		},
		{
			name: "non-batch payload is skipped",
			messages: []*allocationpb.ProcessorMessage{
				{ClientId: "client-1"},
				{
					ClientId: "client-1",
					Payload:  &allocationpb.ProcessorMessage_Pull{Pull: &allocationpb.PullRequest{Message: "pull"}},
				},
			},
			wantErr:     true,
			wantClients: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			allocFunc := tc.allocFunc
			if allocFunc == nil {
				allocFunc = func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
					return &allocationv1.GameServerAllocation{
						Status: allocationv1.GameServerAllocationStatus{
							State:          allocationv1.GameServerAllocationAllocated,
							GameServerName: "gs-default",
						},
					}, nil
				}
			}
			h := newTestHandler(ctx, allocFunc)

			stream := newMockServerStream(ctx)

			go func() {
				for _, msg := range tc.messages {
					stream.recvChan <- msg
				}
				close(stream.recvChan)
			}()

			err := receive(t, runStreamBatches(h, stream))

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			h.mu.RLock()
			assert.Len(t, h.clients, tc.wantClients)
			h.mu.RUnlock()

			if tc.validate != nil {
				var sent []*allocationpb.ProcessorMessage
				for {
					select {
					case msg := <-stream.sendChan:
						sent = append(sent, msg)
					default:
						tc.validate(t, sent)
						return
					}
				}
			}
		})
	}
}

func TestStartPullRequestTicker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		clientCount int
	}{
		{
			name:        "sends pull to connected clients",
			clientCount: 2,
		},
		{
			name:        "no clients means no pulls sent",
			clientCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			h, fc := newPullingTestHandler(t)

			streams := make([]*mockServerStream, tc.clientCount)
			for i := range streams {
				streams[i] = newMockServerStream(ctx)
				streams[i].recvChan <- &allocationpb.ProcessorMessage{ClientId: fmt.Sprintf("client-%d", i)}
				runStreamBatches(h, streams[i])
			}
			waitForClients(t, h, tc.clientCount)

			// every client is pulled on the same tick
			fc.Step(h.pullInterval)
			for i, stream := range streams {
				msg := receive(t, stream.sendChan)
				assert.NotNil(t, msg.GetPull(), "client-%d should receive a pull message", i)
				assert.Equal(t, fmt.Sprintf("client-%d", i), msg.GetClientId())
			}
		})
	}
}

func TestStartPullRequestTickerSlowClientDoesNotDelayOthers(t *testing.T) {
	t.Parallel()

	h, fc := newPullingTestHandler(t)
	slowCtx, disconnectSlow := context.WithCancel(t.Context())
	slow := newControlledSendStream(slowCtx)
	slow.recvChan <- &allocationpb.ProcessorMessage{ClientId: "slow"}
	slowDone := runStreamBatches(h, slow)
	fast := newMockServerStream(t.Context())
	fast.recvChan <- &allocationpb.ProcessorMessage{ClientId: "fast"}
	runStreamBatches(h, fast)
	waitForClients(t, h, 2)

	// the slow client never reads its first pull request
	fc.Step(h.pullInterval)
	require.NotNil(t, nextSend(t, slow).message.GetPull())
	assert.NotNil(t, receive(t, fast.sendChan).GetPull())

	// the slow client has a pull request waiting from the second tick, yet the fast client is still pulled on every tick
	for range 3 {
		fc.Step(h.pullInterval)
		assert.NotNil(t, receive(t, fast.sendChan).GetPull())
	}

	// once the slow client is gone, its handler returns and it is unregistered
	disconnectSlow()
	require.ErrorIs(t, receive(t, slowDone), context.Canceled)
	waitForClients(t, h, 1)
}

func TestStreamBatchesBlockedBatchDoesNotDelayOthers(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	release := make(chan struct{})
	h := newTestHandler(ctx, func(_ context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
		if gsa.Namespace == "slow" {
			<-release
		}
		return allocatedGSA(), nil
	})
	stream := newMockServerStream(ctx)
	stream.recvChan <- &allocationpb.ProcessorMessage{ClientId: "client-1"}
	stream.recvChan <- batchMessage("slow")
	stream.recvChan <- batchMessage("fast")
	runStreamBatches(h, stream)

	// the slow batch is still allocating, yet the fast one is answered
	assert.Equal(t, "fast", receive(t, stream.sendChan).GetBatchResponse().GetBatchId())

	close(release)
	assert.Equal(t, "slow", receive(t, stream.sendChan).GetBatchResponse().GetBatchId())
}

func TestStreamBatchesBatchTimeout(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// as the allocator does, wait for the allocation until the context of the batch is done
	h := newTestHandler(ctx, func(ctx context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	})
	h.batchTimeout = 50 * time.Millisecond
	stream := newMockServerStream(ctx)
	stream.recvChan <- &allocationpb.ProcessorMessage{ClientId: "client-1"}
	stream.recvChan <- batchMessage("b-1")
	// the client has closed its side, and the handler still returns once the timeout ends the batch
	close(stream.recvChan)
	done := runStreamBatches(h, stream)

	response := receive(t, stream.sendChan).GetBatchResponse()
	assert.Equal(t, "b-1", response.GetBatchId())
	require.Len(t, response.GetResponses(), 1)
	assert.Equal(t, int32(codes.DeadlineExceeded), response.GetResponses()[0].GetError().GetCode())
	require.ErrorIs(t, receive(t, done), io.EOF)
}

func TestStreamBatchesClientGoneCancelsBatches(t *testing.T) {
	t.Parallel()

	ctx, disconnect := context.WithCancel(t.Context())
	allocating := make(chan struct{})
	h := newTestHandler(t.Context(), func(ctx context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
		close(allocating)
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	})
	stream := newMockServerStream(ctx)
	stream.recvChan <- &allocationpb.ProcessorMessage{ClientId: "client-1"}
	stream.recvChan <- batchMessage("b-1")
	done := runStreamBatches(h, stream)
	receive(t, allocating)

	// the client goes away mid-allocation, long before the batch timeout; the handler waits for its
	// batches, so it can only return once the batch is cancelled
	disconnect()
	require.ErrorIs(t, receive(t, done), context.Canceled)
}

func TestStreamBatchesSerialisesSends(t *testing.T) {
	t.Parallel()

	h := newTestHandler(t.Context(), func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
		return allocatedGSA(), nil
	})
	stream := newControlledSendStream(t.Context())
	stream.recvChan <- &allocationpb.ProcessorMessage{ClientId: "client-1"}
	stream.recvChan <- batchMessage("b-1")
	stream.recvChan <- batchMessage("b-2")
	runStreamBatches(h, stream)

	// the client is slow to read the first response, while the other response and a pull request wait for their turn
	held := nextSend(t, stream)
	go h.sendPullRequestsToClients()
	assert.Never(t, func() bool { return len(stream.sends) > 0 }, 50*time.Millisecond, 5*time.Millisecond,
		"a send must not start while another one is in progress")

	// once it is read, the other response and the pull request are sent
	close(held.release)
	for range 2 {
		close(nextSend(t, stream).release)
	}
}

func TestStreamBatchesWaitsForResponsesBeforeReturning(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	h := newTestHandler(ctx, func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
		return allocatedGSA(), nil
	})
	stream := newControlledSendStream(ctx)
	stream.recvChan <- &allocationpb.ProcessorMessage{ClientId: "client-1"}
	stream.recvChan <- batchMessage("b-1")
	close(stream.recvChan)
	done := runStreamBatches(h, stream)

	// the receive side is over, but the response is still held in Send
	held := nextSend(t, stream)
	assert.Equal(t, "b-1", held.message.GetBatchResponse().GetBatchId())
	assert.Never(t, func() bool { return len(done) > 0 }, 50*time.Millisecond, 5*time.Millisecond,
		"handler returned while a response was still being sent")

	close(held.release)
	require.ErrorIs(t, receive(t, done), io.EOF)
}

// newTestHandler creates a Handler with a mock allocator.
func newTestHandler(_ context.Context, allocFunc func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error)) *Handler {
	return &Handler{
		allocator:                 &mockAllocator{allocateFunc: allocFunc},
		clients:                   make(map[string]*clientStream),
		grpcUnallocatedStatusCode: codes.ResourceExhausted,
		batchTimeout:              batchProcessingTimeout,
		pullInterval:              100 * time.Millisecond,
		clock:                     clock.RealClock{},
	}
}

// newPullingTestHandler creates a Handler whose pull request ticker is started, and ticks
// each time the returned clock is stepped by the pull interval.
func newPullingTestHandler(t *testing.T) (*Handler, *testclocks.FakeClock) {
	t.Helper()

	h := newTestHandler(t.Context(), nil)
	fc := testclocks.NewFakeClock(time.Now())
	h.clock = fc
	h.StartPullRequestTicker(t.Context())
	require.Eventually(t, fc.HasWaiters, 5*time.Second, time.Millisecond, "ticker goroutine should register")
	return h, fc
}

// waitForClients waits for the given number of clients to be registered.
func waitForClients(t *testing.T, h *Handler, count int) {
	t.Helper()

	require.Eventually(t, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.clients) == count
	}, 5*time.Second, time.Millisecond, "%d clients should be registered", count)
}

type mockAllocator struct {
	allocateFunc func(ctx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
}

func (m *mockAllocator) Allocate(ctx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
	return m.allocateFunc(ctx, gsa)
}

type mockServerStream struct {
	recvChan chan *allocationpb.ProcessorMessage
	sendChan chan *allocationpb.ProcessorMessage
	ctx      context.Context
}

func newMockServerStream(ctx context.Context) *mockServerStream {
	return &mockServerStream{
		recvChan: make(chan *allocationpb.ProcessorMessage, 10),
		sendChan: make(chan *allocationpb.ProcessorMessage, 10),
		ctx:      ctx,
	}
}

func (m *mockServerStream) Send(msg *allocationpb.ProcessorMessage) error {
	select {
	case m.sendChan <- msg:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *mockServerStream) Recv() (*allocationpb.ProcessorMessage, error) {
	select {
	case msg, ok := <-m.recvChan:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *mockServerStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockServerStream) SendHeader(metadata.MD) error { return nil }
func (m *mockServerStream) SetTrailer(metadata.MD)       {}
func (m *mockServerStream) Context() context.Context     { return m.ctx }
func (m *mockServerStream) SendMsg(any) error            { return nil }
func (m *mockServerStream) RecvMsg(any) error            { return nil }

var errSendFailed = errors.New("send failed")

// failingSendStream is a stream that cannot be written to, and counts the attempts to do so
type failingSendStream struct {
	*mockServerStream
	sends atomic.Int32
}

func (f *failingSendStream) Send(_ *allocationpb.ProcessorMessage) error {
	f.sends.Add(1)
	return errSendFailed
}

// sendCall is a Send in progress on a controlledSendStream
type sendCall struct {
	message *allocationpb.ProcessorMessage
	// release lets the Send return, as if the client had read the message
	release chan struct{}
}

// controlledSendStream holds every Send until the test releases it
type controlledSendStream struct {
	*mockServerStream
	sends chan sendCall
}

func newControlledSendStream(ctx context.Context) *controlledSendStream {
	return &controlledSendStream{
		mockServerStream: newMockServerStream(ctx),
		sends:            make(chan sendCall, 10),
	}
}

func (s *controlledSendStream) Send(msg *allocationpb.ProcessorMessage) error {
	call := sendCall{message: msg, release: make(chan struct{})}
	select {
	case s.sends <- call:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	select {
	case <-call.release:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

// nextSend waits for the next Send to start on the stream
func nextSend(t *testing.T, stream *controlledSendStream) sendCall {
	t.Helper()
	return receive(t, stream.sends)
}

// receive waits for the next value from the channel, and fails the test if it does not arrive within 5 seconds
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for channel value")
		var zero T
		return zero
	}
}

func allocatedGSA() k8sruntime.Object {
	return &allocationv1.GameServerAllocation{
		Status: allocationv1.GameServerAllocationStatus{
			State:          allocationv1.GameServerAllocationAllocated,
			GameServerName: "gs-1",
		},
	}
}

// batchMessage builds a single request batch, whose namespace is the batch ID.
func batchMessage(batchID string) *allocationpb.ProcessorMessage {
	return &allocationpb.ProcessorMessage{
		ClientId: "client-1",
		Payload: &allocationpb.ProcessorMessage_BatchRequest{
			BatchRequest: &allocationpb.BatchRequest{
				BatchId: batchID,
				Requests: []*allocationpb.RequestWrapper{
					{RequestId: "r-" + batchID, Request: &allocationpb.AllocationRequest{Namespace: batchID}},
				},
			},
		},
	}
}

// runStreamBatches runs StreamBatches in the background and returns a channel with its result
func runStreamBatches(h *Handler, stream allocationpb.Processor_StreamBatchesServer) <-chan error {
	done := make(chan error, 1)
	go func() { done <- h.StreamBatches(stream) }()
	return done
}
