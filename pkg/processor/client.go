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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/uuid"

	allocationpb "agones.dev/agones/pkg/allocation/go"
)

// Config holds the processor client configuration
type Config struct {
	// ClientID is a unique identifier for this processor client instance
	ClientID string

	// ProcessorAddress specifies the address of the processor service to connect to
	ProcessorAddress string

	// MaxBatchSize determines the maximum number of allocation requests to batch together
	MaxBatchSize int

	// AllocationTimeout is the maximum duration to wait for an allocation response
	AllocationTimeout time.Duration

	// ReconnectInterval is the time to wait before retrying a failed connection
	ReconnectInterval time.Duration
}

// client implements client interface
//
//nolint:govet // fieldalignment: struct alignment is not critical for our use case
type client struct {
	// hotBatch holds the current batch of allocation requests that have been converted to protobuf format
	// It accumulates requests until a pull request is received to be directly sent to the processor
	// After sending, hotBatch is reset and starts collecting new requests for the next batch
	hotBatch *allocationpb.BatchRequest
	// pendingRequests holds the list of allocation requests currently in the batch
	// Each pendingRequest tracks the original request, its unique ID, and channels for response and error
	// This slice is used to correlate responses from the processor back to the original caller
	pendingRequests []*pendingRequest
	logger          logrus.FieldLogger
	config          Config

	batchMutex sync.RWMutex
	// requestIDMapping is a map to correlate request IDs to pendingRequest objects for response handling
	requestIDMapping map[string]*pendingRequest
}

// pendingRequest represents a request waiting for processing
type pendingRequest struct {
	// request is the original allocation request data
	request *allocationpb.AllocationRequest

	// response is the channel to receive the allocation response
	response chan *allocationpb.AllocationResponse

	// error is the channel to receive an error if processing fails
	error chan error

	// id is the unique identifier for this request
	id string
}

// Client interface for allocation operations
// Provides methods to run the processor client and perform allocation requests
type Client interface {
	// Run starts the processor client
	Run(ctx context.Context) error

	// Allocate performs a batch allocation request
	Allocate(ctx context.Context, req *allocationpb.AllocationRequest) (*allocationpb.AllocationResponse, error)
}

// NewClient creates a new processor client
func NewClient(config Config, logger logrus.FieldLogger) Client {
	if config.ClientID == "" {
		config.ClientID = string(uuid.NewUUID())
	}

	return &client{
		config: config,
		logger: logger,
		hotBatch: &allocationpb.BatchRequest{
			Requests: make([]*allocationpb.RequestWrapper, 0, config.MaxBatchSize),
		},
		pendingRequests:  make([]*pendingRequest, 0, config.MaxBatchSize),
		requestIDMapping: make(map[string]*pendingRequest),
	}
}

// Run starts the processor client and manages the connection lifecycle
// It will retry connecting to the processor service until the context is cancelled
func (p *client) Run(ctx context.Context) error {
	p.logger.Info("Starting processor client")

	// Main connection loop with retry
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Processor client stopping")
			return ctx.Err()
		default:
			if err := p.connectAndRun(ctx); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				p.logger.WithError(err).Error("Connection failed, retrying")

				// Wait before retrying connection
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(p.config.ReconnectInterval):
				}
			}
		}
	}
}

