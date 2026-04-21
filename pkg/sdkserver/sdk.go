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

package sdkserver

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"google.golang.org/protobuf/proto"
)

const (
	// metadataPrefix prefix for labels and annotations
	metadataPrefix = "agones.dev/sdk-"
)

// convert converts a K8s GameServer object, into a gRPC SDK GameServer object
func convert(gs *agonesv1.GameServer) *sdk.GameServer {
	meta := gs.ObjectMeta
	status := gs.Status
	health := gs.Spec.Health
	result := &sdk.GameServer{
		ObjectMeta: &sdk.GameServer_ObjectMeta{
			Name:              meta.Name,
			Namespace:         meta.Namespace,
			Uid:               string(meta.UID),
			ResourceVersion:   meta.ResourceVersion,
			Generation:        meta.Generation,
			CreationTimestamp: meta.CreationTimestamp.Unix(),
			Annotations:       meta.Annotations,
			Labels:            meta.Labels,
		},
		Spec: &sdk.GameServer_Spec{
			Health: &sdk.GameServer_Spec_Health{
				Disabled:            health.Disabled,
				PeriodSeconds:       health.PeriodSeconds,
				FailureThreshold:    health.FailureThreshold,
				InitialDelaySeconds: health.InitialDelaySeconds,
			},
		},
		Status: &sdk.GameServer_Status{
			State:   string(status.State),
			Address: status.Address,
		},
	}
	if meta.DeletionTimestamp != nil {
		result.ObjectMeta.DeletionTimestamp = meta.DeletionTimestamp.Unix()
	}

	// look around and add all the addresses
	for _, a := range status.Addresses {
		result.Status.Addresses = append(result.Status.Addresses, &sdk.GameServer_Status_Address{
			Type:    string(a.Type),
			Address: a.Address,
		})
	}
	// loop around and add all the ports
	for _, p := range status.Ports {
		grpcPort := &sdk.GameServer_Status_Port{
			Name: p.Name,
			Port: p.Port,
		}
		result.Status.Ports = append(result.Status.Ports, grpcPort)
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		if gs.Status.Players != nil {
			result.Status.Players = &sdk.GameServer_Status_PlayerStatus{
				Count:    gs.Status.Players.Count,
				Capacity: gs.Status.Players.Capacity,
				Ids:      gs.Status.Players.IDs,
			}
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gs.Status.Counters != nil {
			counters := make(map[string]*sdk.GameServer_Status_CounterStatus, len(gs.Status.Counters))
			for key, counter := range gs.Status.Counters {
				counters[key] = &sdk.GameServer_Status_CounterStatus{Count: *proto.Int64(counter.Count), Capacity: *proto.Int64(counter.Capacity)}
			}
			result.Status.Counters = counters
		}

		if gs.Status.Lists != nil {
			lists := make(map[string]*sdk.GameServer_Status_ListStatus, len(gs.Status.Lists))
			for key, list := range gs.Status.Lists {
				lists[key] = &sdk.GameServer_Status_ListStatus{Capacity: *proto.Int64(list.Capacity), Values: list.DeepCopy().Values}
			}
			result.Status.Lists = lists
		}
	}

	return result
}
