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

// Package sdk is the Go game server sdk.
package sdk

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// GameServerCallback is a function definition to be called
// when a GameServer CRD has been changed.
type GameServerCallback func(gs *sdk.GameServer)

// SDK is an instance of the Agones SDK.
type SDK struct {
	client sdk.SDKClient
	ctx    context.Context
	health sdk.SDK_HealthClient
	alpha  *Alpha
	beta   *Beta
	errs   *errors.Errors
}

// ErrorLog is a function to log the error.
type ErrorLog func(string, error)

// Logger is a pluggable function that outputs the error message to standard error.
var Logger ErrorLog = func(msg string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
}

// NewSDK starts a new SDK instance, defaulting to a connection at "localhost:9357"
// unless overridden by "AGONES_SDK_GRPC_HOST" and "AGONES_SDK_GRPC_PORT" environment variables.
// Blocks until connection and handshake are made.
// Times out after 30 seconds.
func NewSDK() (*SDK, error) {
	host := os.Getenv("AGONES_SDK_GRPC_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("AGONES_SDK_GRPC_PORT")
	if port == "" {
		port = "9357"
	}
	addr := fmt.Sprintf("%s:%s", host, port)
	s := &SDK{
		ctx: context.Background(),
	}
	s.errs = errors.FromStruct(s)
	// Block for at least 30 seconds.
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return s, s.errs.Wrapf(err, "could not connect to %s", addr)
	}
	if err := waitForReady(ctx, conn); err != nil {
		_ = conn.Close()
		return s, s.errs.Wrapf(err, "could not connect to %s", addr)
	}
	s.client = sdk.NewSDKClient(conn)
	s.health, err = s.client.Health(s.ctx)
	s.alpha = newAlpha(conn)
	s.beta = newBeta(conn)
	return s, s.errs.Wrap(err, "could not set up health check")
}

// connectivityWaiter is the part of *grpc.ClientConn that waitForReady uses.
type connectivityWaiter interface {
	Connect()
	GetState() connectivity.State
	WaitForStateChange(ctx context.Context, sourceState connectivity.State) bool
}

