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

package gameserversets

import (
	"context"
	"fmt"
	"time"

	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	logger = runtime.NewLoggerWithSource("metrics")

	keyName      = mt.MustTagKey("name")
	keyNamespace = mt.MustTagKey("namespace")
	keyFleetName = mt.MustTagKey("fleet_name")
	keyType      = mt.MustTagKey("type")

	gameServerCreationDuration = stats.Float64("gameserver_creation/duration", "The duration of gameserver creation", "s")

	stateViews = []*view.View{
		{
			Name:        "gameserver_creation_duration",
			Measure:     gameServerCreationDuration,
			Description: "The time gameserver takes to be created in seconds",
			Aggregation: view.Distribution(0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3),
			TagKeys:     []tag.Key{keyName, keyType, keyFleetName, keyNamespace},
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
	tag.Insert(keyName, "none"),
	tag.Insert(keyFleetName, "none"),
	tag.Insert(keyType, "none"),
}

type metrics struct {
	ctx              context.Context
	gameServerLister listerv1.GameServerLister
	logger           *logrus.Entry
	start            time.Time
}

// record the current current gameserver creation latency
func (r *metrics) record() {
	stats.Record(r.ctx, gameServerCreationDuration.M(time.Since(r.start).Seconds()))
}

// mutate the current set of metric tags
func (r *metrics) mutate(m ...tag.Mutator) {
	var err error
	r.ctx, err = tag.New(r.ctx, m...)
	if err != nil {
		r.logger.WithError(err).Warn("failed to mutate request context.")
	}
}

// setError set the latency status tag as error.
func (r *metrics) setError(errorType string) {
	r.mutate(tag.Update(keyType, errorType))
}

// setRequest set request metric tags.
func (r *metrics) setRequest(count int) {
	r.mutate(tag.Update(keyName, fmt.Sprint(count)))
}
