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
	"io"
	"sync"

	allocationpb "agones.dev/agones/pkg/allocation/go"

	"github.com/sirupsen/logrus"
)

// clientStream represents the stream of a registered client: it handles receiving batch requests and sending pull requests and batch responses
// It ensures that only one send happens at a time, as required by gRPC
//
//nolint:govet // fieldalignment: mu is kept above the stream it guards
type clientStream struct {
	// id of the client
	id     string
	logger *logrus.Entry
	// pulls represent the pending pull requests to be sent to the client
	pulls chan struct{}
	// mu makes the sends take turns
	mu sync.Mutex
	// stream is the gRPC stream to the client
	stream allocationpb.Processor_StreamBatchesServer
}

// newClientStream creates a new clientStream for the given client ID and gRPC stream
func newClientStream(id string, stream allocationpb.Processor_StreamBatchesServer) *clientStream {
	return &clientStream{
		id:     id,
		logger: handlerLogger.WithField(logFieldClientID, id),
		pulls:  make(chan struct{}, 1),
		stream: stream,
	}
}

// receiveBatch receives the next batch request from the client, skipping other message types
// It returns the batch request or an error if the stream ends or encounters an error
func (c *clientStream) receiveBatch() (*allocationpb.BatchRequest, error) {
	for {
		// Receive the next message from the client stream
		msg, err := c.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.logger.Debug("Stream closed by client")
			} else {
				c.logger.WithError(err).Warn("Stream receive error")
			}
			return nil, err
		}

		// Check if the message contains a batch request payload
		payload := msg.GetPayload()
		if payload == nil {
			c.logger.Warn("Received message with nil payload")
			continue
		}

		// Assert that the payload is of type BatchRequest
		batchPayload, ok := payload.(*allocationpb.ProcessorMessage_BatchRequest)
		if !ok {
			c.logger.Warn("Received non-batch request payload")
			continue
		}

		// Extract the batch request from the payload
		batch := batchPayload.BatchRequest
		c.logger.WithFields(logrus.Fields{
			logFieldBatchID:      batch.GetBatchId(),
			logFieldRequestCount: len(batch.GetRequests()),
		}).Debug("Received batch request")

		return batch, nil
	}
}

// requestPull signals that a pull request should be sent to the client
// It does not block if a pull request is already pending
func (c *clientStream) requestPull() {
	select {
	case c.pulls <- struct{}{}:
	default:
	}
}

// sendPulls sends the pull requests asked for with requestPull, until ctx is done or one cannot be sent
func (c *clientStream) sendPulls(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.pulls:
			err := c.send(&allocationpb.ProcessorMessage{
				ClientId: c.id,
				Payload: &allocationpb.ProcessorMessage_Pull{
					Pull: &allocationpb.PullRequest{Message: "pull"},
				},
			})
			if err != nil {
				// gRPC aborts a stream on which a send failed, which ends its handler
				c.logger.WithError(err).Warn("Failed to send pull request")
				return
			}
		}
	}
}

// respond sends the batch response to the client
func (c *clientStream) respond(response *allocationpb.BatchResponse) error {
	return c.send(&allocationpb.ProcessorMessage{
		ClientId: c.id,
		Payload: &allocationpb.ProcessorMessage_BatchResponse{
			BatchResponse: response,
		},
	})
}

// send sends a processor message to the client, ensuring that only one send occurs at a time
func (c *clientStream) send(msg *allocationpb.ProcessorMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.stream.Send(msg)
}
