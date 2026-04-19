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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	autoscalinglisterv1 "agones.dev/agones/pkg/client/listers/autoscaling/v1"
	fleetsv1 "agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/util/runtime"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	noneValue = "none"

	// GameServersStateCount is the size of LRU cache and should contain all gameservers state changes
	// Upper bound could be estimated as 10_000 of gameservers in total each moment, 10 state changes per each gameserver
	// and about 10 minutes for a game session, and 6 gameservers per hour.
	// For one hour 600k capacity would be enough, even if no records would be deleted.
	// And calcDuration algorithm is removing those records, which already has been changed (old statuses).
	// Key is Namespace, fleetName, GameServerName, State and float64 as value.
	// Roughly 256 + 63 + 63 + 16 + 4 = 400 bytes per every record.
	// In total we would have 229 MiB of space required to store GameServer State durations.
	GameServersStateCount = 600_000
)

var (
	// MetricResyncPeriod is the interval to re-synchronize metrics based on indexed cache.
	MetricResyncPeriod = time.Second * 15
)

func init() {
	registerViews()
}

// Controller is a metrics controller collecting Agones state metrics
//
//nolint:govet // ignore fieldalignment, singleton
type Controller struct {
	logger                    *logrus.Entry
	gameServerLister          listerv1.GameServerLister
	nodeLister                v1.NodeLister
	gameServerSynced          cache.InformerSynced
	fleetSynced               cache.InformerSynced
	fleetLister               listerv1.FleetLister
	gameServerSetLister       listerv1.GameServerSetLister
	fasSynced                 cache.InformerSynced
	fasLister                 autoscalinglisterv1.FleetAutoscalerLister
	lock                      sync.Mutex
	stateLock                 sync.Mutex
	gsCount                   GameServerCount
	faCount                   map[string]int64
	gameServerStateLastChange *lru.Cache
	now                       func() time.Time
}

// NewController returns a new metrics controller
func NewController(
	_ kubernetes.Interface,
	_ versioned.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServer := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServer.Informer()

	fleets := agonesInformerFactory.Agones().V1().Fleets()
	fInformer := fleets.Informer()
	fas := agonesInformerFactory.Autoscaling().V1().FleetAutoscalers()
	fasInformer := fas.Informer()
	node := kubeInformerFactory.Core().V1().Nodes()

	gameServerSets := agonesInformerFactory.Agones().V1().GameServerSets()

	// GameServerStateLastChange Contains the time when the GameServer
	// changed its state last time
	// on delete and state change remove GameServerName key
	lruCache, err := lru.New(GameServersStateCount)
	if err != nil {
		logger.WithError(err).Fatal("Unable to create LRU cache")
	}

	c := &Controller{
		gameServerLister:          gameServer.Lister(),
		nodeLister:                node.Lister(),
		gameServerSynced:          gsInformer.HasSynced,
		fleetSynced:               fInformer.HasSynced,
		fleetLister:               fleets.Lister(),
		gameServerSetLister:       gameServerSets.Lister(),
		fasSynced:                 fasInformer.HasSynced,
		fasLister:                 fas.Lister(),
		gsCount:                   GameServerCount{},
		faCount:                   map[string]int64{},
		gameServerStateLastChange: lruCache,
		now:                       time.Now,
	}

	c.logger = runtime.NewLoggerWithType(c)

	_, _ = fInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.recordFleetChanges,
		UpdateFunc: func(_, next interface{}) {
			c.recordFleetChanges(next)
		},
		DeleteFunc: c.recordFleetDeletion,
	})

	_, _ = fasInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(added interface{}) {
			c.recordFleetAutoScalerChanges(nil, added)
		},
		UpdateFunc: c.recordFleetAutoScalerChanges,
		DeleteFunc: c.recordFleetAutoScalerDeletion,
	})

	_, _ = gsInformer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		UpdateFunc: c.recordGameServerStatusChanges,
	}, 0)

	return c
}

