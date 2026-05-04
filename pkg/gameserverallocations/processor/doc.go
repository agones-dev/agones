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

// Package processor provides client functionality for the Agones allocation processor system
//
// The allocation processor system enables batched game server allocation requests
// This package contains the client implementation that connects to the processor service
// batches allocation requests, and handles the stream-based communication protocol
//
// Key components:
// - Client: Manages connection lifecycle and request batching
// - Config: Configuration for processor client behavior
// - Batch handling: Accumulates requests and sends them in batches to the processor
//
// The client establishes a bidirectional gRPC stream with the processor service,
// registers itself, and then handles pull requests (to send batched allocations)
// and batch responses (containing allocation results)
//
// Flow diagram:
//
//	┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
//	│     Client      │    │  Processor Client   │    │  Processor Server   │
//	│  (Allocator/    │    │   (this package)    │    │                     │
//	│   Extension)    │    │                     │    │                     │
//	└─────────────────┘    └─────────────────────┘    └─────────────────────┘
//	         │                        │                          │
//	         │                        │ 1. Connect & Register    │
//	         │                        │    (bidirectional stream)│
//	         │                        ├──────────────────────────►
//	         │                        │                          │
//	         │ 2. Allocate(request)   │                          │
//	         ├────────────────────────►                          │
//	         │                        │                          │
//	         │                        │ 3. Add to hotBatch       │
//	         │                        │    (accumulate)          │
//	         │                        │                          │
//	         │                        │ 4. Pull Request          │
//	         │                        │    (to all clients)      │
//	         │                        ◄──────────────────────────┤
//	         │                        │ 5. Send BatchRequest     │
//	         │                        │    (hotBatch)            │
//	         │                        ├──────────────────────────►
//	         │                        │                          │
//	         │                        │                          │ 6. Process
//	         │                        │                          │    allocations
//	         │                        │                          │
//	         │                        │ 7. BatchResponse         │
//	         │                        │    (results)             │
//	         │                        ◄──────────────────────────┤
//	         │ 8. Return result       │                          │
//	         ◄────────────────────────┤                          │
//	         │                        │                          │
//
//	Note: Multiple Processor Clients can connect to one Processor Server
//	      The server sends pull requests to all connected clients
//
//	Legend:
//	- Client: Makes allocation requests (allocator, extensions, etc.)
//	- Processor Client: Batches requests and manages communication
//	- Processor Server: Processes batched allocations from multiple clients
//	- Pull Request: Server asks all connected clients for pending requests
//	- BatchRequest: Client sends accumulated allocation requests
//	- BatchResponse: Server returns allocation results
package processor
