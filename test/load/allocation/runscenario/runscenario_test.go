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

package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var insecureDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())

// fakeAllocator returns the responses in its cycle in order, holding each request for delay.
// Concurrency is limited to slots, which caps its throughput at slots/delay.
type fakeAllocator struct {
	pb.UnimplementedAllocationServiceServer
	cycle    []error
	delay    time.Duration
	slots    chan struct{}
	mu       sync.Mutex
	n        int
	returned map[string]uint64
}

func newFakeAllocator(cycle []error, delay time.Duration, slots int) *fakeAllocator {
	return &fakeAllocator{cycle: cycle, delay: delay, slots: make(chan struct{}, slots), returned: map[string]uint64{}}
}

func (f *fakeAllocator) Allocate(context.Context, *pb.AllocationRequest) (*pb.AllocationResponse, error) {
	f.slots <- struct{}{}
	defer func() { <-f.slots }()
	f.mu.Lock()
	err := f.cycle[f.n%len(f.cycle)]
	f.n++
	if err == nil {
		f.returned[""]++
	} else {
		f.returned[status.Convert(err).Message()]++
	}
	f.mu.Unlock()
	time.Sleep(f.delay)
	if err != nil {
		return nil, err
	}
	return &pb.AllocationResponse{}, nil
}

func startFakeAllocator(t *testing.T, f *fakeAllocator) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	pb.RegisterAllocationServiceServer(srv, f)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func discardLogs(t *testing.T) {
	t.Helper()
	logger.SetOutput(io.Discard)
	t.Cleanup(func() { logger.SetOutput(os.Stdout) })
}

func TestErrorType(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err  error
		want allocErrorCode
	}{
		// messages returned by pkg/allocation/converters for UnAllocated and Contention allocations
		"no available game server": {status.Error(codes.ResourceExhausted, "there is no available GameServer to allocate"), noAvailableGameServer},
		"contention":               {status.Error(codes.Aborted, "too many concurrent requests have overwhelmed the system"), tooManyConcurrentRequests},
		"client deadline":          {status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error()), deadLineExceeded},
		"connection refused":       {status.Error(codes.Unavailable, "connection error: desc = \"transport: Error while dialing: dial tcp 10.0.0.1:443: connect: connection refused\""), connectionRefused},
		"connection timed out":     {status.Error(codes.Unavailable, "connection error: desc = \"transport: Error while dialing: dial tcp 10.0.0.1:443: connect: connection timed out\""), connectionTimedOut},
		"update conflict":          {status.Error(codes.Unknown, "could not update the selected GameServer"), unknown},
		"unrelated":                {errors.New("something else"), unknown},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, errorType(tc.err))
		})
	}
}

func TestRunScenarioErrorCountsMatchServer(t *testing.T) {
	discardLogs(t)
	cycle := []error{
		nil,
		status.Error(codes.ResourceExhausted, "there is no available GameServer to allocate"),
		status.Error(codes.Aborted, "too many concurrent requests have overwhelmed the system"),
		status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error()),
		status.Error(codes.Unknown, "could not update the selected GameServer"),
	}
	want := map[string]allocErrorCode{
		"there is no available GameServer to allocate":             noAvailableGameServer,
		"too many concurrent requests have overwhelmed the system": tooManyConcurrentRequests,
		context.DeadlineExceeded.Error():                           deadLineExceeded,
		"could not update the selected GameServer":                 unknown,
	}
	fake := newFakeAllocator(cycle, time.Millisecond, 1000)
	addr := startFakeAllocator(t, fake)

	// Two scenarios merged with add, the same way the final totals are calculated.
	total := summary{errors: allocErrorCodeCntMap()}
	for range 2 {
		total.add(runScenario(addr, insecureDialOpts, scenario{duration: 500 * time.Millisecond, numOfClients: 20, intervalMillisecond: 2}))
	}

	fake.mu.Lock()
	defer fake.mu.Unlock()
	require.NotZero(t, total.sent)
	assert.Equal(t, total.sent, total.succeeded+total.failed)
	assert.Equal(t, fake.returned[""], total.succeeded)
	assert.Len(t, total.latencies, int(total.succeeded))
	var failed uint64
	for msg, code := range want {
		failed += fake.returned[msg]
		assert.Equal(t, fake.returned[msg], total.errors[code], "error count for %s", code)
	}
	assert.Equal(t, failed, total.failed)
	assert.Len(t, total.errors, len(allocErrorCodeCntMap()))
}

func TestRunScenarioThroughputBoundedByCapacity(t *testing.T) {
	discardLogs(t)
	// The fake can serve at most 10 requests per 50ms (200/s), while the clients offer 400/s for 1s.
	const capacity = 200.0
	fake := newFakeAllocator([]error{nil}, 50*time.Millisecond, 10)
	addr := startFakeAllocator(t, fake)

	s := runScenario(addr, insecureDialOpts, scenario{duration: time.Second, numOfClients: 20, intervalMillisecond: 50})

	require.NotZero(t, s.succeeded)
	assert.Equal(t, s.sent, s.succeeded)
	assert.LessOrEqual(t, rate(s.succeeded, s.elapsed), capacity)
	assert.LessOrEqual(t, rate(s.succeededInWindow, s.window), capacity)
	assert.GreaterOrEqual(t, s.elapsed, s.window)
	for _, l := range s.latencies {
		assert.GreaterOrEqual(t, l, 50*time.Millisecond)
	}
	t.Logf("sent %v (offered %.1f/s), throughput %.1f/s, within window %.1f/s", s.sent, rate(s.sent, s.window), rate(s.succeeded, s.elapsed), rate(s.succeededInWindow, s.window))
}

func TestRunScenarioConnectionRefused(t *testing.T) {
	discardLogs(t)
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	require.NoError(t, lis.Close())

	s := runScenario(addr, insecureDialOpts, scenario{duration: 500 * time.Millisecond, numOfClients: 2, intervalMillisecond: 100})

	require.NotZero(t, s.sent)
	assert.Equal(t, s.sent, s.failed)
	assert.Equal(t, s.sent, s.errors[connectionRefused])
	assert.Zero(t, s.succeeded)
	assert.Zero(t, rate(s.succeeded, s.elapsed))
}

func TestPercentile(t *testing.T) {
	t.Parallel()
	sorted := make([]time.Duration, 100)
	for i := range sorted {
		sorted[i] = time.Duration(i+1) * time.Millisecond
	}
	assert.Equal(t, 50*time.Millisecond, percentile(sorted, 50))
	assert.Equal(t, 99*time.Millisecond, percentile(sorted, 99))
	assert.Equal(t, 100*time.Millisecond, percentile(sorted, 100))
	assert.Equal(t, 7*time.Millisecond, percentile([]time.Duration{7 * time.Millisecond}, 1))
}
