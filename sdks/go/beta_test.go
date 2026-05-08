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
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"agones.dev/agones/pkg/sdk/beta"
)

func TestBetaGetAndUpdateCounter(t *testing.T) {
	mock := &betaMock{}
	// Counters must be predefined in the GameServer resource on creation.
	mock.counters = make(map[string]*beta.Counter)

	sessions := beta.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	games := beta.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	gamers := beta.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}

	mock.counters["sessions"] = &beta.Counter{
		Name:     "sessions",
		Count:    21,
		Capacity: 42,
	}
	mock.counters["games"] = &beta.Counter{
		Name:     "games",
		Count:    12,
		Capacity: 24,
	}
	mock.counters["gamers"] = &beta.Counter{
		Name:     "gamers",
		Count:    263,
		Capacity: 500,
	}

	b := Beta{
		client: mock,
	}

	t.Parallel()

	t.Run("Set Counter and Set Capacity", func(t *testing.T) {
		count, err := b.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.Count, count)

		capacity, err := b.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.Capacity, capacity)

		wantCapacity := int64(25)
		err = b.SetCounterCapacity("sessions", wantCapacity)
		assert.NoError(t, err)

		capacity, err = b.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)

		wantCount := int64(10)
		err = b.SetCounterCount("sessions", wantCount)
		assert.NoError(t, err)

		count, err = b.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, wantCount, count)
	})

	t.Run("Get and Set Non-Defined Counter", func(t *testing.T) {
		_, err := b.GetCounterCount("secessions")
		assert.Error(t, err)

		_, err = b.GetCounterCapacity("secessions")
		assert.Error(t, err)

		err = b.SetCounterCapacity("secessions", int64(100))
		assert.Error(t, err)

		err = b.SetCounterCount("secessions", int64(0))
		assert.Error(t, err)
	})

	// nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Decrement Counter Fails then Success", func(t *testing.T) {
		count, err := b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		err = b.DecrementCounter("games", 21)
		assert.Error(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		err = b.DecrementCounter("games", -12)
		assert.Error(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.Count, count)

		err = b.DecrementCounter("games", 12)
		assert.NoError(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	// nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Increment Counter Fails then Success", func(t *testing.T) {
		count, err := b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		err = b.IncrementCounter("gamers", 250)
		assert.Error(t, err)

		count, err = b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		err = b.IncrementCounter("gamers", -237)
		assert.Error(t, err)

		count, err = b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.Count, count)

		err = b.IncrementCounter("gamers", 237)
		assert.NoError(t, err)

		count, err = b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, int64(500), count)
	})

}

func TestBetaGetAndUpdateList(t *testing.T) {
	mock := &betaMock{}
	// Lists must be predefined in the GameServer resource on creation.
	mock.lists = make(map[string]*beta.List)

	foo := beta.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	bar := beta.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	baz := beta.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	mock.lists["foo"] = &beta.List{
		Name:     "foo",
		Values:   []string{},
		Capacity: 2,
	}
	mock.lists["bar"] = &beta.List{
		Name:     "bar",
		Values:   []string{"abc", "def"},
		Capacity: 5,
	}
	mock.lists["baz"] = &beta.List{
		Name:     "baz",
		Values:   []string{"123", "456", "789"},
		Capacity: 5,
	}

	b := Beta{
		client: mock,
	}

	t.Parallel()

	t.Run("Get and Set List Capacity", func(t *testing.T) {
		capacity, err := b.GetListCapacity("foo")
		assert.NoError(t, err)
		assert.Equal(t, foo.Capacity, capacity)

		wantCapacity := int64(5)
		err = b.SetListCapacity("foo", wantCapacity)
		assert.NoError(t, err)

		capacity, err = b.GetListCapacity("foo")
		assert.NoError(t, err)
		assert.Equal(t, wantCapacity, capacity)
	})

	t.Run("Get List Length, Get List Values, ListContains, and Append List Value", func(t *testing.T) {
		length, err := b.GetListLength("bar")
		assert.NoError(t, err)
		assert.Equal(t, len(bar.Values), length)

		values, err := b.GetListValues("bar")
		assert.NoError(t, err)
		assert.Equal(t, bar.Values, values)

		err = b.AppendListValue("bar", "ghi")
		assert.NoError(t, err)

		length, err = b.GetListLength("bar")
		assert.NoError(t, err)
		assert.Equal(t, len(bar.Values)+1, length)

		wantValues := []string{"abc", "def", "ghi"}
		values, err = b.GetListValues("bar")
		assert.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := b.ListContains("bar", "ghi")
		assert.NoError(t, err)
		assert.True(t, contains)
	})

	t.Run("Get List Length, Get List Values, ListContains, and Delete List Value", func(t *testing.T) {
		length, err := b.GetListLength("baz")
		assert.NoError(t, err)
		assert.Equal(t, len(baz.Values), length)

		values, err := b.GetListValues("baz")
		assert.NoError(t, err)
		assert.Equal(t, baz.Values, values)

		err = b.DeleteListValue("baz", "456")
		assert.NoError(t, err)

		length, err = b.GetListLength("baz")
		assert.NoError(t, err)
		assert.Equal(t, len(baz.Values)-1, length)

		wantValues := []string{"123", "789"}
		values, err = b.GetListValues("baz")
		assert.NoError(t, err)
		assert.Equal(t, wantValues, values)

		contains, err := b.ListContains("baz", "456")
		assert.NoError(t, err)
		assert.False(t, contains)
	})

}

