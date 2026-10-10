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
	"google.golang.org/grpc"

	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/errors"
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
	b.errs = errors.FromStruct(&b)

	t.Parallel()

	t.Run("Set Counter and Set Capacity", func(t *testing.T) {
		count, err := b.GetCounterCount("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.GetCount(), count)

		capacity, err := b.GetCounterCapacity("sessions")
		assert.NoError(t, err)
		assert.Equal(t, sessions.GetCapacity(), capacity)

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

	//nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Decrement Counter Fails then Success", func(t *testing.T) {
		count, err := b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = b.DecrementCounter("games", 21)
		assert.Error(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = b.DecrementCounter("games", -12)
		assert.Error(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, games.GetCount(), count)

		err = b.DecrementCounter("games", 12)
		assert.NoError(t, err)

		count, err = b.GetCounterCount("games")
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	//nolint:dupl // testing DecrementCounter and IncrementCounter are not duplicates.
	t.Run("Increment Counter Fails then Success", func(t *testing.T) {
		count, err := b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

		err = b.IncrementCounter("gamers", 250)
		assert.Error(t, err)

		count, err = b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

		err = b.IncrementCounter("gamers", -237)
		assert.Error(t, err)

		count, err = b.GetCounterCount("gamers")
		assert.NoError(t, err)
		assert.Equal(t, gamers.GetCount(), count)

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
	b.errs = errors.FromStruct(&b)

	t.Parallel()

	t.Run("Get and Set List Capacity", func(t *testing.T) {
		capacity, err := b.GetListCapacity("foo")
		assert.NoError(t, err)
		assert.Equal(t, foo.GetCapacity(), capacity)

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
		assert.Equal(t, len(bar.GetValues()), length)

		values, err := b.GetListValues("bar")
		assert.NoError(t, err)
		assert.Equal(t, bar.GetValues(), values)

		err = b.AppendListValue("bar", "ghi")
		assert.NoError(t, err)

		length, err = b.GetListLength("bar")
		assert.NoError(t, err)
		assert.Equal(t, len(bar.GetValues())+1, length)

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
		assert.Equal(t, len(baz.GetValues()), length)

		values, err := b.GetListValues("baz")
		assert.NoError(t, err)
		assert.Equal(t, baz.GetValues(), values)

		err = b.DeleteListValue("baz", "456")
		assert.NoError(t, err)

		length, err = b.GetListLength("baz")
		assert.NoError(t, err)
		assert.Equal(t, len(baz.GetValues())-1, length)

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
	if counter, ok := b.counters[in.GetName()]; ok {
		return counter, nil
	}
	return nil, fmt.Errorf("counter not found: %s", in.GetName())
}

func (b *betaMock) UpdateCounter(ctx context.Context, in *beta.UpdateCounterRequest, _ ...grpc.CallOption) (*beta.Counter, error) {
	counter, err := b.GetCounter(ctx, &beta.GetCounterRequest{Name: in.GetCounterUpdateRequest().GetName()})
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

	b.counters[in.GetCounterUpdateRequest().GetName()] = counter
	return b.counters[in.GetCounterUpdateRequest().GetName()], nil
}

// GetList returns the list of betaMock. Note: unlike the SDK Server, this does not return
// a list with any pending batched changes applied.
func (b *betaMock) GetList(_ context.Context, in *beta.GetListRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, stderrors.New("GetListRequest cannot be nil")
	}
	if list, ok := b.lists[in.GetName()]; ok {
		return list, nil
	}
	return nil, fmt.Errorf("list not found: %s", in.GetName())
}

// Note: unlike the SDK Server, UpdateList does not batch changes and instead updates the list
// directly.
func (b *betaMock) UpdateList(_ context.Context, in *beta.UpdateListRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, stderrors.New("UpdateListRequest cannot be nil")
	}
	list, ok := b.lists[in.GetList().GetName()]
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
	b.lists[in.GetList().GetName()] = list
	return &beta.List{}, nil
}

// Note: unlike the SDK Server, AddListValue does not batch changes and instead updates the list
// directly.
func (b *betaMock) AddListValue(_ context.Context, in *beta.AddListValueRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, stderrors.New("AddListValueRequest cannot be nil")
	}
	list, ok := b.lists[in.GetName()]
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
	b.lists[in.GetName()] = list
	return &beta.List{}, nil
}

// Note: unlike the SDK Server, RemoveListValue does not batch changes and instead updates the list
// directly.
func (b *betaMock) RemoveListValue(_ context.Context, in *beta.RemoveListValueRequest, _ ...grpc.CallOption) (*beta.List, error) {
	if in == nil {
		return nil, stderrors.New("RemoveListValueRequest cannot be nil")
	}
	list, ok := b.lists[in.GetName()]
	if !ok {
		return nil, fmt.Errorf("list not found: %s", in.GetName())
	}
	for i, val := range list.GetValues() {
		if in.GetValue() != val {
			continue
		}
		list.Values = append(list.Values[:i], list.GetValues()[i+1:]...)
		b.lists[in.GetName()] = list
		return &beta.List{}, nil
	}
	return nil, fmt.Errorf("not found. Value: %s not found in List: %s", in.GetValue(), in.GetName())
}