// waitForReady blocks until conn is connected or ctx is done, which is what
// the deprecated grpc.WithBlock dial option did. Like that option, it
// reconnects whenever the connection falls back to Idle, since no RPC is
// in flight yet to trigger one.
func waitForReady(ctx context.Context, conn connectivityWaiter) error {
	for {
		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			return nil
		case connectivity.Idle:
			conn.Connect()
		}
		if !conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// Alpha returns the Alpha SDK.
func (s *SDK) Alpha() *Alpha {
	return s.alpha
}

// Beta returns the Beta SDK.
func (s *SDK) Beta() *Beta {
	return s.beta
}

// Ready marks the Game Server as ready to receive connections.
func (s *SDK) Ready() error {
	_, err := s.client.Ready(s.ctx, &sdk.Empty{})
	return s.errs.Wrap(err, "could not send Ready message")
}

// Allocate self marks this gameserver as Allocated.
func (s *SDK) Allocate() error {
	_, err := s.client.Allocate(s.ctx, &sdk.Empty{})
	return s.errs.Wrap(err, "could not mark self as Allocated")
}

// Shutdown marks the Game Server as ready to shutdown.
func (s *SDK) Shutdown() error {
	_, err := s.client.Shutdown(s.ctx, &sdk.Empty{})
	return s.errs.Wrap(err, "could not send Shutdown message")
}

// Reserve marks the Game Server as Reserved for a given duration, at which point
// it will return the GameServer to a Ready state.
// Do note, the smallest unit available in the time.Duration argument is a second.
func (s *SDK) Reserve(d time.Duration) error {
	_, err := s.client.Reserve(s.ctx, &sdk.Duration{Seconds: int64(d.Seconds())})
	return s.errs.Wrap(err, "could not send Reserve message")
}

// Health sends a ping to the sidecar health check to indicate that this Game Server is healthy.
func (s *SDK) Health() error {
	return s.errs.Wrap(s.health.Send(&sdk.Empty{}), "could not send Health ping")
}

// SetLabel sets a metadata label on the `GameServer` with the prefix "agones.dev/sdk-".
func (s *SDK) SetLabel(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetLabel(s.ctx, kv)
	return s.errs.Wrap(err, "could not set label")
}

// SetAnnotation sets a metadata annotation on the `GameServer` with the prefix "agones.dev/sdk-".
func (s *SDK) SetAnnotation(key, value string) error {
	kv := &sdk.KeyValue{Key: key, Value: value}
	_, err := s.client.SetAnnotation(s.ctx, kv)
	return s.errs.Wrap(err, "could not set annotation")
}

// GameServer retrieve the GameServer details.
func (s *SDK) GameServer() (*sdk.GameServer, error) {
	gs, err := s.client.GetGameServer(s.ctx, &sdk.Empty{})
	return gs, s.errs.Wrap(err, "could not retrieve gameserver")
}

// WatchGameServer asynchronously calls the given GameServerCallback with the current GameServer
// configuration when the backing GameServer configuration is updated.
// This function can be called multiple times to add more than one GameServerCallback.
func (s *SDK) WatchGameServer(f GameServerCallback) error {
	stream, err := s.client.WatchGameServer(s.ctx, &sdk.Empty{})
	if err != nil {
		return s.errs.Wrap(err, "could not watch gameserver")
	}
	log := func(gs *sdk.GameServer, msg string, err error) {
		if gs == nil || gs.GetObjectMeta().GetDeletionTimestamp() == 0 {
			return
		}
		Logger(msg, err)
	}
	go func() {
		for {
			var gs *sdk.GameServer
			gs, err = stream.Recv()
			if err != nil {
				if stderrors.Is(err, io.EOF) {
					log(gs, "gameserver event stream EOF received", nil)
					return
				}
				log(gs, "error watching GameServer", err)
				// This is to wait for the reconnection, and not peg the CPU at 100%.
				time.Sleep(time.Second)
				continue
			}
			f(gs)
		}
	}()
	return nil
}

// GetCounterCount returns the Count for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) GetCounterCount(key string) (int64, error) {
	counter, err := s.client.GetCounter(s.ctx, &sdk.GetCounterRequest{Name: key})
	if err != nil {
		return -1, s.errs.Wrapf(err, "could not get Counter %s count", key)
	}
	return counter.GetCount(), nil
}

// IncrementCounter increases a counter by the given nonnegative integer amount.
// Will execute the increment operation against the current CRD value. Will max at max(int64).
// Will error if the key was not predefined in the GameServer resource on creation.
// Returns error if the count is at the current capacity (to the latest knowledge of the SDK),
// and no increment will occur.
//
// Note: A potential race condition here is that if count values are set from both the SDK and
// through the K8s API (Allocation or otherwise), since the SDK append operation back to the CRD
// value is batched asynchronous any value incremented past the capacity will be silently truncated.
func (s *SDK) IncrementCounter(key string, amount int64) error {
	if amount < 0 {
		return s.errs.Errorf("amount must be a positive int64, found %d", amount)
	}
	if amount == 0 {
		// The SDK server rejects an update with no changes, so only check the Counter exists.
		if _, err := s.client.GetCounter(s.ctx, &sdk.GetCounterRequest{Name: key}); err != nil {
			return s.errs.Wrapf(err, "could not increment Counter %s by amount %d", key, amount)
		}
		return nil
	}
	_, err := s.client.UpdateCounter(s.ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount,
		}})
	if err != nil {
		return s.errs.Wrapf(err, "could not increment Counter %s by amount %d", key, amount)
	}
	return nil
}

// DecrementCounter decreases the current count by the given nonnegative integer amount.
// The Counter Will not go below 0. Will execute the decrement operation against the current CRD value.
// Will error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
func (s *SDK) DecrementCounter(key string, amount int64) error {
	if amount < 0 {
		return s.errs.Errorf("amount must be a positive int64, found %d", amount)
	}
	if amount == 0 {
		// The SDK server rejects an update with no changes, so only check the Counter exists.
		if _, err := s.client.GetCounter(s.ctx, &sdk.GetCounterRequest{Name: key}); err != nil {
			return s.errs.Wrapf(err, "could not decrement Counter %s by amount %d", key, amount)
		}
		return nil
	}
	_, err := s.client.UpdateCounter(s.ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount * -1,
		}})
	if err != nil {
		return s.errs.Wrapf(err, "could not decrement Counter %s by amount %d", key, amount)
	}
	return nil
}

