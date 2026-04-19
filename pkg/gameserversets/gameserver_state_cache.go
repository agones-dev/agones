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
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// gameServerSetCacheEntry manages a list of items created and deleted locally for a single game server set.
type gameServerSetCacheEntry struct {
	pendingCreation map[string]*agonesv1.GameServer
	pendingDeletion map[string]*agonesv1.GameServer
	mu              sync.Mutex
}

func (e *gameServerSetCacheEntry) created(gs *agonesv1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pendingCreation == nil {
		e.pendingCreation = map[string]*agonesv1.GameServer{}
	}
	e.pendingCreation[gs.Name] = gs.DeepCopy()
}

func (e *gameServerSetCacheEntry) deleted(gs *agonesv1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.pendingDeletion == nil {
		e.pendingDeletion = map[string]*agonesv1.GameServer{}
	}

	// Was pending creation, but deleted already.
	delete(e.pendingCreation, gs.Name)

	gsClone := gs.DeepCopy()
	t := metav1.Now()
	gsClone.ObjectMeta.DeletionTimestamp = &t
	e.pendingDeletion[gs.Name] = gsClone
}

// reconcileWithUpdatedServerList returns a list of game servers for a game server set taking into account
// the complete list of game servers passed as parameter and a list of pending creations and deletions.
func (e *gameServerSetCacheEntry) reconcileWithUpdatedServerList(list []*agonesv1.GameServer) []*agonesv1.GameServer {
	e.mu.Lock()
	defer e.mu.Unlock()

	var result []*agonesv1.GameServer

	found := map[string]bool{}

	for _, gs := range list {
		if d := e.pendingDeletion[gs.Name]; d != nil {
			if !gs.ObjectMeta.DeletionTimestamp.IsZero() {
				// has deletion timestamp - return theirs
				result = append(result, d)
				delete(e.pendingDeletion, gs.Name)
			} else {
				result = append(result, d)
			}
		} else {
			// object not deleted locally, trust the list.
			result = append(result, gs)
		}

		if e.pendingCreation[gs.Name] != nil {
			// object was pending creation and now showed up in list results, remove local overlay.
			delete(e.pendingCreation, gs.Name)
		}

		found[gs.Name] = true
	}

	// now delete from 'pendingDeletion' all the items that were not found in the result.
	for gsName := range e.pendingDeletion {
		if !found[gsName] {
			// ("GSSC: %v is now fully deleted", gsName)
			delete(e.pendingDeletion, gsName)
		}
	}

	// add all game servers that are pending creation which were not in the list
	for _, gs := range e.pendingCreation {
		result = append(result, gs)
	}

	return result
}

// gameServerStateCache manages per-GSS cache of items created and deleted by this controller process
// to compensate for latency due to eventual consistency between client actions and K8s API server.
type gameServerStateCache struct {
	cache sync.Map
}

func (c *gameServerStateCache) forGameServerSet(gsSet *agonesv1.GameServerSet) *gameServerSetCacheEntry {
	v, _ := c.cache.LoadOrStore(gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name, &gameServerSetCacheEntry{})
	return v.(*gameServerSetCacheEntry)
}

func (c *gameServerStateCache) deleteGameServerSet(gsSet *agonesv1.GameServerSet) {
	c.cache.Delete(gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name)
}
