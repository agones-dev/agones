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

package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

const (
	fleetRolloutPercent                     = "fleet_rollout_percent"
	fleetReplicaCountName                   = "fleets_replicas_count"
	fleetAutoscalerBufferLimitName          = "fleet_autoscalers_buffer_limits"
	fleetAutoscalterBufferSizeName          = "fleet_autoscalers_buffer_size"
	fleetAutoscalerCurrentReplicaCountName  = "fleet_autoscalers_current_replicas_count"
	fleetAutoscalersDesiredReplicaCountName = "fleet_autoscalers_desired_replicas_count"
	fleetAutoscalersAbleToScaleName         = "fleet_autoscalers_able_to_scale"
	fleetAutoscalersLimitedName             = "fleet_autoscalers_limited"
	fleetCountersName                       = "fleet_counters"
	fleetListsName                          = "fleet_lists"
	gameServersCountName                    = "gameservers_count"
	gameServersTotalName                    = "gameservers_total"
	gameServersPlayerConnectedTotalName     = "gameserver_player_connected_total"
	gameServersPlayerCapacityTotalName      = "gameserver_player_capacity_total"
	nodeCountName                           = "nodes_count"
	gameServersNodeCountName                = "gameservers_node_count"
	gameServerStateDurationName             = "gameserver_state_duration"
)