type betaMock struct {
	counters map[string]*beta.Counter
	lists    map[string]*beta.List
}

func (b *betaMock) GetCounter(_ context.Context, in *beta.GetCounterRequest, _ ...grpc.CallOption) (*beta.Counter, error) {
	if counter, ok := b.counters[in.Name]; ok {
		return counter, nil
	}
	return nil, errors.Errorf("counter not found: %s", in.Name)
}

func (b *betaMock) UpdateCounter(ctx context.Context, in *beta.UpdateCounterRequest, _ ...grpc.CallOption) (*beta.Counter, error) {
	counter, err := b.GetCounter(ctx, &beta.GetCounterRequest{Name: in.CounterUpdateRequest.Name})
	if err != nil {
		return nil, err
	}

	switch {
	case in.CounterUpdateRequest.CountDiff != 0:
		count := counter.Count + in.CounterUpdateRequest.CountDiff
		if count < 0 || count > counter.Capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, counter.Capacity)
		}
		counter.Count = count
	case in.CounterUpdateRequest.Count != nil:
		countSet := in.CounterUpdateRequest.Count.GetValue()
		if countSet < 0 || countSet > counter.Capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, counter.Capacity)
		}
		counter.Count = countSet
	case in.CounterUpdateRequest.Capacity != nil:
		capacity := in.CounterUpdateRequest.Capacity.GetValue()
		if capacity < 0 {
			return nil, errors.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d", capacity)
		}
		counter.Capacity = capacity
	default:
		return nil, errors.Errorf("invalid argument. Malformed CounterUpdateRequest: %v",
			in.CounterUpdateRequest)
	}

	b.counters[in.CounterUpdateRequest.Name] = counter
	return b.counters[in.CounterUpdateRequest.Name], nil
}

// GetList returns the list of betaMock. Note: unlike the SDK Server, this does not return
// a list with any pending batched changes applied.
func (b *betaMock) GetList(_ context.Context, in *beta.GetListRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, errors.Errorf("GetListRequest cannot be nil")
	}
	if list, ok := b.lists[in.Name]; ok {
		return list, nil
	}
	return nil, errors.Errorf("list not found: %s", in.Name)
}

// Note: unlike the SDK Server, UpdateList does not batch changes and instead updates the list
// directly.
func (b *betaMock) UpdateList(_ context.Context, in *beta.UpdateListRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, errors.Errorf("UpdateListRequest cannot be nil")
	}
	list, ok := b.lists[in.List.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.List.Name)
	}
	if in.List.Capacity < 0 || in.List.Capacity > 1000 {
		return nil, errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", in.List.Capacity)
	}
	list.Capacity = in.List.Capacity
	if len(list.Values) > int(list.Capacity) {
		list.Values = append([]string{}, list.Values[:list.Capacity]...)
	}
	b.lists[in.List.Name] = list
	return &beta.List{}, nil
}

// Note: unlike the SDK Server, AddListValue does not batch changes and instead updates the list
// directly.
func (b *betaMock) AddListValue(_ context.Context, in *beta.AddListValueRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, errors.Errorf("AddListValueRequest cannot be nil")
	}
	list, ok := b.lists[in.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.Name)
	}
	if int(list.Capacity) <= len(list.Values) {
		return nil, errors.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", list.Capacity, len(list.Values))
	}
	for _, val := range list.Values {
		if in.Value == val {
			return nil, errors.Errorf("already exists. Value: %s already in List: %s", in.Value, in.Name)
		}
	}
	list.Values = append(list.Values, in.Value)
	b.lists[in.Name] = list
	return &beta.List{}, nil
}

// Note: unlike the SDK Server, RemoveListValue does not batch changes and instead updates the list
// directly.
func (b *betaMock) RemoveListValue(_ context.Context, in *beta.RemoveListValueRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, errors.Errorf("RemoveListValueRequest cannot be nil")
	}
	list, ok := b.lists[in.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.Name)
	}
	for i, val := range list.Values {
		if in.Value != val {
			continue
		}
		list.Values = append(list.Values[:i], list.Values[i+1:]...)
		b.lists[in.Name] = list
		return &beta.List{}, nil
	}
	return nil, errors.Errorf("not found. Value: %s not found in List: %s", in.Value, in.Name)
}
