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
	"testing"
	"time"

	allocationpb "agones.dev/agones/pkg/allocation/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
)

// Mock for Processor_StreamBatchesClient
type mockStream struct {
	recvChan chan *allocationpb.ProcessorMessage
	sendChan chan *allocationpb.ProcessorMessage
}

func (m *mockStream) Send(msg *allocationpb.ProcessorMessage) error {
	m.sendChan <- msg
	return nil
}
func (m *mockStream) Recv() (*allocationpb.ProcessorMessage, error) {
	msg, ok := <-m.recvChan
	if !ok {
		return nil, errors.New("stream closed")
	}
	return msg, nil
}
func (m *mockStream) CloseSend() error             { close(m.sendChan); return nil }
func (m *mockStream) Context() context.Context     { return context.Background() }
func (m *mockStream) Header() (metadata.MD, error) { return metadata.MD{}, nil }
func (m *mockStream) Trailer() metadata.MD         { return metadata.MD{} }
func (m *mockStream) SendMsg(interface{}) error    { return nil }
func (m *mockStream) RecvMsg(interface{}) error    { return nil }

func TestProcessorClient_Allocate(t *testing.T) {
	testCases := []struct {
		name          string
		batchSize     int
		setupResponse func(stream *mockStream, reqIDs []string)
		expectError   []bool
	}{
		{
			name:      "successful allocation with batchSize 1",
			batchSize: 1,
			setupResponse: func(stream *mockStream, reqIDs []string) {
				msg := <-stream.sendChan
				batchID := msg.GetBatchRequest().BatchId
				stream.recvChan <- &allocationpb.ProcessorMessage{
					Payload: &allocationpb.ProcessorMessage_BatchResponse{
						BatchResponse: &allocationpb.BatchResponse{
							BatchId: batchID,
							Responses: []*allocationpb.ResponseWrapper{
								{
									RequestId: reqIDs[0],
									Result: &allocationpb.ResponseWrapper_Response{
										Response: &allocationpb.AllocationResponse{},
									},
								},
							},
						},
					},
				}
			},
			expectError: []bool{false},
		},
		{
			name:      "successful allocation with batchSize 3",
			batchSize: 3,
			setupResponse: func(stream *mockStream, reqIDs []string) {
				msg := <-stream.sendChan
				batchID := msg.GetBatchRequest().BatchId
				responses := make([]*allocationpb.ResponseWrapper, 3)
				for i := 0; i < 3; i++ {
					responses[i] = &allocationpb.ResponseWrapper{
						RequestId: reqIDs[i],
						Result: &allocationpb.ResponseWrapper_Response{
							Response: &allocationpb.AllocationResponse{},
						},
					}
				}
				stream.recvChan <- &allocationpb.ProcessorMessage{
					Payload: &allocationpb.ProcessorMessage_BatchResponse{
						BatchResponse: &allocationpb.BatchResponse{
							BatchId:   batchID,
							Responses: responses,
						},
					},
				}
			},
			expectError: []bool{false, false, false},
		},
		{
			name:          "pull received but no batch available",
			batchSize:     0,
			setupResponse: func(_ *mockStream, _ []string) {},
			expectError:   []bool{},
		},
		{
			name:      "allocation error response",
			batchSize: 1,
			setupResponse: func(stream *mockStream, reqIDs []string) {
				msg := <-stream.sendChan
				batchID := msg.GetBatchRequest().BatchId
				stream.recvChan <- &allocationpb.ProcessorMessage{
					Payload: &allocationpb.ProcessorMessage_BatchResponse{
						BatchResponse: &allocationpb.BatchResponse{
							BatchId: batchID,
							Responses: []*allocationpb.ResponseWrapper{
								{
									RequestId: reqIDs[0],
									Result: &allocationpb.ResponseWrapper_Error{
										Error: &status.Status{
											Code:    int32(13), // INTERNAL
											Message: "mock error",
										},
									},
								},
							},
						},
					},
				}
			},
			expectError: []bool{true},
		},
		{
			name:      "allocation timeout",
			batchSize: 1,
			setupResponse: func(_ *mockStream, _ []string) {
				// Do not send any batch response, let it timeout
				time.Sleep(300 * time.Millisecond)
			},
			expectError: []bool{true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := logrus.New()
			config := Config{
				MaxBatchSize:      10,
				AllocationTimeout: 200 * time.Millisecond,
				ClientID:          "test-client",
			}
			stream := &mockStream{
				recvChan: make(chan *allocationpb.ProcessorMessage, 10),
				sendChan: make(chan *allocationpb.ProcessorMessage, 10),
			}
			p := &client{
				config:           config,
				logger:           logger,
				hotBatch:         &allocationpb.BatchRequest{Requests: make([]*allocationpb.RequestWrapper, 0, config.MaxBatchSize)},
				pendingRequests:  make([]*pendingRequest, 0, config.MaxBatchSize),
				requestIDMapping: make(map[string]*pendingRequest),
			}

			// Start handleStream in a goroutine
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				_ = p.handleStream(ctx, stream)
			}()

			reqIDs := make([]string, tc.batchSize)
			responses := make([]*allocationpb.AllocationResponse, tc.batchSize)
			errorsArr := make([]error, tc.batchSize)
			doneChans := make([]chan struct{}, tc.batchSize)

			for i := 0; i < tc.batchSize; i++ {
				doneChans[i] = make(chan struct{})
				req := &allocationpb.AllocationRequest{}
				go func(idx int) {
					responses[idx], errorsArr[idx] = p.Allocate(context.Background(), req)
					close(doneChans[idx])
				}(i)
			}

			// Wait until hotBatch has the expected number of requests
			// to ensure all the allocate calls have been processed and batched
			assert.Eventually(t, func() bool {
				p.batchMutex.RLock()
				defer p.batchMutex.RUnlock()
				return len(p.hotBatch.Requests) == tc.batchSize
			}, 500*time.Millisecond, 50*time.Millisecond)

			// Extract request IDs after the batch is ready
			p.batchMutex.RLock()
			for i := 0; i < tc.batchSize; i++ {
				reqIDs[i] = p.hotBatch.Requests[i].RequestId
			}
			p.batchMutex.RUnlock()

			go tc.setupResponse(stream, reqIDs)

			// Simulate a pullRequest
			stream.recvChan <- &allocationpb.ProcessorMessage{Payload: &allocationpb.ProcessorMessage_Pull{}}

			for i := 0; i < tc.batchSize; i++ {
				<-doneChans[i]
				if tc.expectError[i] && errorsArr[i] == nil {
					t.Errorf("expected error for request %d, got nil", i)
				}
				if !tc.expectError[i] && errorsArr[i] != nil {
					t.Errorf("expected no error for request %d, got %v", i, errorsArr[i])
				}
				if !tc.expectError[i] && responses[i] == nil {
					t.Errorf("expected response for request %d, got nil", i)
				}
			}
		})
	}
}