// Allocate performs an allocation request by batching it and waiting for a response or error
func (p *client) Allocate(ctx context.Context, req *allocationpb.AllocationRequest) (*allocationpb.AllocationResponse, error) {
	requestID := string(uuid.NewUUID())

	// Create a pendingRequest to track this allocation request and its response/error
	pendingReq := &pendingRequest{
		id:       requestID,
		request:  req,
		response: make(chan *allocationpb.AllocationResponse, 1),
		error:    make(chan error, 1),
	}

	// Wrap the request for batching.
	wrapper := &allocationpb.RequestWrapper{
		RequestId: requestID,
		Request:   req,
	}

	p.batchMutex.Lock()
	p.requestIDMapping[requestID] = pendingReq
	p.hotBatch.Requests = append(p.hotBatch.Requests, wrapper)
	p.pendingRequests = append(p.pendingRequests, pendingReq)
	p.batchMutex.Unlock()

	// Wait for response, error, cancellation, or timeout
	timeout := p.config.AllocationTimeout

	select {
	case response := <-pendingReq.response:
		p.logger.WithField("requestID", requestID).Debug("Received successful response")
		return response, nil

	case err := <-pendingReq.error:
		p.logger.WithField("requestID", requestID).WithError(err).Debug("Received error response")
		return nil, err

	case <-ctx.Done():
		p.batchMutex.Lock()
		delete(p.requestIDMapping, requestID)
		p.batchMutex.Unlock()
		p.logger.WithField("requestID", requestID).Debug("Request cancelled by context")
		return nil, ctx.Err()

	case <-time.After(timeout):
		p.batchMutex.Lock()
		delete(p.requestIDMapping, requestID)
		p.batchMutex.Unlock()
		p.logger.WithField("requestID", requestID).Error("Timeout waiting for processor response")
		return nil, status.Errorf(codes.DeadlineExceeded, "allocation timeout after %v", timeout)
	}
}

// handleStream processes incoming messages from the processor stream
// It listens for pull requests and batch responses, dispatching them to appropriate handlers
func (p *client) handleStream(ctx context.Context, stream allocationpb.Processor_StreamBatchesClient) error {
	p.logger.Info("Starting stream message handling")

	// Channel to handle pull requests asynchronously
	pullRequestChan := make(chan struct{}, 20)

	// Start goroutine to handle pull requests without blocking
	go p.pullRequestHandler(ctx, stream, pullRequestChan)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stream handling stopping due to context cancellation")
			return ctx.Err()
		default:
			// Receive message from processor
			msg, err := stream.Recv()
			if err != nil {
				p.logger.WithError(err).Error("Failed to receive message from processor")
				return errors.Wrap(err, "stream recv error")
			}

			// Handle message based on its payload type
			switch payload := msg.GetPayload().(type) {
			case *allocationpb.ProcessorMessage_Pull:
				// Pull request: queue for async handling
				select {
				case pullRequestChan <- struct{}{}:
					p.logger.Debug("Pull request queued successfully")
				default:
					p.logger.Warn("Pull request queue full - dropping request")
				}

			case *allocationpb.ProcessorMessage_BatchResponse:
				// Batch response: handle immediately
				p.handleBatchResponse(payload.BatchResponse)

			default:
				// Unknown message type
				p.logger.WithField("messageType", fmt.Sprintf("%T", payload)).Warn("Received unknown message type from processor")
			}
		}
	}
}

// pullRequestHandler handles pull requests asynchronously without blocking the main stream
// It waits for pull requests on the channel and processes them as they arrive
func (p *client) pullRequestHandler(ctx context.Context, stream allocationpb.Processor_StreamBatchesClient, pullRequestChan <-chan struct{}) {
	p.logger.Debug("Starting async pull request handler")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Pull request handler stopping")
			return

		case <-pullRequestChan:
			p.handlePullRequest(stream)
		}
	}
}

// handlePullRequest responds to pull requests by sending the current batch of allocation requests
// It swaps out the hot batch, resets it for new requests, and sends the ready batch to the processor
func (p *client) handlePullRequest(stream allocationpb.Processor_StreamBatchesClient) {
	// Swap out the hot batch and pending requests
	p.batchMutex.Lock()
	readyBatch := p.hotBatch
	readyRequests := p.pendingRequests

	// Reset hot batch for next requests
	p.hotBatch = &allocationpb.BatchRequest{
		Requests: make([]*allocationpb.RequestWrapper, 0, p.config.MaxBatchSize),
	}
	p.pendingRequests = make([]*pendingRequest, 0, p.config.MaxBatchSize)
	p.batchMutex.Unlock()

	if len(readyRequests) == 0 {
		p.logger.Debug("No requests to send in batch")
		return
	}

	// Send batch to processor
	p.sendBatch(stream, readyBatch, readyRequests)
}

