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
	"sort"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	informerv1 "agones.dev/agones/pkg/client/informers/externalversions/agones/v1"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

type matcher func(*agonesv1.GameServer) bool

// readyOrAllocatedGameServerMatcher returns true when a GameServer is in a Ready or Allocated state.
func readyOrAllocatedGameServerMatcher(gs *agonesv1.GameServer) bool {
	return gs.Status.State == agonesv1.GameServerStateReady || gs.Status.State == agonesv1.GameServerStateAllocated
}

// AllocationCache maintains a cache of GameServers that could potentially be allocated.
type AllocationCache struct {
	baseLogger       *logrus.Entry
	cache            gameServerCache
	gameServerLister listerv1.GameServerLister
	gameServerSynced cache.InformerSynced
	workerqueue      *workerqueue.WorkerQueue
	counter          *gameservers.PerNodeCounter
	matcher          matcher
}

// NewAllocationCache creates a new instance of AllocationCache
func NewAllocationCache(informer informerv1.GameServerInformer, counter *gameservers.PerNodeCounter, health healthcheck.Handler) *AllocationCache {
	c := &AllocationCache{
		gameServerSynced: informer.Informer().HasSynced,
		gameServerLister: informer.Lister(),
		counter:          counter,
		matcher:          readyOrAllocatedGameServerMatcher,
	}

	_, _ = informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			// only interested in if the old / new state was/is Ready
			oldGs := oldObj.(*agonesv1.GameServer)
			newGs := newObj.(*agonesv1.GameServer)
			key, ok := c.getKey(newGs)
			if !ok {
				return
			}
			if oldGs.ObjectMeta.ResourceVersion == newGs.ObjectMeta.ResourceVersion {
				return
			}
			switch {
			case newGs.IsBeingDeleted():
				c.cache.Delete(key)
			case c.matcher(newGs):
				c.cache.Store(key, newGs)
			case c.matcher(oldGs):
				c.cache.Delete(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if !ok {
				return
			}
			var key string
			if key, ok = c.getKey(gs); ok {
				c.cache.Delete(key)
			}
		},
	})

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.SyncGameServers, c.baseLogger, logfields.GameServerKey, agones.GroupName+".AllocationCache")
	health.AddLivenessCheck("allocationcache-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	return c
}

func (c *AllocationCache) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerKey, key)
}

// RemoveGameServer removes a gameserver from the cache of game servers
func (c *AllocationCache) RemoveGameServer(gs *agonesv1.GameServer) error {
	key, _ := cache.MetaNamespaceKeyFunc(gs)
	if ok := c.cache.Delete(key); !ok {
		return ErrConflictInGameServerSelection
	}
	return nil
}

// Sync builds the initial cache from the current set GameServers in the cluster
func (c *AllocationCache) Sync(ctx context.Context) error {
	c.baseLogger.Debug("Wait for AllocationCache cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// build the cache
	return c.syncCache()
}

// Resync enqueues an empty game server to be synced. Using queue helps avoiding multiple threads syncing at the same time.
func (c *AllocationCache) Resync() {
	// this will trigger syncing of the cache (assuming cache might not be up to date)
	c.workerqueue.EnqueueImmediately(&agonesv1.GameServer{})
}

// Run prepares cache to start
func (c *AllocationCache) Run(ctx context.Context) error {
	if err := c.Sync(ctx); err != nil {
		return err
	}
	// we don't want mutiple workers refresh cache at the same time so one worker will be better.
	// Also we don't expect to have too many failures when allocating
	go c.workerqueue.Run(ctx, 1)
	return nil
}

// AddGameServer adds a gameserver to the cache of allocatable GameServers
func (c *AllocationCache) AddGameServer(gs *agonesv1.GameServer) {
	key, _ := cache.MetaNamespaceKeyFunc(gs)

	c.cache.Store(key, gs)
}

// getGameServers returns a list of game servers in the cache.
func (c *AllocationCache) getGameServers() []*agonesv1.GameServer {
	length := c.cache.Len()
	if length == 0 {
		return nil
	}

	list := make([]*agonesv1.GameServer, 0, length)
	c.cache.Range(func(_ string, gs *agonesv1.GameServer) bool {
		list = append(list, gs)
		return true
	})
	return list
}

// ListSortedGameServers returns a list of the cached gameservers
// sorted by most allocated to least.
func (c *AllocationCache) ListSortedGameServers(gsa *allocationv1.GameServerAllocation) []*agonesv1.GameServer {
	list := c.getGameServers()
	if list == nil {
		return []*agonesv1.GameServer{}
	}
	counts := c.counter.Counts()

	sort.Slice(list, func(i, j int) bool {
		gs1 := list[i]
		gs2 := list[j]

		// Search Allocated GameServers first.
		if gs1.Status.State != gs2.Status.State {
			return gs1.Status.State == agonesv1.GameServerStateAllocated
		}

		c1, ok := counts[gs1.Status.NodeName]
		if !ok {
			return false
		}

		c2, ok := counts[gs2.Status.NodeName]
		if !ok {
			return true
		}

		if c1.Allocated > c2.Allocated {
			return true
		}
		if c1.Allocated < c2.Allocated {
			return false
		}

		// prefer nodes that have the most Ready gameservers on them - they are most likely to be
		// completely filled and least likely target for scale down.
		if c1.Ready < c2.Ready {
			return false
		}
		if c1.Ready > c2.Ready {
			return true
		}

		// if player tracking is enabled, prefer game servers with the least amount of room left
		if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
			if gs1.Status.Players != nil && gs2.Status.Players != nil {
				cap1 := gs1.Status.Players.Capacity - gs1.Status.Players.Count
				cap2 := gs2.Status.Players.Capacity - gs2.Status.Players.Count

				// if they are equal, pass the comparison through.
				if cap1 < cap2 {
					return true
				} else if cap2 < cap1 {
					return false
				}
			}
		}

		// if we end up here, then break the tie with Counter or List Priority.
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) && (gsa != nil) {
			if res := gs1.CompareCountAndListPriorities(gsa.Spec.Priorities, gs2); res != nil {
				return *res
			}
		}

		// finally sort lexicographically, so we have a stable order
		return gs1.GetObjectMeta().GetName() < gs2.GetObjectMeta().GetName()
	})

	return list
}

