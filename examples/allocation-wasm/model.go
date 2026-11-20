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

import "encoding/json"

/*
A copy of the GameServer to avoid pulling in the entire Agones codebase into this example
(which won't compile to Wasm anyway).

The GameServer that is returned from the plugin replaces the GameServer that was passed in,
so any field that is not on these structs would be removed from the GameServer. Fields that
this plugin does not need to change are kept as json.RawMessage, so they are returned exactly
as they were passed in.
*/

// GameServer is the data structure for a GameServer resource.
type GameServer struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	// Spec is not changed by this plugin.
	Spec     json.RawMessage  `json:"spec,omitempty"`
	Status   GameServerStatus `json:"status"`
	Metadata ObjectMeta       `json:"metadata"`
}

// ObjectMeta is the metadata of a GameServer.
type ObjectMeta struct {
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	DeletionGrace     *int64            `json:"deletionGracePeriodSeconds,omitempty"`
	Name              string            `json:"name,omitempty"`
	GenerateName      string            `json:"generateName,omitempty"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	CreationTimestamp json.RawMessage   `json:"creationTimestamp,omitempty"`
	DeletionTimestamp json.RawMessage   `json:"deletionTimestamp,omitempty"`
	OwnerReferences   json.RawMessage   `json:"ownerReferences,omitempty"`
	Finalizers        []string          `json:"finalizers,omitempty"`
	ManagedFields     json.RawMessage   `json:"managedFields,omitempty"`
	Generation        int64             `json:"generation,omitempty"`
}

// GameServerStatus is the status for a GameServer resource.
type GameServerStatus struct {
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	Counters map[string]CounterStatus `json:"counters,omitempty"`
	// [Stage:Beta]
	// [FeatureFlag:CountsAndLists]
	Lists map[string]ListStatus `json:"lists,omitempty"`
	// Eviction specifies the eviction tolerance of the GameServer.
	Eviction *Eviction `json:"eviction,omitempty"`
	// GameServerState is the current state of a GameServer, e.g. Creating, Starting, Ready, etc
	State    string                 `json:"state"`
	Address  string                 `json:"address"`
	NodeName string                 `json:"nodeName"`
	Ports    []GameServerStatusPort `json:"ports"`
	// Addresses is the array of addresses at which the GameServer can be reached; copy of Node.Status.addresses.
	Addresses     []NodeAddress   `json:"addresses"`
	ReservedUntil json.RawMessage `json:"reservedUntil"`
}

// GameServerStatusPort shows the port that was allocated to a GameServer.
type GameServerStatusPort struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port"`
}

// NodeAddress contains information for the node's address.
type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// CounterStatus stores the current counter values and maximum capacity
type CounterStatus struct {
	Count    int64 `json:"count"`
	Capacity int64 `json:"capacity"`
}

// ListStatus stores the current list values and maximum capacity
type ListStatus struct {
	Values   []string `json:"values"`
	Capacity int64    `json:"capacity"`
}

// Eviction specifies the eviction tolerance of the GameServer
type Eviction struct {
	Safe string `json:"safe,omitempty"`
}