// sendBatch sends a batch of allocation requests to the processor
func (p *client) sendBatch(stream allocationpb.Processor_StreamBatchesClient, batch *allocationpb.BatchRequest, requests []*pendingRequest) {
	batch.BatchId = string(uuid.NewUUID())

	// Prepare batch message
	batchMsg := &allocationpb.ProcessorMessage{
		ClientId: p.config.ClientID,
		Payload: &allocationpb.ProcessorMessage_BatchRequest{
			BatchRequest: batch,
		},
	}

	sendStart := time.Now()
	if err := stream.Send(batchMsg); err != nil {
		p.logger.WithError(err).Error("Failed to send batch")

		// Re-add the request to the hot batch and pendingRequests for the next pull
		p.batchMutex.Lock()
		for _, req := range requests {
			p.hotBatch.Requests = append(p.hotBatch.Requests, &allocationpb.RequestWrapper{
				RequestId: req.id,
				Request:   req.request,
			})
			p.pendingRequests = append(p.pendingRequests, req)
		}
		p.batchMutex.Unlock()
		return
	}

	sendDuration := time.Since(sendStart)
	p.logger.WithFields(logrus.Fields{
		"batchID":      batch.BatchId,
		"requestCount": len(requests),
		"sendDuration": sendDuration,
	}).Debug("Batch sent successfully")
}

// handleBatchResponse processes responses from the processor for a batch of requests
// It matches responses to pending requests, sends results/errors, and cleans up processed requests
func (p *client) handleBatchResponse(batchResp *allocationpb.BatchResponse) {
	p.logger.WithFields(logrus.Fields{
		"component":     "processor-client",
		"batchID":       batchResp.BatchId,
		"responseCount": len(batchResp.Responses),
	}).Debug("Processing batch response")

	successCount := 0
	errorCount := 0
	notFoundCount := 0

	for _, respWrapper := range batchResp.Responses {
		requestID := respWrapper.RequestId

		// Try to load the pending request for this response
		p.batchMutex.RLock()
		req, exists := p.requestIDMapping[requestID]
		p.batchMutex.RUnlock()

		if exists {
			// Track if response was processed successfully
			responseProcessed := false

			switch result := respWrapper.Result.(type) {
			case *allocationpb.ResponseWrapper_Response:
				// Success case: send response to caller
				successCount++
				responseProcessed = true

				select {
				case req.response <- result.Response:
					p.logger.WithField("requestID", requestID).Debug("Response sent successfully")
				default:
					p.logger.WithField("requestID", requestID).Warn("Failed to send response - channel full")
					responseProcessed = false
				}

			case *allocationpb.ResponseWrapper_Error:
				// Error case: send error to caller
				errorCount++
				responseProcessed = true

				code := codes.Code(result.Error.Code)
				msg := result.Error.Message

				p.logger.WithFields(logrus.Fields{
					"component": "processor-client",
					"requestID": requestID,
					"batchID":   batchResp.BatchId,
					"errorCode": code,
					"errorMsg":  msg,
				}).Error("Request failed with error from processor")

				select {
				case req.error <- status.Error(code, msg):
					p.logger.WithField("requestID", requestID).Debug("Error sent successfully")
				default:
					p.logger.WithField("requestID", requestID).Warn("Failed to send error - channel full")
					responseProcessed = false
				}

			default:
				// Missing result: treat as internal error
				errorCount++
				responseProcessed = true

				p.logger.WithFields(logrus.Fields{
					"component": "processor-client",
					"requestID": requestID,
					"batchID":   batchResp.BatchId,
				}).Error("Response wrapper has no result")

				select {
				case req.error <- status.Errorf(codes.Internal, "empty response from processor"):
					p.logger.WithField("requestID", requestID).Debug("Error sent successfully")
				default:
					p.logger.WithField("requestID", requestID).Warn("Failed to send error - channel full")
					responseProcessed = false
				}
			}

			// Only delete if response was processed successfully
			if responseProcessed {
				p.batchMutex.Lock()
				delete(p.requestIDMapping, requestID)
				p.batchMutex.Unlock()
				p.logger.WithField("requestID", requestID).Debug("Request cleaned up successfully")
			} else {
				p.logger.WithField("requestID", requestID).Warn("Keeping request in map due to failed processing")
			}
		} else {
			// No pending request found for this response
			notFoundCount++
			p.logger.WithFields(logrus.Fields{
				"component": "processor-client",
				"requestID": requestID,
				"batchID":   batchResp.BatchId,
			}).Warn("No pending request found for response - may have timed out")
		}
	}

	// Log summary of batch response processing
	p.logger.WithFields(logrus.Fields{
		"component":     "processor-client",
		"batchID":       batchResp.BatchId,
		"successCount":  successCount,
		"errorCount":    errorCount,
		"notFoundCount": notFoundCount,
		"totalCount":    len(batchResp.Responses),
	}).Debug("Batch response processing completed")
}

