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

// batchProcessingTimeout is the timeout used when processing the allocations of a batch
const batchProcessingTimeout = 30 * time.Second

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
//
//nolint:govet // fieldalignment: mu is kept above the clients it guards
type Handler struct {
	allocationpb.UnimplementedProcessorServer
	allocator                 GameServerAllocator
	mu                        sync.RWMutex
	clients                   map[string]*clientStream
	clock                     clock.WithTicker
	batchTimeout              time.Duration
	pullInterval              time.Duration
	grpcUnallocatedStatusCode codes.Code
}

// NewServiceHandler creates a new ProcessorHandler with the given allocator.
func NewServiceHandler(allocator GameServerAllocator, pullInterval time.Duration, grpcUnallocatedStatusCode codes.Code) *Handler {
	return &Handler{
		allocator:                 allocator,
		clients:                   make(map[string]*clientStream),
		grpcUnallocatedStatusCode: grpcUnallocatedStatusCode,
		batchTimeout:              batchProcessingTimeout,
		pullInterval:              pullInterval,
		clock:                     clock.RealClock{},
	}
}

// StreamBatches handles a bidirectional stream for batch allocation requests from a client.
// Registers the client, processes incoming batches, and sends responses.
func (h *Handler) StreamBatches(stream allocationpb.Processor_StreamBatchesServer) error {
	// Receive the client ID from the first message of the stream
	clientID, err := receiveClientID(stream)
	if err != nil {
		return err
	}

	// Create a new client stream for this client ID and stream
	client := newClientStream(clientID, stream)

	// Register the client and ensure it is removed when the handler exits
	h.addClient(clientID, client)
	defer h.removeClient(clientID, client)
	client.logger.Debug("Client registered")

	// Start serving the client stream, processing incoming batches until the stream ends
	return h.serve(client)
}

// receiveClientID extracts the client ID from the first message of the stream and validates it
func receiveClientID(stream allocationpb.Processor_StreamBatchesServer) (string, error) {
	msg, err := stream.Recv()
	if err != nil {
		handlerLogger.WithError(err).Debug("Stream receive error on connect")
		return "", err
	}

	clientID := msg.GetClientId()
	if clientID == "" {
		handlerLogger.Warn("Received empty clientID, closing stream")
		return "", status.Error(codes.InvalidArgument, "clientID is required")
	}

	return clientID, nil
}

// serve processes incoming batches from a client stream until the stream ends
func (h *Handler) serve(client *clientStream) error {
	// The sends WaitGroup tracks all goroutines that send responses to the client
	// ensuring they complete before the handler returns
	var sends sync.WaitGroup
	defer sends.Wait()

	// The pull sender is started with its own context, which can be
	// cancelled independently of the main handler
	pullCtx, stopPulls := context.WithCancel(client.stream.Context())
	defer stopPulls()
	sends.Go(func() { client.sendPulls(pullCtx) })

	for {
		// Receive the next batch from the client stream
		batch, err := client.receiveBatch()
		if err != nil {
			return err
		}

		// Process the batch asynchronously to avoid blocking the receipt of subsequent batches
		sends.Go(func() { h.processBatch(client, batch) })
	}
}

// processBatch handles a single batch of allocation requests from a client, submitting it for
// processing and sending the response back to the client
func (h *Handler) processBatch(client *clientStream, batch *allocationpb.BatchRequest) {
	// Create a context with a timeout for processing this batch to ensure
	// it does not exceed the allowed time
	ctx, cancel := context.WithTimeout(client.stream.Context(), h.batchTimeout)
	defer cancel()

	// Submit the batch for processing and obtain the response
	response := h.submitBatch(ctx, batch.GetBatchId(), batch.GetRequests())

	// Send the response back to the client
	if err := client.respond(response); err != nil {
		client.logger.WithFields(logrus.Fields{
			logFieldBatchID:      batch.GetBatchId(),
			logFieldRequestCount: len(batch.GetRequests()),
		}).WithError(err).Error("Failed to send response")
	}
}

// StartPullRequestTicker starts a goroutine that periodically sends pull requests to
// all connected clients
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

// sendPullRequestsToClients iterates over all connected clients and requests their
// next batch of allocation requests
func (h *Handler) sendPullRequestsToClients() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		client.requestPull()
	}
}

// processAllocationsConcurrently processes multiple allocation requests in parallel
func (h *Handler) processAllocationsConcurrently(ctx context.Context, requestWrappers []*allocationpb.RequestWrapper) []allocationResult {
	var wg sync.WaitGroup
	results := make([]allocationResult, len(requestWrappers))

	for i, reqWrapper := range requestWrappers {
		wg.Go(func() {
			results[i] = h.processAllocation(ctx, reqWrapper.GetRequest())
		})
	}

	wg.Wait()

	return results
}

// processAllocation handles a single allocation request by using the allocator and returns the result
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
			error: &rpcstatus.Status{Code: int32(code), Message: msg}, //nolint:gosec // G115: gRPC codes are 0-16.
		}
	}

	resultObj, err := h.allocator.Allocate(ctx, gsa)
	if err != nil {
		return makeError(err, codes.Internal)
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(GRPCCodeFromHTTPStatus(int(s.Code))), //nolint:gosec // G115: gRPC codes are 0-16.
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
			RequestId: requestWrappers[i].GetRequestId(),
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

func (h *Handler) addClient(clientID string, client *clientStream) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[clientID] = client
}

func (h *Handler) removeClient(clientID string, client *clientStream) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[clientID] == client {
		delete(h.clients, clientID)
	}
}