func (c *Controller) recordFleetAutoScalerChanges(old, next interface{}) {

	fas, ok := next.(*autoscalingv1.FleetAutoscaler)
	if !ok {
		return
	}

	// we looking for fleet name changes if that happens we need to reset
	// metrics for the old fas.
	if old != nil {
		if oldFas, ok := old.(*autoscalingv1.FleetAutoscaler); ok &&
			oldFas.Spec.FleetName != fas.Spec.FleetName {
			c.recordFleetAutoScalerDeletion(old)
		}
	}

	// do not record fleetautoscaler, delete event will do this.
	if fas.DeletionTimestamp != nil {
		return
	}

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fas.Name),
		tag.Upsert(keyFleetName, fas.Spec.FleetName), tag.Upsert(keyNamespace, fas.Namespace))

	ableToScale := 0
	limited := 0
	if fas.Status.AbleToScale {
		ableToScale = 1
	}
	if fas.Status.ScalingLimited {
		limited = 1
	}
	// recording status
	stats.Record(ctx,
		fasCurrentReplicasStats.M(int64(fas.Status.CurrentReplicas)),
		fasDesiredReplicasStats.M(int64(fas.Status.DesiredReplicas)),
		fasAbleToScaleStats.M(int64(ableToScale)),
		fasLimitedStats.M(int64(limited)))

	// recording buffer policy
	if fas.Spec.Policy.Buffer != nil {
		// recording limits
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "max")},
			fasBufferLimitsCountStats.M(int64(fas.Spec.Policy.Buffer.MaxReplicas)))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "min")},
			fasBufferLimitsCountStats.M(int64(fas.Spec.Policy.Buffer.MinReplicas)))

		// recording size
		if fas.Spec.Policy.Buffer.BufferSize.Type == intstr.String {
			// as percentage
			sizeString := fas.Spec.Policy.Buffer.BufferSize.StrVal
			if sizeString != "" {
				if size, err := strconv.Atoi(sizeString[:len(sizeString)-1]); err == nil {
					RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "percentage")},
						fasBufferSizeStats.M(int64(size)))
				}
			}
		} else {
			// as count
			RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "count")},
				fasBufferSizeStats.M(int64(fas.Spec.Policy.Buffer.BufferSize.IntVal)))
		}
	}
}

func (c *Controller) recordFleetAutoScalerDeletion(obj interface{}) {
	_, ok := obj.(*autoscalingv1.FleetAutoscaler)
	if !ok {
		return
	}

	if err := c.resyncFleetAutoScaler(); err != nil {
		c.logger.WithError(err).Warn("Could not resync Fleet Autoscaler metrics")
	}
}

func (c *Controller) recordFleetChanges(obj interface{}) {
	f, ok := obj.(*agonesv1.Fleet)
	if !ok {
		return
	}

	// do not record fleet, delete event will do this.
	if f.DeletionTimestamp != nil {
		return
	}

	c.recordFleetReplicas(f.Name, f.Namespace, f.Status.Replicas, f.Status.AllocatedReplicas,
		f.Status.ReadyReplicas, f.Spec.Replicas, f.Status.ReservedReplicas)

	c.recordFleetRolloutPercentage(f)

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if f.Status.Counters != nil {
			c.recordCounters(f.Name, f.Namespace, f.Status.Counters)
		}
		if f.Status.Lists != nil {
			c.recordLists(f.Name, f.Namespace, f.Status.Lists)
		}
	}
}

func (c *Controller) recordFleetRolloutPercentage(fleet *agonesv1.Fleet) {
	gameServerSetNamespacedLister := c.gameServerSetLister.GameServerSets(fleet.ObjectMeta.Namespace)
	list, err := fleetsv1.ListGameServerSetsByFleetOwner(gameServerSetNamespacedLister, fleet)
	if err != nil {
		c.logger.Errorf("Error listing GameServerSets for fleet %s in namespace %s: %v", fleet.Name, fleet.Namespace, err.Error())
		return
	}

	active, _ := c.filterGameServerSetByActive(fleet, list)

	if active == nil {
		fleetName := fleet.ObjectMeta.Namespace + "/" + fleet.ObjectMeta.Name
		c.logger.Debugf("Could not find active GameServerSet %s", fleetName)
		active = fleet.GameServerSet()
	}

	currentReplicas := active.Status.Replicas
	desiredReplicas := fleet.Spec.Replicas

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fleet.Name), tag.Upsert(keyNamespace, fleet.GetNamespace()))

	// Record current replicas count
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "current_replicas")},
		fleetRolloutPercentStats.M(int64(currentReplicas)))

	// Record desired replicas count
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "desired_replicas")},
		fleetRolloutPercentStats.M(int64(desiredReplicas)))
}