// connectAndRun handles the full connection lifecycle to the processor service
// It establishes a connection, creates a stream, registers the client, and then
// delegates to handleStream to process messages until an error or cancellation
func (p *client) connectAndRun(ctx context.Context) error {
	// Connect to the processor
	conn, err := p.connect(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to connect")
	}
	defer func() { _ = conn.Close() }()

	// Create a new processor client from the connection
	client := allocationpb.NewProcessorClient(conn)

	// Open a streaming RPC to the processor
	stream, err := client.StreamBatches(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create stream")
	}

	// Register this client instance with the processor
	if err := p.registerClient(stream); err != nil {
		return errors.Wrap(err, "failed to register")
	}

	p.logger.Info("Connected to processor")

	// Handle the stream until an error occurs or the context is cancelled
	return p.handleStream(ctx, stream)
}

// connect attempts to connect to the processor service with health checks
// Returns a healthy gRPC connection or an error
func (p *client) connect(ctx context.Context) (*grpc.ClientConn, error) {
	p.logger.Info("Attempting connection")

	conn, err := grpc.NewClient(p.config.ProcessorAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		p.logger.WithError(err).Error("connection failed")
		return nil, err
	}

	// Perform a health check on the connection
	if err := p.healthCheck(ctx, conn); err != nil {
		p.logger.WithError(err).Error("health check failed")
		_ = conn.Close()
		return nil, err
	}

	p.logger.Info("Successfully connected to processor")
	return conn, nil
}

// healthCheck verifies that the processor service is healthy and serving requests
// Returns an error if the health check fails or the service is not in SERVING state
func (p *client) healthCheck(ctx context.Context, conn *grpc.ClientConn) error {
	healthClient := grpc_health_v1.NewHealthClient(conn)

	// Set a timeout for the health check RPC
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := healthClient.Check(healthCtx, &grpc_health_v1.HealthCheckRequest{
		Service: "processor",
	})
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return errors.Errorf("processor not serving: %v", resp.Status)
	}

	return nil
}

// registerClient sends a registration message to the processor over the stream
// This identifies the client instance to the processor service
func (p *client) registerClient(stream allocationpb.Processor_StreamBatchesClient) error {
	p.logger.WithField("clientID", p.config.ClientID).Info("Registering client with processor")

	registerMsg := &allocationpb.ProcessorMessage{
		ClientId: p.config.ClientID,
	}

	// Send the registration message
	err := stream.Send(registerMsg)
	if err != nil {
		p.logger.WithField("clientID", p.config.ClientID).WithError(err).Error("Failed to register client")
		return err
	}

	return nil
}
