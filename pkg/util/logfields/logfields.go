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

// Package logfields specifies constants for use in logging.
package logfields

import (
	"github.com/sirupsen/logrus"
)

const (
	// NilGameServer is the value to show in logging when the GameServer is a nil value
	NilGameServer = "NilGameServer"
)

// ResourceType identifies the type of a resource for the purpose of putting it in structural logs.
type ResourceType string

// Identifiers used in logs for identifying Agones objects.
const (
	GameServerKey           ResourceType = "gsKey"
	GameServerSetKey        ResourceType = "gssKey"
	GameServerAllocationKey ResourceType = "gsaKey"
	FleetKey                ResourceType = "fleetKey"
	FleetAutoscalerKey      ResourceType = "fasKey"
)

// AugmentLogEntry creates derived log entry with a given resource identifier ("namespace/name")
func AugmentLogEntry(base *logrus.Entry, resourceType ResourceType, resourceID string) *logrus.Entry {
	return base.WithField(string(resourceType), resourceID)
}
