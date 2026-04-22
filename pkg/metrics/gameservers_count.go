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
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/errors"
)

const defaultFleetTag = "none"

// GameServerCount  is the count of gameserver per current state and per fleet name
type GameServerCount map[agonesv1.GameServerState]map[fleetKey]int64

type fleetKey struct {
	name      string
	namespace string
}

// increment adds the count of gameservers for a given fleetName and state
func (c GameServerCount) increment(key fleetKey, state agonesv1.GameServerState) {
	fleets, ok := c[state]
	if !ok {
		fleets = map[fleetKey]int64{}
		c[state] = fleets
	}
	fleets[key]++
}

// reset sets zero to the whole metrics set
func (c GameServerCount) reset() {
	for _, fleets := range c {
		for fleet := range fleets {
			fleets[fleet] = 0
		}
	}
}

// record counts the list of gameserver per status and fleet name and record it to OpenCensus
func (c GameServerCount) record(gameservers []*agonesv1.GameServer) error {
	// Currently there is no way to remove a metric so we have to reset our values to zero
	// so that statuses that have no count anymore are zeroed.
	// Otherwise OpenCensus will write the last value recorded to the prom endpoint.
	// TL;DR we can't remove a gauge
	c.reset()

	// only record gameservers's fleet count
	fleetNameMap := map[fleetKey]struct{}{}

	// counts gameserver per state and fleet
	for _, g := range gameservers {
		fleetName := g.Labels[agonesv1.FleetNameLabel]
		fleetNamespace := g.GetNamespace()
		key := fleetKey{name: fleetName, namespace: fleetNamespace}

		fleetNameMap[key] = struct{}{}
		c.increment(key, g.Status.State)
	}

	errs := []error{}
	deletedFleets := map[agonesv1.GameServerState][]fleetKey{}
	for state, fleets := range c {
		for fleet, count := range fleets {
			if _, ok := fleetNameMap[fleet]; !ok {
				if _, ok := deletedFleets[state]; !ok {
					deletedFleets[state] = []fleetKey{}
				}
				deletedFleets[state] = append(deletedFleets[state], fleet)
			}

			if fleet.name == "" {
				fleet.name = noneValue
			}
			if fleet.namespace == "" {
				fleet.namespace = noneValue
			}

			if err := stats.RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyType, string(state)),
				tag.Upsert(keyFleetName, fleet.name), tag.Upsert(keyNamespace, fleet.namespace)}, gameServerCountStats.M(count)); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for state, fleets := range deletedFleets {
		for _, fleet := range fleets {
			delete(c[state], fleet)
		}
	}
	return errors.NewAggregate(errs)
}
