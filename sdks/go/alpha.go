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

	"agones.dev/agones/pkg/sdk/alpha"
)

// Alpha is the struct for Alpha SDK functionality.
type Alpha struct {
	client alpha.SDKClient
}

// newAlpha creates a new Alpha SDK with the passed in connection.
func newAlpha(conn *grpc.ClientConn) *Alpha {
	return &Alpha{
		client: alpha.NewSDKClient(conn),
	}
}

// GetPlayerCapacity gets the last player capacity that was set through the SDK.
// If the player capacity is set from outside the SDK, use SDK.GameServer() instead.
func (a *Alpha) GetPlayerCapacity() (int64, error) {
	c, err := a.client.GetPlayerCapacity(context.Background(), &alpha.Empty{})
	return c.GetCount(), errors.Wrap(err, "could not get player capacity")
}

// SetPlayerCapacity changes the player capacity to a new value.
func (a *Alpha) SetPlayerCapacity(capacity int64) error {
	_, err := a.client.SetPlayerCapacity(context.Background(), &alpha.Count{Count: capacity})
	return errors.Wrap(err, "could not set player capacity")
}

// PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to status.players.id.
// Will return true and add the playerID to the list of playerIDs if the playerIDs was not already in the
// list of connected playerIDs.
func (a *Alpha) PlayerConnect(id string) (bool, error) {
	ok, err := a.client.PlayerConnect(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not register connected player")
}

// PlayerDisconnect Decreases the SDK’s stored player count by one, and removes the playerID from status.players.id.
// Will return true and remove the supplied playerID from the list of connected playerIDs if the
// playerID value exists within the list.
func (a *Alpha) PlayerDisconnect(id string) (bool, error) {
	ok, err := a.client.PlayerDisconnect(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not register disconnected player")
}

// GetPlayerCount returns the current player count.
func (a *Alpha) GetPlayerCount() (int64, error) {
	count, err := a.client.GetPlayerCount(context.Background(), &alpha.Empty{})
	return count.GetCount(), errors.Wrap(err, "could not get player count")
}

// IsPlayerConnected returns if the playerID is currently connected to the GameServer.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
func (a *Alpha) IsPlayerConnected(id string) (bool, error) {
	ok, err := a.client.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: id})
	return ok.GetBool(), errors.Wrap(err, "could not get if player is connected")
}

// GetConnectedPlayers returns the list of the currently connected player ids.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
func (a *Alpha) GetConnectedPlayers() ([]string, error) {
	list, err := a.client.GetConnectedPlayers(context.Background(), &alpha.Empty{})
	return list.GetList(), errors.Wrap(err, "could not list connected players")
}