// filterGameServerSetByActive returns the active GameServerSet (or nil if it
// doesn't exist) and then the rest of the GameServerSets that are controlled
// by this Fleet
func (c *Controller) filterGameServerSetByActive(fleet *agonesv1.Fleet, list []*agonesv1.GameServerSet) (*agonesv1.GameServerSet, []*agonesv1.GameServerSet) {
	var active *agonesv1.GameServerSet
	var rest []*agonesv1.GameServerSet

	for _, gsSet := range list {
		if apiequality.Semantic.DeepEqual(gsSet.Spec.Template, fleet.Spec.Template) {
			active = gsSet
		} else {
			rest = append(rest, gsSet)
		}
	}

	return active, rest
}

func (c *Controller) recordFleetDeletion(obj interface{}) {
	_, ok := obj.(*agonesv1.Fleet)
	if !ok {
		return
	}

	if err := c.resyncFleets(); err != nil {
		// If for some reason resync fails, the entire metric state for fleets
		// will be reset whenever the next Fleet gets deleted, in which case
		// we end up back in a healthy state - so we aren't going to actively retry.
		c.logger.WithError(err).Warn("Could not resync Fleet Metrics")
	}
}

// resyncFleets resets all views associated with a Fleet, and recalculates all totals.
func (c *Controller) resyncFleets() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	fleets, err := c.fleetLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not resync Fleets")
	}

	fasList, err := c.fasLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not resync Fleets")
	}

	resetViews(fleetViews)
	for _, f := range fleets {
		c.recordFleetChanges(f)
	}
	for _, fas := range fasList {
		c.recordFleetAutoScalerChanges(nil, fas)
	}
	c.collectGameServerCounts()

	return nil
}

// resyncFleetAutoScaler resets all views associated with FleetAutoscalers, and recalculates metric totals.
func (c *Controller) resyncFleetAutoScaler() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	fasList, err := c.fasLister.List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "could not resync FleetAutoScalers")
	}

	resetViews(fleetAutoscalerViews)
	for _, fas := range fasList {
		c.recordFleetAutoScalerChanges(nil, fas)
	}

	return nil
}

func (c *Controller) recordFleetReplicas(fleetName, fleetNamespace string, total, allocated, ready, desired, reserved int32) {

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fleetName), tag.Upsert(keyNamespace, fleetNamespace))

	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total")},
		fleetsReplicasCountStats.M(int64(total)))
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated")},
		fleetsReplicasCountStats.M(int64(allocated)))
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "ready")},
		fleetsReplicasCountStats.M(int64(ready)))
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "desired")},
		fleetsReplicasCountStats.M(int64(desired)))
	RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "reserved")},
		fleetsReplicasCountStats.M(int64(reserved)))
}

// nolint:dupl // Linter errors on lines are duplicate of recordLists
func (c *Controller) recordCounters(fleetName, fleetNamespace string, counters map[string]agonesv1.AggregatedCounterStatus) {

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fleetName), tag.Upsert(keyNamespace, fleetNamespace))

	for counter, counterStatus := range counters {
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated_count"), tag.Upsert(keyCounter, counter)},
			fleetCountersStats.M(counterStatus.AllocatedCount))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated_capacity"), tag.Upsert(keyCounter, counter)},
			fleetCountersStats.M(counterStatus.AllocatedCapacity))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total_count"), tag.Upsert(keyCounter, counter)},
			fleetCountersStats.M(counterStatus.Count))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total_capacity"), tag.Upsert(keyCounter, counter)},
			fleetCountersStats.M(counterStatus.Capacity))
	}
}

