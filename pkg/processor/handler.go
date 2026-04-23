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
	"fmt"
	"io"
	"sync"
	"time"

	"agones.dev/agones/pkg/allocation/converters"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"
)

var handlerLogger = runtime.NewLoggerWithSource("processorHandler")

// GameServerAllocator represents the interface to allocate game servers.
type GameServerAllocator interface {
	Allocate(ctx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error)
}

// allocationResult represents the result of an allocation attempt.
type allocationResult struct {
	response *allocationpb.AllocationResponse
	error    *rpcstatus.Status
}

// Handler is the gRPC server for processing allocation requests.
type Handler struct {
	allocationpb.UnimplementedProcessorServer
	allocator                 GameServerAllocator
	clients                   map[string]allocationpb.Processor_StreamBatchesServer
	clock                     clock.WithTicker
	mu                        sync.RWMutex
	pullInterval              time.Duration
	grpcUnallocatedStatusCode codes.Code
}

// NewServiceHandler creates a new ProcessorHandler with the given allocator.
func NewServiceHandler(allocator GameServerAllocator, pullInterval time.Duration, grpcUnallocatedStatusCode codes.Code) *Handler {
	return &Handler{
		allocator:                 allocator,
		clients:                   make(map[string]allocationpb.Processor_StreamBatchesServer),
		grpcUnallocatedStatusCode: grpcUnallocatedStatusCode,
		pullInterval:              pullInterval,
		clock:                     clock.RealClock{},
	}
}

// StreamBatches handles a bidirectional stream for batch allocation requests from a client.
// Registers the client, processes incoming batches, and sends responses.
func (h *Handler) StreamBatches(stream allocationpb.Processor_StreamBatchesServer) error {
	var clientID string

	// Wait for first message to get clientID
	msg, err := stream.Recv()
	if err != nil {
		handlerLogger.WithError(err).Debug("Stream receive error on connect")
		return err
	}

	clientID = msg.GetClientId()
	if clientID == "" {
		handlerLogger.Warn("Received empty clientID, closing stream")
		return status.Error(codes.InvalidArgument, "clientID is required")
	}

	h.addClient(clientID, stream)
	defer h.removeClient(clientID)
	handlerLogger.WithField("clientID", clientID).Debug("Client registered")

	// Main loop: handle incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				handlerLogger.WithField("clientID", clientID).Debug("Stream closed by client")
			} else {
				handlerLogger.WithField("clientID", clientID).WithError(err).Warn("Stream receive error")
			}
			return err
		}

		payload := msg.GetPayload()
		if payload == nil {
			handlerLogger.WithField("clientID", clientID).Warn("Received message with nil payload")
			continue
		}

		batchPayload, ok := payload.(*allocationpb.ProcessorMessage_BatchRequest)
		if !ok {
			handlerLogger.WithField("clientID", clientID).Warn("Received non-batch request payload")
			continue
		}

		batchRequest := batchPayload.BatchRequest
		batchID := batchRequest.GetBatchId()
		requestWrappers := batchRequest.GetRequests()

		handlerLogger.WithFields(logrus.Fields{
			"clientID":     clientID,
			"batchID":      batchID,
			"requestCount": len(requestWrappers),
		}).Debug("Received batch request")

		// Submit batch for processing
		response := h.submitBatch(stream.Context(), batchID, requestWrappers)

		respMsg := &allocationpb.ProcessorMessage{
			ClientId: clientID,
			Payload: &allocationpb.ProcessorMessage_BatchResponse{
				BatchResponse: response,
			},
		}

		if err := stream.Send(respMsg); err != nil {
			handlerLogger.WithFields(logrus.Fields{
				"clientID":     clientID,
				"batchID":      batchID,
				"requestCount": len(requestWrappers),
			}).WithError(err).Error("Failed to send response")
			continue
		}
	}
}

// StartPullRequestTicker periodically sends pull requests to all connected clients.
func (h *Handler) StartPullRequestTicker(ctx context.Context) {
	go func() {
		ticker := h.clock.NewTicker(h.pullInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				h.sendPullRequestsToClients()
			}
		}
	}()
}

