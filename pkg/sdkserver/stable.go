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

package sdkserver

import (
	"context"

	"agones.dev/agones/pkg/sdk"
)

// stableServer is the subset of the stable SDK service implemented by both SDKServer and LocalSDKServer.
// Their Counter and List handlers use the beta types, as Go cannot have two methods of the same name on one type.
type stableServer interface {
	Ready(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Allocate(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Health(sdk.SDK_HealthServer) error
	GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error)
	WatchGameServer(*sdk.Empty, sdk.SDK_WatchGameServerServer) error
	SetLabel(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	SetAnnotation(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	Reserve(context.Context, *sdk.Duration) (*sdk.Empty, error)
}

var (
	_ sdk.SDKServer = NewStableSDKServer(&SDKServer{})
	_ sdk.SDKServer = NewStableSDKServer(&LocalSDKServer{})
)

// unimplementedSDKServer is nested one level deeper than stableServer, so only the methods
// stableServer does not have (the Counter and List RPCs) are promoted from it.
type unimplementedSDKServer struct {
	sdk.UnimplementedSDKServer
}

// stableSDKServer serves the stable SDK service. The Counter and List RPCs return
// codes.Unimplemented until they are implemented on the stable API surface.
type stableSDKServer struct {
	stableServer
	unimplementedSDKServer
}

// NewStableSDKServer returns an implementation of the stable SDK service backed by s.
func NewStableSDKServer(s stableServer) sdk.SDKServer {
	return &stableSDKServer{stableServer: s}
}