var (
	// fleetAutoscalerViews are metric views associated with FleetAutoscalers
	fleetAutoscalerViews = []string{fleetAutoscalerBufferLimitName, fleetAutoscalterBufferSizeName, fleetAutoscalerCurrentReplicaCountName,
		fleetAutoscalersDesiredReplicaCountName, fleetAutoscalersAbleToScaleName, fleetAutoscalersLimitedName}
	// fleetViews are metric views associated with Fleets
	fleetViews = append([]string{fleetRolloutPercent, fleetReplicaCountName, gameServersCountName, gameServersTotalName, gameServersPlayerConnectedTotalName, gameServersPlayerCapacityTotalName, gameServerStateDurationName, fleetCountersName, fleetListsName}, fleetAutoscalerViews...)

	stateDurationSeconds           = []float64{0, 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384}
	fleetRolloutPercentStats       = stats.Int64("fleets/rollout_percent", "The current fleet rollout percentage", "1")
	fleetsReplicasCountStats       = stats.Int64("fleets/replicas_count", "The count of replicas per fleet", "1")
	fasBufferLimitsCountStats      = stats.Int64("fas/buffer_limits", "The buffer limits of autoscalers", "1")
	fasBufferSizeStats             = stats.Int64("fas/buffer_size", "The buffer size value of autoscalers", "1")
	fasCurrentReplicasStats        = stats.Int64("fas/current_replicas_count", "The current replicas cout as seen by autoscalers", "1")
	fasDesiredReplicasStats        = stats.Int64("fas/desired_replicas_count", "The desired replicas cout as seen by autoscalers", "1")
	fasAbleToScaleStats            = stats.Int64("fas/able_to_scale", "The fleet autoscaler can access the fleet to scale (0 indicates false, 1 indicates true)", "1")
	fasLimitedStats                = stats.Int64("fas/limited", "The fleet autoscaler is capped (0 indicates false, 1 indicates true)", "1")
	fleetCountersStats             = stats.Int64("fleets/counters", "Aggregated Counters counts and capacity across GameServers in the Fleet", "1")
	fleetListsStats                = stats.Int64("fleets/lists", "Aggregated Lists counts and capacity across GameServers in the Fleet", "1")
	gameServerCountStats           = stats.Int64("gameservers/count", "The count of gameservers", "1")
	gameServerTotalStats           = stats.Int64("gameservers/total", "The total of gameservers", "1")
	gameServerPlayerConnectedTotal = stats.Int64("gameservers/player_connected", "The total number of players connected to gameservers", "1")
	gameServerPlayerCapacityTotal  = stats.Int64("gameservers/player_capacity", "The available player capacity for gameservers", "1")
	nodesCountStats                = stats.Int64("nodes/count", "The count of nodes in the cluster", "1")
	gsPerNodesCountStats           = stats.Int64("gameservers_node/count", "The count of gameservers per node in the cluster", "1")
	gsStateDurationSec             = stats.Float64("gameservers_state/duration", "The duration of gameservers to be in a particular state", stats.UnitSeconds)

	stateViews = []*view.View{
		{
			Name:        fleetRolloutPercent,
			Measure:     fleetRolloutPercentStats,
			Description: "Measures the current progress of fleet rollout",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyType, keyNamespace},
		},
		{
			Name:        fleetReplicaCountName,
			Measure:     fleetsReplicasCountStats,
			Description: "The number of replicas per fleet",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyType, keyNamespace},
		},
		{
			Name:        fleetAutoscalerBufferLimitName,
			Measure:     fasBufferLimitsCountStats,
			Description: "The limits of buffer based fleet autoscalers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyType, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetAutoscalterBufferSizeName,
			Measure:     fasBufferSizeStats,
			Description: "The buffer size of fleet autoscalers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyType, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetAutoscalerCurrentReplicaCountName,
			Measure:     fasCurrentReplicasStats,
			Description: "The current replicas count as seen by autoscalers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetAutoscalersDesiredReplicaCountName,
			Measure:     fasDesiredReplicasStats,
			Description: "The desired replicas count as seen by autoscalers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetAutoscalersAbleToScaleName,
			Measure:     fasAbleToScaleStats,
			Description: "The fleet autoscaler can access the fleet to scale",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetAutoscalersLimitedName,
			Measure:     fasLimitedStats,
			Description: "The fleet autoscaler is capped",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyFleetName, keyNamespace},
		},
		{
			Name:        fleetCountersName,
			Measure:     fleetCountersStats,
			Description: "Aggregated Counters counts and capacity across GameServers in the Fleet",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyNamespace, keyType, keyCounter},
		},
		{
			Name:        fleetListsName,
			Measure:     fleetListsStats,
			Description: "Aggregated Lists counts and capacity across GameServers in the Fleet",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyName, keyNamespace, keyType, keyList},
		},
		{
			Name:        gameServersCountName,
			Measure:     gameServerCountStats,
			Description: "The number of gameservers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyType, keyFleetName, keyNamespace},
		},
		{
			Name:        gameServersTotalName,
			Measure:     gameServerTotalStats,
			Description: "The total of gameservers",
			Aggregation: view.Count(),
			TagKeys:     []tag.Key{keyType, keyFleetName, keyNamespace},
		},
		{
			Name:        gameServersPlayerConnectedTotalName,
			Measure:     gameServerPlayerConnectedTotal,
			Description: "The current amount of players connected in gameservers",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyFleetName, keyName, keyNamespace},
		},
		{
			Name:        gameServersPlayerCapacityTotalName,
			Measure:     gameServerPlayerCapacityTotal,
			Description: "The available player capacity per gameserver",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyFleetName, keyName, keyNamespace},
		},
		{
			Name:        nodeCountName,
			Measure:     nodesCountStats,
			Description: "The count of nodes in the cluster",
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{keyEmpty},
		},
		{
			Name:        gameServersNodeCountName,
			Measure:     gsPerNodesCountStats,
			Description: "The count of gameservers per node in the cluster",
			Aggregation: view.Distribution(0.00001, 1.00001, 2.00001, 3.00001, 4.00001, 5.00001, 6.00001, 7.00001, 8.00001, 9.00001, 10.00001, 11.00001, 12.00001, 13.00001, 14.00001, 15.00001, 16.00001, 32.00001, 40.00001, 50.00001, 60.00001, 70.00001, 80.00001, 90.00001, 100.00001, 110.00001, 120.00001),
		},
		{
			Name:        gameServerStateDurationName,
			Measure:     gsStateDurationSec,
			Description: "The time gameserver exists in the current state in seconds",
			Aggregation: view.Distribution(stateDurationSeconds...),
			TagKeys:     []tag.Key{keyType, keyFleetName, keyNamespace},
		},
	}
)

// register all our state views to OpenCensus
func registerViews() {
	for _, v := range stateViews {
		if err := view.Register(v); err != nil {
			logger.WithError(err).Error("could not register view")
		}
	}
}

// unregister views, this is only useful for tests as it trigger reporting.
func unRegisterViews() {
	for _, v := range stateViews {
		view.Unregister(v)
	}
}

// resetViews resets the values of an entire view.
// Since we have no way to delete a gauge, we have to reset
// the whole thing and start from scratch.
func resetViews(names []string) {
	for _, v := range stateViews {
		for _, name := range names {
			if v.Name == name {
				view.Unregister(v)
				if err := view.Register(v); err != nil {
					logger.WithError(err).Error("could not register view")
				}
			}
		}
	}
}