// SetCounterCount sets a count to the given value. Use with care, as this will overwrite any previous
// invocations’ value. Cannot be greater than Capacity.
func (s *SDK) SetCounterCount(key string, amount int64) error {
	_, err := s.client.UpdateCounter(s.ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{
			Name:  key,
			Count: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return s.errs.Wrapf(err, "could not set Counter %s count to amount %d", key, amount)
	}
	return nil
}

// GetCounterCapacity returns the Capacity for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) GetCounterCapacity(key string) (int64, error) {
	counter, err := s.client.GetCounter(s.ctx, &sdk.GetCounterRequest{Name: key})
	if err != nil {
		return -1, s.errs.Wrapf(err, "could not get Counter %s capacity", key)
	}
	return counter.GetCapacity(), nil
}

// SetCounterCapacity sets the capacity for the given Counter. A capacity of 0 is no capacity.
func (s *SDK) SetCounterCapacity(key string, amount int64) error {
	_, err := s.client.UpdateCounter(s.ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{
			Name:     key,
			Capacity: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return s.errs.Wrapf(err, "could not set Counter %s capacity to amount %d", key, amount)
	}
	return nil
}

// GetListCapacity returns the Capacity for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) GetListCapacity(key string) (int64, error) {
	list, err := s.client.GetList(s.ctx, &sdk.GetListRequest{Name: key})
	if err != nil {
		return -1, s.errs.Wrapf(err, "could not get List %s", key)
	}
	return list.GetCapacity(), nil
}

// SetListCapacity sets the capacity for a given list. Capacity must be between 0 and 1000.
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) SetListCapacity(key string, amount int64) error {
	_, err := s.client.UpdateList(s.ctx, &sdk.UpdateListRequest{
		List: &sdk.List{
			Name:     key,
			Capacity: amount,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
	})
	if err != nil {
		return s.errs.Wrapf(err, "could not set List %s capacity to amount %d", key, amount)
	}
	return nil
}

// ListContains returns if a string exists in a List's values list, given the List's key (name)
// and the string value. Search is case-sensitive.
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) ListContains(key, value string) (bool, error) {
	list, err := s.client.GetList(s.ctx, &sdk.GetListRequest{Name: key})
	if err != nil {
		return false, s.errs.Wrapf(err, "could not get List %s", key)
	}
	return slices.Contains(list.GetValues(), value), nil
}

// GetListLength returns the length of the Values list for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) GetListLength(key string) (int, error) {
	list, err := s.client.GetList(s.ctx, &sdk.GetListRequest{Name: key})
	if err != nil {
		return -1, s.errs.Wrapf(err, "could not get List %s", key)
	}
	return len(list.GetValues()), nil
}

// GetListValues returns the Values for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) GetListValues(key string) ([]string, error) {
	list, err := s.client.GetList(s.ctx, &sdk.GetListRequest{Name: key})
	if err != nil {
		return nil, s.errs.Wrapf(err, "could not get List %s", key)
	}
	return list.GetValues(), nil
}

// AppendListValue appends a string to a List's values list, given the List's key (name)
// and the string value. Will error if the string already exists in the list.
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) AppendListValue(key, value string) error {
	_, err := s.client.AddListValue(s.ctx, &sdk.AddListValueRequest{Name: key, Value: value})
	if err != nil {
		return s.errs.Wrapf(err, "could not get List %s", key)
	}
	return nil
}

// DeleteListValue removes a string from a List's values list, given the List's key (name)
// and the string value. Will error if the string does not exist in the list.
// Will error if the key was not predefined in the GameServer resource on creation.
func (s *SDK) DeleteListValue(key, value string) error {
	_, err := s.client.RemoveListValue(s.ctx, &sdk.RemoveListValueRequest{Name: key, Value: value})
	if err != nil {
		return s.errs.Wrapf(err, "could not get List %s", key)
	}
	return nil
}
