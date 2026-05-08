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

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"agones.dev/agones/pkg/sdk/alpha"
)

func TestAlphaGetAndSetPlayerCapacity(t *testing.T) {
	mock := &alphaMock{}
	a := Alpha{
		client: mock,
	}

	err := a.SetPlayerCapacity(15)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), mock.capacity)

	capacity, err := a.GetPlayerCapacity()
	assert.NoError(t, err)
	assert.Equal(t, int64(15), capacity)

	playerID := "one"
	ok, err := a.PlayerConnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, playerID, mock.playerConnected)

	count, err := a.GetPlayerCount()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	ok, err = a.PlayerDisconnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, playerID, mock.playerDisconnected)

	// Put the player back in.
	ok, err = a.PlayerConnect(playerID)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(1), count)

	ok, err = a.IsPlayerConnected(playerID)
	assert.NoError(t, err)
	assert.True(t, ok, "Player should be connected")

	ok, err = a.IsPlayerConnected("false")
	assert.NoError(t, err)
	assert.False(t, ok, "Player should not be connected")

	list, err := a.GetConnectedPlayers()
	assert.NoError(t, err)
	assert.Equal(t, []string{playerID}, list)
}

type alphaMock struct {
	capacity           int64
	playerCount        int64
	playerConnected    string
	playerDisconnected string
}

func (a *alphaMock) PlayerConnect(_ context.Context, id *alpha.PlayerID, _ ...grpc.CallOption) (*alpha.Bool, error) {
	a.playerConnected = id.PlayerID
	a.playerCount++
	return &alpha.Bool{Bool: true}, nil
}

func (a *alphaMock) PlayerDisconnect(_ context.Context, id *alpha.PlayerID, _ ...grpc.CallOption) (*alpha.Bool, error) {
	a.playerDisconnected = id.PlayerID
	a.playerCount--
	return &alpha.Bool{Bool: true}, nil
}

func (a *alphaMock) IsPlayerConnected(_ context.Context, id *alpha.PlayerID, _ ...grpc.CallOption) (*alpha.Bool, error) {
	return &alpha.Bool{Bool: id.PlayerID == a.playerConnected}, nil
}

func (a *alphaMock) GetConnectedPlayers(_ context.Context, _ *alpha.Empty, _ ...grpc.CallOption) (*alpha.PlayerIDList, error) {
	return &alpha.PlayerIDList{List: []string{a.playerConnected}}, nil
}

func (a *alphaMock) SetPlayerCapacity(_ context.Context, in *alpha.Count, _ ...grpc.CallOption) (*alpha.Empty, error) {
	a.capacity = in.Count
	return &alpha.Empty{}, nil
}

func (a *alphaMock) GetPlayerCapacity(_ context.Context, _ *alpha.Empty, _ ...grpc.CallOption) (*alpha.Count, error) {
	return &alpha.Count{Count: a.capacity}, nil
}

func (a *alphaMock) GetPlayerCount(_ context.Context, _ *alpha.Empty, _ ...grpc.CallOption) (*alpha.Count, error) {
	return &alpha.Count{Count: a.playerCount}, nil
}
