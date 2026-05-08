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

/*
A copy of the FleetAutoscaleReview to avoid pulling in the entire Agones codebase into this example
(which won't compile to Wasm anyway).
*/

// AggregatedPlayerStatus stores total player tracking values
type AggregatedPlayerStatus struct {
	Count    int64 `json:"count"`
	Capacity int64 `json:"capacity"`
}

// AggregatedCounterStatus stores total and allocated Counter tracking values
type AggregatedCounterStatus struct {
	AllocatedCount    int64 `json:"allocatedCount"`
	AllocatedCapacity int64 `json:"allocatedCapacity"`
	Count             int64 `json:"count"`
	Capacity          int64 `json:"capacity"`
}

// AggregatedListStatus stores total and allocated List tracking values
type AggregatedListStatus struct {
	AllocatedCount    int64 `json:"allocatedCount"`
	AllocatedCapacity int64 `json:"allocatedCapacity"`
	Count             int64 `json:"count"`
	Capacity          int64 `json:"capacity"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerTracking]
	// Players are the current total player capacity and count for this Fleet
	// +optional
	Players *AggregatedPlayerStatus `json:"players,omitempty"`
	// (Beta, CountsAndLists feature flag) Counters provides aggregated Counter capacity and Counter
	// count for this Fleet.
	// +optional
	Counters map[string]AggregatedCounterStatus `json:"counters,omitempty"`
	// (Beta, CountsAndLists feature flag) Lists provides aggregated List capacityv and List values
	// for this Fleet.
	// +optional
	Lists map[string]AggregatedListStatus `json:"lists,omitempty"`
}

// FleetAutoscalerPolicyType is the policy for autoscaling
// for a given Fleet
type FleetAutoscalerPolicyType string

// FleetAutoscaleRequest defines the request to webhook autoscaler endpoint
type FleetAutoscaleRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the Autoscaler and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	UID string `json:"uid"`
	// Name is the name of the Fleet being scaled
	Name string `json:"name"`
	// Namespace is the namespace associated with the request (if any).
	Namespace string `json:"namespace"`
	// The Fleet's status values
	Status FleetStatus `json:"status"`
	// Standard map labels; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Standard map annotations; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FleetAutoscaleResponse defines the response of webhook autoscaler endpoint
type FleetAutoscaleResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding FleetAutoscaleRequest.
	UID string `json:"uid"`
	// Set to false if no scaling should occur to the Fleet
	Scale bool `json:"scale"`
	// The targeted replica count
	Replicas int32 `json:"replicas"`
}

// FleetAutoscaleReview is passed to the webhook with a populated Request value,
// and then returned with a populated Response.
type FleetAutoscaleReview struct {
	Request  *FleetAutoscaleRequest  `json:"request"`
	Response *FleetAutoscaleResponse `json:"response"`
}
