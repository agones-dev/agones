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

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"agones.dev/agones/pkg/sdk/beta"
)

// Beta is the struct for Beta SDK functionality.
type Beta struct {
	client beta.SDKClient
}

// newBeta creates a new Beta SDK with the passed in connection.
func newBeta(conn *grpc.ClientConn) *Beta {
	return &Beta{
		client: beta.NewSDKClient(conn),
	}
}

// GetCounterCount returns the Count for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) GetCounterCount(key string) (int64, error) {
	counter, err := b.client.GetCounter(context.Background(), &beta.GetCounterRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get Counter %s count", key)
	}
	return counter.Count, nil
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
func (b *Beta) IncrementCounter(key string, amount int64) error {
	if amount < 0 {
		return errors.Errorf("amount must be a positive int64, found %d", amount)
	}
	_, err := b.client.UpdateCounter(context.Background(), &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount,
		}})
	if err != nil {
		return errors.Wrapf(err, "could not increment Counter %s by amount %d", key, amount)
	}
	return nil
}

// DecrementCounter decreases the current count by the given nonnegative integer amount.
// The Counter Will not go below 0. Will execute the decrement operation against the current CRD value.
// Will error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
func (b *Beta) DecrementCounter(key string, amount int64) error {
	if amount < 0 {
		return errors.Errorf("amount must be a positive int64, found %d", amount)
	}
	_, err := b.client.UpdateCounter(context.Background(), &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{
			Name:      key,
			CountDiff: amount * -1,
		}})
	if err != nil {
		return errors.Wrapf(err, "could not decrement Counter %s by amount %d", key, amount)
	}
	return nil
}

// SetCounterCount sets a count to the given value. Use with care, as this will overwrite any previous
// invocations’ value. Cannot be greater than Capacity.
func (b *Beta) SetCounterCount(key string, amount int64) error {
	_, err := b.client.UpdateCounter(context.Background(), &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{
			Name:  key,
			Count: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return errors.Wrapf(err, "could not set Counter %s count to amount %d", key, amount)
	}
	return nil
}

// GetCounterCapacity returns the Capacity for a Counter, given the Counter's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) GetCounterCapacity(key string) (int64, error) {
	counter, err := b.client.GetCounter(context.Background(), &beta.GetCounterRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get Counter %s capacity", key)
	}
	return counter.Capacity, nil
}

// SetCounterCapacity sets the capacity for the given Counter. A capacity of 0 is no capacity.
func (b *Beta) SetCounterCapacity(key string, amount int64) error {
	_, err := b.client.UpdateCounter(context.Background(), &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{
			Name:     key,
			Capacity: wrapperspb.Int64(amount),
		}})
	if err != nil {
		return errors.Wrapf(err, "could not set Counter %s capacity to amount %d", key, amount)
	}
	return nil
}

// GetListCapacity returns the Capacity for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) GetListCapacity(key string) (int64, error) {
	list, err := b.client.GetList(context.Background(), &beta.GetListRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get List %s", key)
	}
	return list.Capacity, nil
}

// SetListCapacity sets the capacity for a given list. Capacity must be between 0 and 1000.
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) SetListCapacity(key string, amount int64) error {
	_, err := b.client.UpdateList(context.Background(), &beta.UpdateListRequest{
		List: &beta.List{
			Name:     key,
			Capacity: amount,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
	})
	if err != nil {
		return errors.Wrapf(err, "could not set List %s capacity to amount %d", key, amount)
	}
	return nil
}

// ListContains returns if a string exists in a List's values list, given the List's key (name)
// and the string value. Search is case-sensitive.
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) ListContains(key, value string) (bool, error) {
	list, err := b.client.GetList(context.Background(), &beta.GetListRequest{Name: key})
	if err != nil {
		return false, errors.Wrapf(err, "could not get List %s", key)
	}
	for _, val := range list.Values {
		if val == value {
			return true, nil
		}
	}
	return false, nil
}

// GetListLength returns the length of the Values list for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) GetListLength(key string) (int, error) {
	list, err := b.client.GetList(context.Background(), &beta.GetListRequest{Name: key})
	if err != nil {
		return -1, errors.Wrapf(err, "could not get List %s", key)
	}
	return len(list.Values), nil
}

// GetListValues returns the Values for a List, given the List's key (name).
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) GetListValues(key string) ([]string, error) {
	list, err := b.client.GetList(context.Background(), &beta.GetListRequest{Name: key})
	if err != nil {
		return nil, errors.Wrapf(err, "could not get List %s", key)
	}
	return list.Values, nil
}

// AppendListValue appends a string to a List's values list, given the List's key (name)
// and the string value. Will error if the string already exists in the list.
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) AppendListValue(key, value string) error {
	_, err := b.client.AddListValue(context.Background(), &beta.AddListValueRequest{Name: key, Value: value})
	if err != nil {
		return errors.Wrapf(err, "could not get List %s", key)
	}
	return nil
}

// DeleteListValue removes a string from a List's values list, given the List's key (name)
// and the string value. Will error if the string does not exist in the list.
// Will error if the key was not predefined in the GameServer resource on creation.
func (b *Beta) DeleteListValue(key, value string) error {
	_, err := b.client.RemoveListValue(context.Background(), &beta.RemoveListValueRequest{Name: key, Value: value})
	if err != nil {
		return errors.Wrapf(err, "could not get List %s", key)
	}
	return nil
}
