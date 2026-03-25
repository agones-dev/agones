// Copyright 2026 Google LLC All Rights Reserved.
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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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
)

// newTestHandler creates a processorHandler with a mock allocator
func newTestHandler(ctx context.Context, allocFunc func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error)) *processorHandler {
	handlerCtx, cancel := context.WithCancel(ctx)
	return &processorHandler{
		allocator:                 &mockAllocator{allocateFunc: allocFunc},
		ctx:                       handlerCtx,
		cancel:                    cancel,
		clients:                   make(map[string]allocationpb.Processor_StreamBatchesServer),
		grpcUnallocatedStatusCode: codes.ResourceExhausted,
		pullInterval:              100 * time.Millisecond,
	}
}

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

			h := &processorHandler{}
			h.clients = make(map[string]allocationpb.Processor_StreamBatchesServer)
			stream := newMockServerStream(context.Background())

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

			h := &processorHandler{
				clients: make(map[string]allocationpb.Processor_StreamBatchesServer),
			}
			stream := newMockServerStream(context.Background())
			for _, id := range tc.initialIDs {
				h.addClient(id, stream)
			}

			h.removeClient(tc.removeID)

			h.mu.RLock()
			defer h.mu.RUnlock()
			assert.Len(t, h.clients, tc.wantLen)
		})
	}
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
				assert.Equal(t, int32(tc.wantErrorCode), result.error.Code)
				assert.Contains(t, result.error.Message, tc.wantErrorMsg)
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

			assert.Equal(t, tc.wantBatchID, resp.BatchId)
			require.Len(t, resp.Responses, len(tc.wantErrors))

			for i, wrapper := range resp.Responses {
				assert.Equal(t, fmt.Sprintf("req-%d", i), wrapper.RequestId)
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
				assert.Equal(t, "b-1", batchResp.BatchId)
				require.Len(t, batchResp.Responses, 1)
				assert.Equal(t, "r-1", batchResp.Responses[0].RequestId)
				assert.NotNil(t, batchResp.Responses[0].GetResponse())
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

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

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

			err := h.StreamBatches(stream)

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
		wantPulls   bool
	}{
		{
			name:        "sends pull to connected clients",
			clientCount: 2,
			wantPulls:   true,
		},
		{
			name:        "no clients means no pulls sent",
			clientCount: 0,
			wantPulls:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			h := newTestHandler(ctx, nil)

			streams := make([]*mockServerStream, tc.clientCount)
			for i := 0; i < tc.clientCount; i++ {
				streams[i] = newMockServerStream(ctx)
				h.addClient(fmt.Sprintf("client-%d", i), streams[i])
			}

			h.StartPullRequestTicker()

			if tc.wantPulls {
				for i, stream := range streams {
					assert.Eventually(t, func() bool {
						select {
						case msg := <-stream.sendChan:
							return msg.GetPull() != nil
						default:
							return false
						}
					}, 5*time.Second, 10*time.Millisecond, "client-%d should receive a pull", i)
				}
			} else {
				assert.Never(t, func() bool {
					h.mu.RLock()
					defer h.mu.RUnlock()
					return len(h.clients) > 0
				}, 200*time.Millisecond, 20*time.Millisecond, "no clients should appear")
			}
		})
	}
}

func TestStartPullRequestTickerRemovesFailingClient(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h := newTestHandler(ctx, nil)
	h.pullInterval = 20 * time.Millisecond

	failStream := &failingSendStream{
		mockServerStream: *newMockServerStream(ctx),
	}
	h.addClient("failing-client", failStream)

	h.StartPullRequestTicker()

	assert.Eventually(t, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return len(h.clients) == 0
	}, 5*time.Second, 20*time.Millisecond, "failing client should be removed")
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
func (m *mockServerStream) SendMsg(interface{}) error    { return nil }
func (m *mockServerStream) RecvMsg(interface{}) error    { return nil }

type failingSendStream struct {
	mockServerStream
}

func (f *failingSendStream) Send(_ *allocationpb.ProcessorMessage) error {
	return errors.New("send failed")
}
