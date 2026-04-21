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

package gameservers

import (
	"context"
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// PerNodeCounter counts how many Allocated and
// Ready GameServers currently exist on each node.
// This is useful for scheduling allocations, fleet management
// mostly under a Packed strategy
//
//nolint:govet // ignore fieldalignment, singleton
type PerNodeCounter struct {
	logger           *logrus.Entry
	gameServerSynced cache.InformerSynced
	gameServerLister listerv1.GameServerLister
	countMutex       sync.RWMutex
	counts           map[string]*NodeCount
	processed        map[types.UID]processed
}

// processed tracks the last processed state of a GameServer to prevent duplicate event processing
type processed struct {
	resourceVersion string
	state           agonesv1.GameServerState
	nodeName        string
}

// NodeCount is just a convenience data structure for
// keeping relevant GameServer counts about Nodes
type NodeCount struct {
	// Ready is ready count
	Ready int64
	// Allocated is allocated out
	Allocated int64
}

// NewPerNodeCounter returns a new PerNodeCounter
func NewPerNodeCounter(
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *PerNodeCounter {

	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	pnc := &PerNodeCounter{
		gameServerSynced: gsInformer.HasSynced,
		gameServerLister: gameServers.Lister(),
		countMutex:       sync.RWMutex{},
		counts:           map[string]*NodeCount{},
		processed:        map[types.UID]processed{},
	}

	pnc.logger = runtime.NewLoggerWithType(pnc)

	_, _ = gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gs := obj.(*agonesv1.GameServer)

			pnc.countMutex.Lock()
			defer pnc.countMutex.Unlock()

			// Check if we've already processed this GameServer
			if processed, exists := pnc.processed[gs.ObjectMeta.UID]; exists {
				// Skip if same ResourceVersion (when set) and same state
				if processed.resourceVersion == gs.ObjectMeta.ResourceVersion &&
					processed.state == gs.Status.State {
					// Already processed this exact version, skip
					return
				}

				// If state changed, handle it as an update
				if processed.state != gs.Status.State {
					ready, allocated := pnc.calculateStateTransition(processed.state, gs.Status.State)
					updateProcessed(pnc.processed, gs)
					pnc.inc(gs, ready, allocated)
				}
				return
			}

			// Track this state
			updateProcessed(pnc.processed, gs)

			switch gs.Status.State {
			case agonesv1.GameServerStateReady:
				pnc.inc(gs, 1, 0)
			case agonesv1.GameServerStateAllocated:
				pnc.inc(gs, 0, 1)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldGS := oldObj.(*agonesv1.GameServer)
			newGS := newObj.(*agonesv1.GameServer)

			pnc.countMutex.Lock()
			defer pnc.countMutex.Unlock()

			// Check if we've already processed this exact state
			if pnc.isAlreadyProcessed(newGS.ObjectMeta.UID, newGS.ObjectMeta.ResourceVersion) {
				return
			}

			// Use the tracked previous state instead of oldGS to handle duplicates
			if processed, exists := pnc.processed[newGS.ObjectMeta.UID]; exists {
				oldGS = &agonesv1.GameServer{
					Status: agonesv1.GameServerStatus{
						State:    processed.state,
						NodeName: processed.nodeName,
					},
				}
			}

			ready, allocated := pnc.calculateStateTransition(oldGS.Status.State, newGS.Status.State)
			updateProcessed(pnc.processed, newGS)
			pnc.inc(newGS, ready, allocated)
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if !ok {
				return
			}

			pnc.countMutex.Lock()
			defer pnc.countMutex.Unlock()

			// Check if we've tracked this GameServer
			processed, exists := pnc.processed[gs.ObjectMeta.UID]
			if exists {
				// Use the tracked state for accurate counting, as the current state may not be
				// allocated or ready at this point (could very well be Shutdown).
				gs = &agonesv1.GameServer{
					Status: agonesv1.GameServerStatus{
						State:    processed.state,
						NodeName: processed.nodeName,
					},
				}
			}

			switch gs.Status.State {
			case agonesv1.GameServerStateReady:
				pnc.inc(gs, -1, 0)
			case agonesv1.GameServerStateAllocated:
				pnc.inc(gs, 0, -1)
			}

			// Remove from tracking since the object is deleted
			delete(pnc.processed, gs.ObjectMeta.UID)
		},
	})

	// remove the record when the node is deleted
	_, _ = kubeInformerFactory.Core().V1().Nodes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if !ok {
				return
			}

			pnc.countMutex.Lock()
			defer pnc.countMutex.Unlock()

			delete(pnc.counts, node.ObjectMeta.Name)
		},
	})

	return pnc
}

