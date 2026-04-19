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

package gameserverallocations

import (
	"context"
	"strconv"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyFleetName          = mt.MustTagKey("fleet_name")
	keyClusterName        = mt.MustTagKey("cluster_name")
	keyMultiCluster       = mt.MustTagKey("is_multicluster")
	keyStatus             = mt.MustTagKey("status")
	keySchedulingStrategy = mt.MustTagKey("scheduling_strategy")

	gameServerAllocationsLatency    = stats.Float64("gameserver_allocations/latency", "The duration of gameserver allocations", "s")
	gameServerAllocationsRetryTotal = stats.Int64("gameserver_allocations/errors", "The errors of gameserver allocations", "1")

	stateViews = []*view.View{
		{
			Name:        "gameserver_allocations_duration_seconds",
			Measure:     gameServerAllocationsLatency,
			Description: "The distribution of gameserver allocation requests latencies.",
			Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
		},
		{
			Name:        "gameserver_allocations_retry_total",
			Measure:     gameServerAllocationsRetryTotal,
			Description: "The count of gameserver allocation retry until it succeeds",
			Aggregation: view.Distribution(1, 2, 3, 4, 5),
			TagKeys:     []tag.Key{keyFleetName, keyClusterName, keyMultiCluster, keyStatus, keySchedulingStrategy},
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

// default set of tags for latency metric
var latencyTags = []tag.Mutator{
	tag.Insert(keyMultiCluster, "none"),
	tag.Insert(keyClusterName, "none"),
	tag.Insert(keySchedulingStrategy, "none"),
	tag.Insert(keyFleetName, "none"),
	tag.Insert(keyStatus, "none"),
}

type metrics struct {
	ctx              context.Context
	gameServerLister listerv1.GameServerLister
	logger           *logrus.Entry
	start            time.Time
}

// mutate the current set of metric tags
func (r *metrics) mutate(m ...tag.Mutator) {
	var err error
	r.ctx, err = tag.New(r.ctx, m...)
	if err != nil {
		r.logger.WithError(err).Warn("failed to mutate request context.")
	}
}

// setStatus set the latency status tag.
func (r *metrics) setStatus(status string) {
	r.mutate(tag.Update(keyStatus, status))
}

// setError set the latency status tag as error.
func (r *metrics) setError() {
	r.mutate(tag.Update(keyStatus, "error"))
}

// setRequest set request metric tags.
func (r *metrics) setRequest(in *allocationv1.GameServerAllocation) {
	tags := []tag.Mutator{
		tag.Update(keySchedulingStrategy, string(in.Spec.Scheduling)),
	}

	tags = append(tags, tag.Update(keyMultiCluster, strconv.FormatBool(in.Spec.MultiClusterSetting.Enabled)))
	r.mutate(tags...)
}

// setResponse set response metric tags.
func (r *metrics) setResponse(o k8sruntime.Object) {
	out, ok := o.(*allocationv1.GameServerAllocation)
	if out == nil || !ok {
		return
	}
	r.setStatus(string(out.Status.State))
	var tags []tag.Mutator
	// sets the fleet name tag if possible
	if out.Status.State == allocationv1.GameServerAllocationAllocated && out.Status.Source == localAllocationSource {
		gs, err := r.gameServerLister.GameServers(out.Namespace).Get(out.Status.GameServerName)
		if err != nil {
			r.logger.WithError(err).Warnf("failed to get gameserver:%s namespace:%s", out.Status.GameServerName, out.Namespace)
			return
		}
		fleetName := gs.Labels[agonesv1.FleetNameLabel]
		if fleetName != "" {
			tags = append(tags, tag.Update(keyFleetName, fleetName))
		}
	}
	r.mutate(tags...)
}

// record the current allocation latency.
func (r *metrics) record() {
	stats.Record(r.ctx, gameServerAllocationsLatency.M(time.Since(r.start).Seconds()))
}

// record the current allocation retry rate.
func (r *metrics) recordAllocationRetrySuccess(ctx context.Context, retryCount int) {
	mt.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyStatus, "Success")},
		gameServerAllocationsRetryTotal.M(int64(retryCount)))
}
