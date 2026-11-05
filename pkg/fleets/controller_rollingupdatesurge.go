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

package fleets

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// rollingUpdateActiveSurge uses the same budget before and after inactive GameServerSets reach
// zero desired replicas. Existing targets may be retained above that budget, but cannot grow
// beyond it; the Fleet target still limits retention when the Fleet is scaled down.
func (c *Controller) rollingUpdateActiveSurge(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	if fleet.Spec.Replicas == 0 {
		return 0, nil
	}

	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxSurge, int(fleet.Spec.Replicas), true)
	if err != nil {
		return 0, c.errs.Wrapf(err, "error parsing MaxSurge value: %s", fleet.ObjectMeta.Name)
	}
	surge := int32(r)

	// count the GameServers an inactive GameServerSet is still creating (Spec above Status) as well
	// as those it cannot remove because they are Allocated (Status above Spec).
	restReplicas := agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return max(gsSet.Spec.Replicas, gsSet.Status.Replicas)
	})
	blocked := agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Status.AllocatedReplicas + gsSet.Status.ReservedReplicas
	})

	surged := fleet.LowerBoundReplicas(fleet.UpperBoundReplicas(fleet.Spec.Replicas + surge - restReplicas))
	// Retain the current target without preventing an explicit Fleet scale down.
	retained := max(min(active.Spec.Replicas, fleet.Spec.Replicas-blocked), 0)
	replicas := max(retained, surged)

	// while the active GameServerSet is still creating GameServers, do not raise its target past
	// the GameServers already requested, but still lower it when the budget requires a scale down.
	// Status above Spec is not waited on: Allocated GameServers can keep it there indefinitely.
	if active.Spec.Replicas > active.Status.Replicas {
		replicas = min(replicas, active.Spec.Replicas)
	}

	loggerForFleet(fleet, c.baseLogger).WithField("gameserverset", active.ObjectMeta.Name).WithField("replicas", replicas).
		WithField("surge", surge).WithField("restReplicas", restReplicas).
		Debug("applying rolling update surge to active gameserverset")

	return replicas, nil
}