// sendPullRequestsToClients sends a pull request to every connected client,
// removing any client whose stream has become unhealthy.
func (h *Handler) sendPullRequestsToClients() {
	h.mu.RLock()
	snapshot := make(map[string]allocationpb.Processor_StreamBatchesServer, len(h.clients))
	for id, s := range h.clients {
		snapshot[id] = s
	}
	h.mu.RUnlock()

	for clientID, stream := range snapshot {
		pullMsg := &allocationpb.ProcessorMessage{
			ClientId: clientID,
			Payload: &allocationpb.ProcessorMessage_Pull{
				Pull: &allocationpb.PullRequest{Message: "pull"},
			},
		}
		if err := stream.Send(pullMsg); err != nil {
			handlerLogger.WithFields(logrus.Fields{
				"clientID": clientID,
				"error":    err,
			}).Warn("Failed to send pull request, removing client")
			h.removeClient(clientID)
		}
	}
}

// processAllocationsConcurrently processes multiple allocation requests in parallel.
func (h *Handler) processAllocationsConcurrently(ctx context.Context, requestWrappers []*allocationpb.RequestWrapper) []allocationResult {
	var wg sync.WaitGroup
	results := make([]allocationResult, len(requestWrappers))

	for i, reqWrapper := range requestWrappers {
		wg.Go(func() {
			results[i] = h.processAllocation(ctx, reqWrapper.Request)
		})
	}

	wg.Wait()

	return results
}

// processAllocation handles a single allocation request by using the allocator.
func (h *Handler) processAllocation(ctx context.Context, req *allocationpb.AllocationRequest) allocationResult {
	gsa := converters.ConvertAllocationRequestToGSA(req)
	gsa.ApplyDefaults()

	makeError := func(err error, fallbackCode codes.Code) allocationResult {
		code := fallbackCode
		msg := err.Error()
		if grpcStatus, ok := status.FromError(err); ok {
			code = grpcStatus.Code()
			msg = grpcStatus.Message()
		}
		return allocationResult{
			error: &rpcstatus.Status{Code: int32(code), Message: msg},
		}
	}

	resultObj, err := h.allocator.Allocate(ctx, gsa)
	if err != nil {
		return makeError(err, codes.Internal)
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(GRPCCodeFromHTTPStatus(int(s.Code))),
				Message: s.Message,
			},
		}
	}

	allocatedGsa, ok := resultObj.(*allocationv1.GameServerAllocation)
	if !ok {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(codes.Internal),
				Message: fmt.Sprintf("internal server error - Bad GSA format %v", resultObj),
			},
		}
	}

	response, err := converters.ConvertGSAToAllocationResponse(allocatedGsa, h.grpcUnallocatedStatusCode)
	if err != nil {
		return makeError(err, codes.Internal)
	}

	return allocationResult{response: response}
}

// submitBatch accepts a batch of allocation requests, processes them, and assembles a batch response.
func (h *Handler) submitBatch(ctx context.Context, batchID string, requestWrappers []*allocationpb.RequestWrapper) *allocationpb.BatchResponse {
	results := h.processAllocationsConcurrently(ctx, requestWrappers)
	responseWrappers := make([]*allocationpb.ResponseWrapper, len(requestWrappers))

	for i, result := range results {
		wrapper := &allocationpb.ResponseWrapper{
			RequestId: requestWrappers[i].RequestId,
		}

		if result.error != nil {
			wrapper.Result = &allocationpb.ResponseWrapper_Error{
				Error: result.error,
			}
		} else {
			wrapper.Result = &allocationpb.ResponseWrapper_Response{
				Response: result.response,
			}
		}
		responseWrappers[i] = wrapper
	}

	return &allocationpb.BatchResponse{
		BatchId:   batchID,
		Responses: responseWrappers,
	}
}

// GetGRPCServerOptions returns a list of gRPC server options.
func (h *Handler) GetGRPCServerOptions() []grpc.ServerOption {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),

		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 1 * time.Minute,
			Timeout:           30 * time.Second,
			Time:              30 * time.Second,
		}),
	}

	return opts
}

// addClient registers a new client for streaming allocation responses.
func (h *Handler) addClient(clientID string, stream allocationpb.Processor_StreamBatchesServer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[clientID] = stream
}

// removeClient unregisters a client from streaming allocation responses.
func (h *Handler) removeClient(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, clientID)
}
