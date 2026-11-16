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

package sdk

import (
	"context"
	stderrors "errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/errors"
)

func TestSDKGetAndUpdateCounter(t *testing.T) {
	mock := &countersListsMock{}
	// Counters must be predefined in the GameServer resource on creation.
	mock.counters = make(map[string]*sdk.Counter)

	sessions := sdk.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	games := sdk.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	gamers := sdk.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}

	mock.counters["sessions"] = &sdk.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	mock.counters["games"] = &sdk.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	mock.counters["gamers"] = &sdk.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}

	s := SDK{
		client: mock,
		ctx:    context.Background(),
	}
	s.errs = errors.FromStruct(&s)

	t.Parallel()

	t.Run("Set Counter and Set Capacity", func(t *testing.T) {
		count, err := s.GetCounterCount("sessions")
		require.NoError(t, err)
		assert.Equal(t, sessions.GetCount(), count)

		capacity, err := s.GetCounterCapacity("sessions")
		require.NoError(t, err)
		assert.Equal(t, sessions.GetCapacity(), capacity)

		wantCapacity := int64(25)
		err = s.SetCounterCapacity("sessions", wantCapacity)
		require.NoError(t, err)

		capacity, err = s.GetCounterCapacity("sessions")
		require.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)

		wantCount := int64(10)
		err = s.SetCounterCount("sessions", wantCount)
		require.NoError(t, err)

		count, err = s.GetCounterCount("sessions")
		require.NoError(t, err)
		assert.Equal(t, wantCount, count)
	})

	t.Run("Get and Set Non-Defined Counter", func(t *testing.T) {
		_, err := s.GetCounterCount("secessions")
		require.Error(t, err)

		_, err = s.GetCounterCapacity("secessions")
		require.Error(t, err)

		err = s.SetCounterCapacity("secessions", int64(100))
		require.Error(t, err)

		err = s.SetCounterCount("secessions", int64(0))
		assert.Error(t, err)
	})

	//nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Decrement Counter Fails then Success", func(t *testing.T) {
		count, err := s.GetCounterCount("games")
		require.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = s.DecrementCounter("games", 21)
		require.Error(t, err)

		count, err = s.GetCounterCount("games")
		require.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = s.DecrementCounter("games", -12)
		require.Error(t, err)

		count, err = s.GetCounterCount("games")
		require.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = s.DecrementCounter("games", 12)
		require.NoError(t, err)

		count, err = s.GetCounterCount("games")
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	//nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Increment Counter Fails then Success", func(t *testing.T) {
		count, err := s.GetCounterCount("gamers")
		require.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

		err = s.IncrementCounter("gamers", 250)
		require.Error(t, err)

		count, err = s.GetCounterCount("gamers")
		require.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

		err = s.IncrementCounter("gamers", -237)
		require.Error(t, err)

		count, err = s.GetCounterCount("gamers")
		require.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

		err = s.IncrementCounter("gamers", 237)
		require.NoError(t, err)

		count, err = s.GetCounterCount("gamers")
		require.NoError(t, err)
		assert.Equal(t, int64(500), count)
	})

}

func TestSDKGetAndUpdateList(t *testing.T) {
	mock := &countersListsMock{}
	// Lists must be predefined in the GameServer resource on creation.
	mock.lists = make(map[string]*sdk.List)

	foo := sdk.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	bar := sdk.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	baz := sdk.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	mock.lists["foo"] = &sdk.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	mock.lists["bar"] = &sdk.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	mock.lists["baz"] = &sdk.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	s := SDK{
		client: mock,
		ctx:    context.Background(),
	}
	s.errs = errors.FromStruct(&s)

	t.Parallel()

	t.Run("Get and Set List Capacity", func(t *testing.T) {
		capacity, err := s.GetListCapacity("foo")
		require.NoError(t, err)
		assert.Equal(t, foo.GetCapacity(), capacity)

		wantCapacity := int64(5)
		err = s.SetListCapacity("foo", wantCapacity)
		require.NoError(t, err)

		capacity, err = s.GetListCapacity("foo")
		require.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)
	})

	//nolint:dupl // the deprecated Beta SDK tests mirror the Stable SDK tests, until the Beta SDK is removed.
	t.Run("Get List Length, Get List Values, ListContains, and Append List Value", func(t *testing.T) {
		length, err := s.GetListLength("bar")
		require.NoError(t, err)
		assert.Equal(t, len(bar.GetValues()), length)

		values, err := s.GetListValues("bar")
		require.NoError(t, err)
		assert.Equal(t, bar.GetValues(), values)

		err = s.AppendListValue("bar", "ghi")
		require.NoError(t, err)

		length, err = s.GetListLength("bar")
		require.NoError(t, err)
		assert.Equal(t, len(bar.GetValues())+1, length)

		wantValues := []string{"abc", "def", "ghi"}
		values, err = s.GetListValues("bar")
		require.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := s.ListContains("bar", "ghi")
		require.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("Get List Length, Get List Values, ListContains, and Delete List Value", func(t *testing.T) {
		length, err := s.GetListLength("baz")
		require.NoError(t, err)
		assert.Equal(t, len(baz.GetValues()), length)

		values, err := s.GetListValues("baz")
		require.NoError(t, err)
		assert.Equal(t, baz.GetValues(), values)

		err = s.DeleteListValue("baz", "456")
		require.NoError(t, err)

		length, err = s.GetListLength("baz")
		require.NoError(t, err)
		assert.Equal(t, len(baz.GetValues())-1, length)

		wantValues := []string{"123", "789"}
		values, err = s.GetListValues("baz")
		require.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := s.ListContains("baz", "456")
		require.NoError(t, err)
		assert.False(t, contains)
	})

}

// countersListsMock implements the Counter and List methods of the stable SDK client.
// Other methods are not called by these tests, and panic via the nil embedded interface.
type countersListsMock struct {
	sdk.SDKClient
	counters map[string]*sdk.Counter
	lists    map[string]*sdk.List
}

func (m *countersListsMock) GetCounter(_ context.Context, in *sdk.GetCounterRequest, _ ...grpc.CallOption) (*sdk.Counter, error) {
	if counter, ok := m.counters[in.GetName()]; ok {
		return counter, nil
	}
	return nil, fmt.Errorf("counter not found: %s", in.GetName())
}

func (m *countersListsMock) UpdateCounter(ctx context.Context, in *sdk.UpdateCounterRequest, _ ...grpc.CallOption) (*sdk.Counter, error) {
	counter, err := m.GetCounter(ctx, &sdk.GetCounterRequest{Name: in.GetCounterUpdateRequest().GetName()})
	if err != nil {
		return nil, err
	}

	switch {
	case in.GetCounterUpdateRequest().GetCountDiff() != 0:
		count := counter.GetCount() + in.GetCounterUpdateRequest().GetCountDiff()
		if count < 0 || count > counter.GetCapacity() {
			return nil, fmt.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, counter.GetCapacity())
		}
		counter.Count = count
	case in.GetCounterUpdateRequest().GetCount() != nil:
		countSet := in.GetCounterUpdateRequest().GetCount().GetValue()
		if countSet < 0 || countSet > counter.GetCapacity() {
			return nil, fmt.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, counter.GetCapacity())
		}
		counter.Count = countSet
	case in.GetCounterUpdateRequest().GetCapacity() != nil:
		capacity := in.GetCounterUpdateRequest().GetCapacity().GetValue()
		if capacity < 0 {
			return nil, fmt.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d", capacity)
		}
		counter.Capacity = capacity
	default:
		return nil, fmt.Errorf("invalid argument. Malformed CounterUpdateRequest: %v",
			in.GetCounterUpdateRequest())
	}

	m.counters[in.GetCounterUpdateRequest().GetName()] = counter
	return m.counters[in.GetCounterUpdateRequest().GetName()], nil
}

