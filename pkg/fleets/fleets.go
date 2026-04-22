/*
 * Copyright Contributors to Agones a Series of LF Projects, LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleets

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListGameServerSetsByFleetOwner lists all the GameServerSets for a given
// Fleet
func ListGameServerSetsByFleetOwner(gameServerSetNamespacedLister listerv1.GameServerSetNamespaceLister, f *agonesv1.Fleet) ([]*agonesv1.GameServerSet, error) {
	list, err := gameServerSetNamespacedLister.List(labels.SelectorFromSet(labels.Set{agonesv1.FleetNameLabel: f.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameserversets for fleet %s", f.ObjectMeta.Name)
	}

	var result []*agonesv1.GameServerSet
	for _, gsSet := range list {
		if metav1.IsControlledBy(gsSet, f) {
			result = append(result, gsSet)
		}
	}

	return result, nil
}

// ListGameServersByFleetOwner lists all GameServers that belong to a fleet through the
// GameServer -> GameServerSet -> Fleet owner chain
func ListGameServersByFleetOwner(gameServerNamespacedLister listerv1.GameServerNamespaceLister,
	fleet *agonesv1.Fleet) ([]*agonesv1.GameServer, error) {

	list, err := gameServerNamespacedLister.List(labels.SelectorFromSet(labels.Set{agonesv1.FleetNameLabel: fleet.ObjectMeta.Name}))
	if err != nil {
		return list, errors.Wrapf(err, "error listing gameservers for fleets %s", fleet.ObjectMeta.Name)
	}
	return list, nil
}
