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
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// AllocationOverflowController watches `GameServerSets`, and those with configured
// AllocationOverflow settings, will the relevant labels and annotations to `GameServers` attached to the given
// `GameServerSet`
type AllocationOverflowController struct {
	baseLogger          *logrus.Entry
	counter             *gameservers.PerNodeCounter
	gameServerSynced    cache.InformerSynced
	gameServerGetter    getterv1.GameServersGetter
	gameServerLister    listerv1.GameServerLister
	gameServerSetSynced cache.InformerSynced
	gameServerSetLister listerv1.GameServerSetLister
	workerqueue         *workerqueue.WorkerQueue
}

// NewAllocatorOverflowController returns a new AllocationOverflowController
func NewAllocatorOverflowController(
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *AllocationOverflowController {
	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gameServerSet := agonesInformerFactory.Agones().V1().GameServerSets()
	gsSetInformer := gameServerSet.Informer()

	c := &AllocationOverflowController{
		counter:             counter,
		gameServerSynced:    gameServers.Informer().HasSynced,
		gameServerGetter:    agonesClient.AgonesV1(),
		gameServerLister:    gameServers.Lister(),
		gameServerSetSynced: gsSetInformer.HasSynced,
		gameServerSetLister: gameServerSet.Lister(),
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.baseLogger.Debug("Created!")
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServerSet, c.baseLogger, logfields.GameServerSetKey, agones.GroupName+".GameServerSetController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("gameserverset-allocationoverflow-workerqueue", c.workerqueue.Healthy)

	_, _ = gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			newGss := newObj.(*agonesv1.GameServerSet)

			// Only process if there is an AllocationOverflow, and it has labels or annotations.
			if newGss.Spec.AllocationOverflow == nil {
				return
			} else if len(newGss.Spec.AllocationOverflow.Labels) == 0 && len(newGss.Spec.AllocationOverflow.Annotations) == 0 {
				return
			}
			if newGss.Status.AllocatedReplicas > newGss.Spec.Replicas {
				c.workerqueue.Enqueue(newGss)
			}
		},
	})

	return c
}

// Run this controller. Will block until stop is closed.
func (c *AllocationOverflowController) Run(ctx context.Context) error {
	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.gameServerSetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(ctx, 1)
	return nil
}

// syncGameServerSet checks to see if there are overflow Allocated GameServers and applied the labels and/or
// annotations to the requisite number of GameServers needed to alert the underlying system.
func (c *AllocationOverflowController) syncGameServerSet(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(loggerForGameServerSetKey(c.baseLogger, key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gsSet, err := c.gameServerSetLister.GameServerSets(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			loggerForGameServerSetKey(c.baseLogger, key).Debug("GameServerSet is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServerSet %s from namespace %s", name, namespace)
	}

	// just in case something changed, double check to avoid panics and/or sending work to the K8s API that we don't
	// need to
	if gsSet.Spec.AllocationOverflow == nil {
		return nil
	}
	if gsSet.Status.AllocatedReplicas <= gsSet.Spec.Replicas {
		return nil
	}

	overflow := gsSet.Status.AllocatedReplicas - gsSet.Spec.Replicas

	list, err := ListGameServersByGameServerSetOwner(c.gameServerLister, gsSet)
	if err != nil {
		return err
	}

	matches, rest := gsSet.Spec.AllocationOverflow.CountMatches(list)
	if matches >= overflow {
		return nil
	}

	rest = SortGameServersByStrategy(gsSet.Spec.Scheduling, rest, c.counter.Counts(), gsSet.Spec.Priorities)
	rest = rest[:(overflow - matches)]

	opts := v1.UpdateOptions{}
	for _, gs := range rest {
		gsCopy := gs.DeepCopy()
		gsSet.Spec.AllocationOverflow.Apply(gsCopy)

		if _, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, opts); err != nil {
			return errors.Wrapf(err, "error updating GameServer %s with overflow labels and/or annotations", gs.ObjectMeta.Name)
		}
	}

	return nil
}