// nolint:dupl // Linter errors on lines are duplicate of recordCounters
func (c *Controller) recordLists(fleetName, fleetNamespace string, lists map[string]agonesv1.AggregatedListStatus) {

	ctx, _ := tag.New(context.Background(), tag.Upsert(keyName, fleetName), tag.Upsert(keyNamespace, fleetNamespace))

	for list, listStatus := range lists {
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated_count"), tag.Upsert(keyList, list)},
			fleetListsStats.M(listStatus.AllocatedCount))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "allocated_capacity"), tag.Upsert(keyList, list)},
			fleetListsStats.M(listStatus.AllocatedCapacity))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total_count"), tag.Upsert(keyList, list)},
			fleetListsStats.M(listStatus.Count))
		RecordWithTags(ctx, []tag.Mutator{tag.Upsert(keyType, "total_capacity"), tag.Upsert(keyList, list)},
			fleetListsStats.M(listStatus.Capacity))
	}
}

// recordGameServerStatusChanged records gameserver status changes, however since it's based
// on cache events some events might collapsed and not appear, for example transition state
// like creating, port allocation, could be skipped.
// This is still very useful for final state, like READY, ERROR and since this is a counter
// (as opposed to gauge) you can aggregate using a rate, let's say how many gameserver are failing
// per second.
// Addition to the cache are not handled, otherwise resync would make metrics inaccurate by doubling
// current gameservers states.
func (c *Controller) recordGameServerStatusChanges(old, next interface{}) {
	newGs, ok := next.(*agonesv1.GameServer)
	if !ok {
		return
	}
	oldGs, ok := old.(*agonesv1.GameServer)
	if !ok {
		return
	}

	fleetName := newGs.Labels[agonesv1.FleetNameLabel]
	if fleetName == "" {
		fleetName = noneValue
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) &&
		newGs.Status.Players != nil &&
		oldGs.Status.Players != nil {

		if newGs.Status.Players.Count != oldGs.Status.Players.Count {
			RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyFleetName, fleetName),
				tag.Upsert(keyName, newGs.GetName()), tag.Upsert(keyNamespace, newGs.GetNamespace())}, gameServerPlayerConnectedTotal.M(newGs.Status.Players.Count))
		}

		if newGs.Status.Players.Capacity-newGs.Status.Players.Count != oldGs.Status.Players.Capacity-oldGs.Status.Players.Count {
			RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyFleetName, fleetName),
				tag.Upsert(keyName, newGs.GetName()), tag.Upsert(keyNamespace, newGs.GetNamespace())}, gameServerPlayerCapacityTotal.M(newGs.Status.Players.Capacity-newGs.Status.Players.Count))
		}

	}

	if newGs.Status.State != oldGs.Status.State {
		RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyType, string(newGs.Status.State)),
			tag.Upsert(keyFleetName, fleetName), tag.Upsert(keyNamespace, newGs.GetNamespace())}, gameServerTotalStats.M(1))

		// Calculate the duration of the current state
		duration, err := c.calcDuration(oldGs, newGs)
		if err != nil {
			c.logger.Warn(err.Error())
		} else {
			RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(keyType, string(oldGs.Status.State)),
				tag.Upsert(keyFleetName, fleetName), tag.Upsert(keyNamespace, newGs.GetNamespace())}, gsStateDurationSec.M(duration))
		}
	}
}