// Run sets up the current state GameServer counts across nodes
// non blocking Run function.
func (pnc *PerNodeCounter) Run(ctx context.Context, _ int) error {
	pnc.countMutex.Lock()
	defer pnc.countMutex.Unlock()

	pnc.logger.Debug("Running")

	if !cache.WaitForCacheSync(ctx.Done(), pnc.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	gsList, err := pnc.gameServerLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "error attempting to list all GameServers")
	}

	counts := map[string]*NodeCount{}
	processedGS := map[types.UID]processed{}

	for _, gs := range gsList {
		_, ok := counts[gs.Status.NodeName]
		if !ok {
			counts[gs.Status.NodeName] = &NodeCount{}
		}

		switch gs.Status.State {
		case agonesv1.GameServerStateReady:
			counts[gs.Status.NodeName].Ready++
		case agonesv1.GameServerStateAllocated:
			counts[gs.Status.NodeName].Allocated++
		}

		// Track this GameServer to prevent duplicate processing
		updateProcessed(processedGS, gs)
	}

	pnc.counts = counts
	pnc.processed = processedGS
	return nil
}

// Counts returns the NodeCount map in a thread safe way
func (pnc *PerNodeCounter) Counts() map[string]NodeCount {
	pnc.countMutex.RLock()
	defer pnc.countMutex.RUnlock()

	result := make(map[string]NodeCount, len(pnc.counts))

	// return a copy, so it's thread safe
	for k, v := range pnc.counts {
		result[k] = *v
	}

	return result
}

// incLocked increments the counts for a GameServer without acquiring the lock.
// The caller must hold the countMutex lock.
func (pnc *PerNodeCounter) inc(gs *agonesv1.GameServer, ready, allocated int64) {
	_, ok := pnc.counts[gs.Status.NodeName]
	if !ok {
		pnc.counts[gs.Status.NodeName] = &NodeCount{}
	}

	pnc.counts[gs.Status.NodeName].Allocated += allocated
	pnc.counts[gs.Status.NodeName].Ready += ready

	// just in case
	if pnc.counts[gs.Status.NodeName].Allocated < 0 {
		pnc.logger.WithField("node", gs.Status.NodeName).Warn("Allocated count went negative, resetting to 0")
		pnc.counts[gs.Status.NodeName].Allocated = 0
	}

	if pnc.counts[gs.Status.NodeName].Ready < 0 {
		pnc.counts[gs.Status.NodeName].Ready = 0
	}
}

// calculateStateTransition calculates the ready and allocated deltas when transitioning
// from oldState to newState.
func (pnc *PerNodeCounter) calculateStateTransition(oldState, newState agonesv1.GameServerState) (ready, allocated int64) {
	if oldState == agonesv1.GameServerStateReady && newState != agonesv1.GameServerStateReady {
		ready = -1
	} else if newState == agonesv1.GameServerStateReady && oldState != agonesv1.GameServerStateReady {
		ready = 1
	}

	if oldState == agonesv1.GameServerStateAllocated && newState != agonesv1.GameServerStateAllocated {
		allocated = -1
	} else if newState == agonesv1.GameServerStateAllocated && oldState != agonesv1.GameServerStateAllocated {
		allocated = 1
	}

	return ready, allocated
}

// isAlreadyProcessed checks if a GameServer with the given UID and ResourceVersion
// has already been processed. The caller must hold the countMutex lock.
func (pnc *PerNodeCounter) isAlreadyProcessed(uid types.UID, resourceVersion string) bool {
	if processed, exists := pnc.processed[uid]; exists {
		if processed.resourceVersion == resourceVersion {
			return true
		}
	}
	return false
}

// updateProcessed updates the tracking state for a GameServer in the specified map.
// The caller must hold the countMutex lock when updating pnc.processed.
func updateProcessed(processedMap map[types.UID]processed, gs *agonesv1.GameServer) {
	processedMap[gs.ObjectMeta.UID] = processed{
		resourceVersion: gs.ObjectMeta.ResourceVersion,
		state:           gs.Status.State,
		nodeName:        gs.Status.NodeName,
	}
}