// GetList returns the list of countersListsMock. Note: unlike the SDK Server, this does not return
// a list with any pending batched changes applied.
func (m *countersListsMock) GetList(_ context.Context, in *sdk.GetListRequest, _ ...grpc.CallOption) (*sdk.List, error) {
	if in == nil {
		return nil, stderrors.New("GetListRequest cannot be nil")
	}
	if list, ok := m.lists[in.GetName()]; ok {
		return list, nil
	}
	return nil, fmt.Errorf("list not found: %s", in.GetName())
}

// Note: unlike the SDK Server, UpdateList does not batch changes and instead updates the list
// directly.
//
//nolint:dupl // the deprecated Beta SDK tests mirror the Stable SDK tests, until the Beta SDK is removed.
func (m *countersListsMock) UpdateList(_ context.Context, in *sdk.UpdateListRequest, _ ...grpc.CallOption) (*sdk.List, error) {
	if in == nil {
		return nil, stderrors.New("UpdateListRequest cannot be nil")
	}
	list, ok := m.lists[in.GetList().GetName()]
	if !ok {
		return nil, fmt.Errorf("list not found: %s", in.GetList().GetName())
	}
	if in.GetList().GetCapacity() < 0 || in.GetList().GetCapacity() > 1000 {
		return nil, fmt.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", in.GetList().GetCapacity())
	}
	list.Capacity = in.GetList().GetCapacity()
	if len(list.GetValues()) > int(list.GetCapacity()) {
		list.Values = append([]string{}, list.GetValues()[:list.GetCapacity()]...)
	}
	m.lists[in.GetList().GetName()] = list
	return &sdk.List{}, nil
}

// Note: unlike the SDK Server, AddListValue does not batch changes and instead updates the list
// directly.
func (m *countersListsMock) AddListValue(_ context.Context, in *sdk.AddListValueRequest, _ ...grpc.CallOption) (*sdk.List, error) {
	if in == nil {
		return nil, stderrors.New("AddListValueRequest cannot be nil")
	}
	list, ok := m.lists[in.GetName()]
	if !ok {
		return nil, fmt.Errorf("list not found: %s", in.GetName())
	}
	if int(list.GetCapacity()) <= len(list.GetValues()) {
		return nil, fmt.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", list.GetCapacity(), len(list.GetValues()))
	}
	if slices.Contains(list.GetValues(), in.GetValue()) {
		return nil, fmt.Errorf("already exists. Value: %s already in List: %s", in.GetValue(), in.GetName())
	}
	list.Values = append(list.Values, in.GetValue())
	m.lists[in.GetName()] = list
	return &sdk.List{}, nil
}

// Note: unlike the SDK Server, RemoveListValue does not batch changes and instead updates the list
// directly.
func (m *countersListsMock) RemoveListValue(_ context.Context, in *sdk.RemoveListValueRequest, _ ...grpc.CallOption) (*sdk.List, error) {
	if in == nil {
		return nil, stderrors.New("RemoveListValueRequest cannot be nil")
	}
	list, ok := m.lists[in.GetName()]
	if !ok {
		return nil, fmt.Errorf("list not found: %s", in.GetName())
	}
	for i, val := range list.GetValues() {
		if in.GetValue() != val {
			continue
		}
		list.Values = append(list.Values[:i], list.GetValues()[i+1:]...)
		m.lists[in.GetName()] = list
		return &sdk.List{}, nil
	}
	return nil, fmt.Errorf("not found. Value: %s not found in List: %s", in.GetValue(), in.GetName())
}