// calcDuration calculates the duration between state changes
// store current time from creationTimestamp for each update received
// Assumptions: there is a possibility that one of the previous state change timestamps would be evicted,
// this measurement would be skipped. This is a trade off between accuracy of distribution calculation and the performance.
// Presumably occasional miss would not change the statistics too much.
func (c *Controller) calcDuration(oldGs, newGs *agonesv1.GameServer) (duration float64, err error) {
	// currentTime - GameServer time from its start
	currentTime := c.now().UTC().Sub(newGs.ObjectMeta.CreationTimestamp.Local().UTC()).Seconds()

	fleetName := newGs.Labels[agonesv1.FleetNameLabel]
	if fleetName == "" {
		fleetName = defaultFleetTag
	}

	newGSKey := fmt.Sprintf("%s/%s/%s/%s", newGs.ObjectMeta.Namespace, fleetName, newGs.ObjectMeta.Name, newGs.Status.State)
	oldGSKey := fmt.Sprintf("%s/%s/%s/%s", oldGs.ObjectMeta.Namespace, fleetName, oldGs.ObjectMeta.Name, oldGs.Status.State)

	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	switch {
	case newGs.Status.State == agonesv1.GameServerStateCreating || newGs.Status.State == agonesv1.GameServerStatePortAllocation:
		duration = currentTime
	case !c.gameServerStateLastChange.Contains(oldGSKey):
		err = fmt.Errorf("unable to calculate '%s' state duration of '%s' GameServer", oldGs.Status.State, oldGs.ObjectMeta.Name)
		return 0, err
	default:
		val, ok := c.gameServerStateLastChange.Get(oldGSKey)
		if !ok {
			err = fmt.Errorf("could not find expected key %s", oldGSKey)
			return 0, err
		}
		c.gameServerStateLastChange.Remove(oldGSKey)
		duration = currentTime - val.(float64)
	}

	// Assuming that no State changes would occur after Shutdown
	if newGs.Status.State != agonesv1.GameServerStateShutdown {
		c.gameServerStateLastChange.Add(newGSKey, currentTime)
		c.logger.Debugf("Adding new key %s, relative time: %f", newGSKey, currentTime)
	}
	if duration < 0. {
		duration = 0
		err = fmt.Errorf("negative duration for '%s' state of '%s' GameServer", oldGs.Status.State, oldGs.ObjectMeta.Name)
	}
	return duration, err
}

// Run the Metrics controller. Will block until stop is closed.
// Collect metrics via cache changes and parse the cache periodically to record resource counts.
func (c *Controller) Run(ctx context.Context, _ int) error {
	c.logger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.fleetSynced, c.fasSynced) {
		return errors.New("failed to wait for caches to sync")
	}
	wait.Until(c.collect, MetricResyncPeriod, ctx.Done())
	return nil
}

// collect all metrics that are not event-based.
// this is fired periodically.
func (c *Controller) collect() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.collectGameServerCounts()
	c.collectNodeCounts()
}

// collects gameservers count by going through our informer cache
// this not meant to be called concurrently
func (c *Controller) collectGameServerCounts() {

	gameservers, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		c.logger.WithError(err).Warn("failed listing gameservers")
		return
	}

	if err := c.gsCount.record(gameservers); err != nil {
		c.logger.WithError(err).Warn("error while recoding stats")
	}
}

// collectNodeCounts count gameservers per node using informer cache.
func (c *Controller) collectNodeCounts() {
	gsPerNodes := map[string]int32{}

	gameservers, err := c.gameServerLister.List(labels.Everything())
	if err != nil {
		c.logger.WithError(err).Warn("failed listing gameservers")
		return
	}
	for _, gs := range gameservers {
		if gs.Status.NodeName != "" {
			gsPerNodes[gs.Status.NodeName]++
		}
	}

	nodes, err := c.nodeLister.List(labels.Everything())
	if err != nil {
		c.logger.WithError(err).Warn("failed listing gameservers")
		return
	}

	nodes = removeSystemNodes(nodes)
	RecordWithTags(context.Background(), []tag.Mutator{tag.Insert(keyEmpty, "true")},
		nodesCountStats.M(int64(len(nodes)-len(gsPerNodes))))
	RecordWithTags(context.Background(), []tag.Mutator{tag.Insert(keyEmpty, "false")},
		nodesCountStats.M(int64(len(gsPerNodes))))

	for _, node := range nodes {
		stats.Record(context.Background(), gsPerNodesCountStats.M(int64(gsPerNodes[node.Name])))
	}
}

func removeSystemNodes(nodes []*corev1.Node) []*corev1.Node {
	var result []*corev1.Node

	for _, n := range nodes {
		if !isSystemNode(n) {
			result = append(result, n)
		}
	}

	return result
}

// isSystemNode determines if a node is a system node, by checking if it has any taints starting with "agones.dev/"
func isSystemNode(n *corev1.Node) bool {
	for _, t := range n.Spec.Taints {
		if strings.HasPrefix(t.Key, "agones.dev/") {
			return true
		}
	}

	return false
}