// ListSortedGameServersPriorities sorts and returns a list of game servers based on the
// list of Priorities.
func (c *AllocationCache) ListSortedGameServersPriorities(gsa *allocationv1.GameServerAllocation) []*agonesv1.GameServer {
	list := c.getGameServers()
	if list == nil {
		return []*agonesv1.GameServer{}
	}

	sort.Slice(list, func(i, j int) bool {
		gs1 := list[i]
		gs2 := list[j]

		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) && (gsa != nil) {
			if res := gs1.CompareCountAndListPriorities(gsa.Spec.Priorities, gs2); res != nil {
				return *res
			}
		}

		// finally sort lexicographically, so we have a stable order
		return gs1.GetObjectMeta().GetName() < gs2.GetObjectMeta().GetName()
	})

	return list
}

// SyncGameServers synchronises the GameServers to Gameserver cache. This is called when a failure
// happened during the allocation. This method will sync and make sure the cache is up to date.
func (c *AllocationCache) SyncGameServers(_ context.Context, key string) error {
	c.loggerForGameServerKey(key).Debug("Refreshing Allocation Gameserver cache")

	return c.syncCache()
}

// syncCache syncs the gameserver cache and updates the local cache for any changes.
func (c *AllocationCache) syncCache() error {
	// build the cache
	gsList, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not list GameServers")
	}

	// convert list of current gameservers to map for faster access
	currGameservers := make(map[string]*agonesv1.GameServer)
	for _, gs := range gsList {
		if key, ok := c.getKey(gs); ok {
			currGameservers[key] = gs
		}
	}

	// first remove the gameservers are not in the list anymore
	tobeDeletedGSInCache := make([]string, 0)
	c.cache.Range(func(key string, _ *agonesv1.GameServer) bool {
		if _, ok := currGameservers[key]; !ok {
			tobeDeletedGSInCache = append(tobeDeletedGSInCache, key)
		}
		return true
	})

	for _, staleGSKey := range tobeDeletedGSInCache {
		c.cache.Delete(staleGSKey)
	}

	// refresh the cache of possible allocatable GameServers
	for key, gs := range currGameservers {
		if gsCache, ok := c.cache.Load(key); ok {
			if !(gs.DeletionTimestamp.IsZero() && c.matcher(gs)) {
				c.cache.Delete(key)
			} else if gs.ObjectMeta.ResourceVersion != gsCache.ObjectMeta.ResourceVersion {
				c.cache.Store(key, gs)
			}
		} else if gs.DeletionTimestamp.IsZero() && c.matcher(gs) {
			c.cache.Store(key, gs)
		}
	}

	return nil
}

// getKey extract the key of gameserver object
func (c *AllocationCache) getKey(gs *agonesv1.GameServer) (string, bool) {
	var key string
	ok := true
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(gs); err != nil {
		ok = false
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(c.baseLogger.WithField("obj", gs), err)
	}
	return key, ok
}
