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

// Processor
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agones.dev/agones/pkg/allocation/converters"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"

	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// allocationResult represents the result of an allocation attempt
type allocationResult struct {
	response *allocationpb.AllocationResponse
	error    *rpcstatus.Status
}

// processorHandler represents the gRPC server for processing allocation requests
type processorHandler struct {
	allocationpb.UnimplementedProcessorServer
	ctx                       context.Context
	cancel                    context.CancelFunc
	mu                        sync.RWMutex
	allocator                 *gameserverallocations.Allocator
	clients                   map[string]allocationpb.Processor_StreamBatchesServer
	grpcUnallocatedStatusCode codes.Code
	pullInterval              time.Duration
}

// newServiceHandler creates a new instance of processorHandler
func newServiceHandler(ctx context.Context, kubeClient kubernetes.Interface, agonesClient versioned.Interface,
	health healthcheck.Handler, config processorConfig, grpcUnallocatedStatusCode codes.Code) *processorHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		agonesClient.AgonesV1(),
		kubeClient,
		gameserverallocations.NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), gsCounter, health),
		config.RemoteAllocationTimeout,
		config.TotalRemoteAllocationTimeout,
		config.AllocationBatchWaitTime)

	batchCtx, cancel := context.WithCancel(ctx)
	h := processorHandler{
		allocator:                 allocator,
		ctx:                       batchCtx,
		cancel:                    cancel,
		grpcUnallocatedStatusCode: grpcUnallocatedStatusCode,
		pullInterval:              config.PullInterval,
	}

	kubeInformerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())

	if err := allocator.Run(ctx); err != nil {
		logger.WithError(err).Fatal("Starting allocator failed.")
	}

	return &h
}

// StreamBatches handles a bidirectional stream for batch allocation requests from a client
// Registers the client, processes incoming batches, and sends responses
func (h *processorHandler) StreamBatches(stream allocationpb.Processor_StreamBatchesServer) error {
	var clientID string

	// Wait for first message to get clientID
	msg, err := stream.Recv()
	if err != nil {
		logger.WithError(err).Debug("Stream receive error on connect")
		return err
	}

	clientID = msg.GetClientId()
	if clientID == "" {
		logger.Warn("Received empty clientID, closing stream")
		return nil
	}

	h.addClient(clientID, stream)
	defer h.removeClient(clientID)
	logger.WithField("clientID", clientID).Debug("Client registered")

	// Main loop: handle incoming messages
	for {
		msg, err := stream.Recv()
		if err != nil {
			logger.WithError(err).Debug("Stream receive error")
			return err
		}

		payload := msg.GetPayload()
		if payload == nil {
			logger.WithField("clientID", clientID).Warn("Received message with nil payload")
			continue
		}

		batchPayload, ok := payload.(*allocationpb.ProcessorMessage_BatchRequest)
		if !ok {
			logger.WithField("clientID", clientID).Warn("Received non-batch request payload")
			continue
		}

		batchRequest := batchPayload.BatchRequest
		batchID := batchRequest.GetBatchId()
		requestWrappers := batchRequest.GetRequests()

		logger.WithFields(logrus.Fields{
			"clientID":     clientID,
			"batchID":      batchID,
			"requestCount": len(requestWrappers),
		}).Debug("Received batch request")

		// Extract request IDs for logging
		requestIDs := make([]string, len(requestWrappers))
		for i, wrapper := range requestWrappers {
			requestIDs[i] = wrapper.GetRequestId()
		}

		// Submit batch for processing
		response := h.submitBatch(batchID, requestWrappers)

		respMsg := &allocationpb.ProcessorMessage{
			ClientId: clientID,
			Payload: &allocationpb.ProcessorMessage_BatchResponse{
				BatchResponse: response,
			},
		}

		// TODO: we might want to retry on failure here ?
		if err := stream.Send(respMsg); err != nil {
			logger.WithFields(logrus.Fields{
				"clientID":     clientID,
				"batchID":      batchID,
				"requestCount": len(requestWrappers),
			}).WithError(err).Error("Failed to send response")
			continue
		}
	}
}

// StartPullRequestTicker periodically sends pull requests to all connected clients
func (h *processorHandler) StartPullRequestTicker() {
	go func() {
		ticker := time.NewTicker(h.pullInterval)
		defer ticker.Stop()

		for {
			select {
			case <-h.ctx.Done():
				return
			case <-ticker.C:
				h.mu.RLock()
				for clientID, stream := range h.clients {
					pullMsg := &allocationpb.ProcessorMessage{
						ClientId: clientID,
						Payload: &allocationpb.ProcessorMessage_Pull{
							Pull: &allocationpb.PullRequest{Message: "pull"},
						},
					}
					go func(id string, s allocationpb.Processor_StreamBatchesServer) {
						if err := s.Send(pullMsg); err != nil {
							logger.WithFields(logrus.Fields{
								"clientID": id,
								"error":    err,
							}).Warn("Failed to send pull request, removing client")
							h.removeClient(id)
						}
					}(clientID, stream)
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// processAllocationsConcurrently processes multiple allocation requests in parallel
func (h *processorHandler) processAllocationsConcurrently(requestWrappers []*allocationpb.RequestWrapper) []allocationResult {
	var wg sync.WaitGroup
	results := make([]allocationResult, len(requestWrappers))

	for i, reqWrapper := range requestWrappers {
		wg.Add(1)
		go func(index int, requestWrapper *allocationpb.RequestWrapper) {
			defer wg.Done()
			results[index] = h.processAllocation(requestWrapper.Request)
		}(i, reqWrapper)
	}

	wg.Wait()

	return results
}

// processAllocation handles a single allocation request by using the allocator
func (h *processorHandler) processAllocation(req *allocationpb.AllocationRequest) allocationResult {
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

	resultObj, err := h.allocator.Allocate(h.ctx, gsa)
	if err != nil {
		return makeError(err, h.grpcUnallocatedStatusCode)
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return allocationResult{
			error: &rpcstatus.Status{
				Code:    int32(grpcCodeFromHTTPStatus(int(s.Code))),
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
		return makeError(err, h.grpcUnallocatedStatusCode)
	}

	return allocationResult{response: response}
}

// submitBatch accepts a batch of allocation requests, processes them, and assembles a batch response
func (h *processorHandler) submitBatch(batchID string, requestWrappers []*allocationpb.RequestWrapper) *allocationpb.BatchResponse {
	results := h.processAllocationsConcurrently(requestWrappers)
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

// getGRPCServerOptions returns a list of GRPC server options to use when only serving gRPC requests.
func (h *processorHandler) getGRPCServerOptions() []grpc.ServerOption {
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

// addClient registers a new client for streaming allocation responses
func (h *processorHandler) addClient(clientID string, stream allocationpb.Processor_StreamBatchesServer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients == nil {
		h.clients = make(map[string]allocationpb.Processor_StreamBatchesServer)
	}

	h.clients[clientID] = stream
}

// removeClient unregisters a client from streaming allocation responses
func (h *processorHandler) removeClient(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, clientID)
}
