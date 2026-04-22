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
	"encoding/json"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/google/go-cmp/cmp"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/tag"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	// ErrNoGameServerSetOwner is returned when a GameServerSet can't be found as an owner
	// for a GameServer
	ErrNoGameServerSetOwner = errors.New("No GameServerSet owner for this GameServer")
)

const (
	// gameServerErrorDeletionDelay is the minimum amount of time to delay the deletion
	// of a GameServer in Error state.
	gameServerErrorDeletionDelay = 30 * time.Second
)

// Extensions struct contains what is needed to bind webhook handlers
type Extensions struct {
	baseLogger *logrus.Entry
	apiHooks   agonesv1.APIHooks
}

func init() {
	registerViews()
}

// Controller is a GameServerSet controller
type Controller struct {
	baseLogger                     *logrus.Entry
	counter                        *gameservers.PerNodeCounter
	crdGetter                      apiextclientv1.CustomResourceDefinitionInterface
	gameServerGetter               getterv1.GameServersGetter
	gameServerLister               listerv1.GameServerLister
	gameServerSynced               cache.InformerSynced
	gameServerSetGetter            getterv1.GameServerSetsGetter
	gameServerSetLister            listerv1.GameServerSetLister
	gameServerSetSynced            cache.InformerSynced
	workerqueue                    *workerqueue.WorkerQueue
	recorder                       record.EventRecorder
	stateCache                     *gameServerStateCache
	allocationController           *AllocationOverflowController
	maxCreationParallelism         int
	maxGameServerCreationsPerBatch int
	maxDeletionParallelism         int
	maxGameServerDeletionsPerBatch int
	maxPodPendingCount             int
}

// NewController returns a new gameserverset crd controller
func NewController(
	health healthcheck.Handler,
	counter *gameservers.PerNodeCounter,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
	maxCreationParallelism int,
	maxDeletionParallelism int,
	maxGameServerCreationsPerBatch int,
	maxGameServerDeletionsPerBatch int,
	maxPodPendingCount int) *Controller {

	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()
	gameServerSets := agonesInformerFactory.Agones().V1().GameServerSets()
	gsSetInformer := gameServerSets.Informer()

	c := &Controller{
		crdGetter:                      extClient.ApiextensionsV1().CustomResourceDefinitions(),
		counter:                        counter,
		gameServerGetter:               agonesClient.AgonesV1(),
		gameServerLister:               gameServers.Lister(),
		gameServerSynced:               gsInformer.HasSynced,
		gameServerSetGetter:            agonesClient.AgonesV1(),
		gameServerSetLister:            gameServerSets.Lister(),
		gameServerSetSynced:            gsSetInformer.HasSynced,
		maxCreationParallelism:         maxCreationParallelism,
		maxDeletionParallelism:         maxDeletionParallelism,
		maxGameServerCreationsPerBatch: maxGameServerCreationsPerBatch,
		maxGameServerDeletionsPerBatch: maxGameServerDeletionsPerBatch,
		maxPodPendingCount:             maxPodPendingCount,
		stateCache:                     &gameServerStateCache{},
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServerSet, c.baseLogger, logfields.GameServerSetKey, agones.GroupName+".GameServerSetController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("gameserverset-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserverset-controller"})

	c.allocationController = NewAllocatorOverflowController(health, counter, agonesClient, agonesInformerFactory)

	_, _ = gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldGss := oldObj.(*agonesv1.GameServerSet)
			newGss := newObj.(*agonesv1.GameServerSet)
			if oldGss.Spec.Replicas != newGss.Spec.Replicas {
				c.workerqueue.Enqueue(newGss)
			}
		},
		DeleteFunc: func(gsSet interface{}) {
			c.stateCache.deleteGameServerSet(gsSet.(*agonesv1.GameServerSet))
		},
	})

	_, _ = gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gameServerEventHandler,
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)
			// ignore if already being deleted
			if gs.ObjectMeta.DeletionTimestamp == nil {
				c.gameServerEventHandler(gs)
			}
		},
		DeleteFunc: c.gameServerEventHandler,
	})

	return c
}

// NewExtensions binds the handlers to the webhook outside the initialization of the controller
// initializes a new logger for extensions.
func NewExtensions(apiHooks agonesv1.APIHooks, wh *webhooks.WebHook) *Extensions {
	ext := &Extensions{apiHooks: apiHooks}

	ext.baseLogger = runtime.NewLoggerWithType(ext)

	wh.AddHandler("/validate", agonesv1.Kind("GameServerSet"), admissionv1.Create, ext.creationValidationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("GameServerSet"), admissionv1.Update, ext.updateValidationHandler)

	return ext
}

// Run the GameServerSet controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(ctx context.Context, workers int) error {
	err := crd.WaitForEstablishedCRD(ctx, c.crdGetter, "gameserversets."+agones.GroupName, c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.gameServerSetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	go func() {
		if err := c.allocationController.Run(ctx); err != nil {
			c.baseLogger.WithError(err).Error("error running allocation overflow controller")
		}
	}()

	c.workerqueue.Run(ctx, workers)
	return nil
}

// updateValidationHandler that validates a GameServerSet when is updated
// Should only be called on gameserverset update operations.
func (ext *Extensions) updateValidationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	ext.baseLogger.WithField("review", review).Debug("updateValidationHandler")

	newGss := &agonesv1.GameServerSet{}
	oldGss := &agonesv1.GameServerSet{}

	newObj := review.Request.Object
	if err := json.Unmarshal(newObj.Raw, newGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling new GameServerSet json: %s", newObj.Raw)
	}

	oldObj := review.Request.OldObject
	if err := json.Unmarshal(oldObj.Raw, oldGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling old GameServerSet json: %s", oldObj.Raw)
	}

	if errs := oldGss.ValidateUpdate(newGss); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: review.Request.Kind.Group,
			Kind:  review.Request.Kind.Kind,
		}
		statusErr := k8serrors.NewInvalid(kind, review.Request.Name, errs)
		review.Response.Allowed = false
		review.Response.Result = &statusErr.ErrStatus
		loggerForGameServerSet(ext.baseLogger, newGss).WithField("review", review).Debug("Invalid GameServerSet update")
	}

	return review, nil
}

// creationValidationHandler that validates a GameServerSet when is created
// Should only be called on gameserverset create operations.
func (ext *Extensions) creationValidationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	ext.baseLogger.WithField("review", review).Debug("creationValidationHandler")

	newGss := &agonesv1.GameServerSet{}

	newObj := review.Request.Object
	if err := json.Unmarshal(newObj.Raw, newGss); err != nil {
		return review, errors.Wrapf(err, "error unmarshalling GameServerSet json after schema validation: %s", newObj.Raw)
	}

	if errs := newGss.Validate(ext.apiHooks); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: review.Request.Kind.Group,
			Kind:  review.Request.Kind.Kind,
		}
		statusErr := k8serrors.NewInvalid(kind, review.Request.Name, errs)
		review.Response.Allowed = false
		review.Response.Result = &statusErr.ErrStatus
		loggerForGameServerSet(ext.baseLogger, newGss).WithField("review", review).Debug("Invalid GameServerSet update")
	}

	return review, nil
}

func (c *Controller) gameServerEventHandler(obj interface{}) {
	gs, ok := obj.(*agonesv1.GameServer)
	if !ok {
		return
	}

	ref := metav1.GetControllerOf(gs)
	if ref == nil {
		return
	}
	gsSet, err := c.gameServerSetLister.GameServerSets(gs.ObjectMeta.Namespace).Get(ref.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.baseLogger.WithField("ref", ref).Debug("Owner GameServerSet no longer available for syncing")
		} else {
			runtime.HandleError(c.baseLogger.WithField("gsKey", gs.ObjectMeta.Namespace+"/"+gs.ObjectMeta.Name).WithField("ref", ref),
				errors.Wrap(err, "error retrieving GameServer owner"))
		}
		return
	}
	c.workerqueue.EnqueueImmediately(gsSet)
}

// syncGameServer synchronises the GameServers for the Set,
// making sure there are aways as many GameServers as requested
func (c *Controller) syncGameServerSet(ctx context.Context, key string) error {
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

	list, err := ListGameServersByGameServerSetOwner(c.gameServerLister, gsSet)
	if err != nil {
		return err
	}

	list = c.stateCache.forGameServerSet(gsSet).reconcileWithUpdatedServerList(list)

	numServersToAdd, toDelete, isPartial := computeReconciliationAction(gsSet.Spec.Scheduling, list, c.counter.Counts(),
		int(gsSet.Spec.Replicas), c.maxGameServerCreationsPerBatch, c.maxGameServerDeletionsPerBatch, c.maxPodPendingCount, gsSet.Spec.Priorities)

	// GameserverSet is marked for deletion then don't add gameservers.
	if !gsSet.DeletionTimestamp.IsZero() {
		numServersToAdd = 0
	}

	status := computeStatus(gsSet, list)
	fields := logrus.Fields{}

	for _, gs := range list {
		key := "gsCount" + string(gs.Status.State)
		if gs.ObjectMeta.DeletionTimestamp != nil {
			key += "Deleted"
		}
		v, ok := fields[key]
		if !ok {
			v = 0
		}

		fields[key] = v.(int) + 1
	}
	loggerForGameServerSet(c.baseLogger, gsSet).
		WithField("targetReplicaCount", gsSet.Spec.Replicas).
		WithField("numServersToAdd", numServersToAdd).
		WithField("numServersToDelete", len(toDelete)).
		WithField("isPartial", isPartial).
		WithField("status", status).
		WithFields(fields).
		Debug("Reconciling GameServerSet")
	if isPartial {
		// we've determined that there's work to do, but we've decided not to do all the work in one shot
		// make sure we get a follow-up, by re-scheduling this GSS in the worker queue immediately before this
		// function returns
		defer c.workerqueue.EnqueueImmediately(gsSet)
	}

	if numServersToAdd > 0 {
		if err := c.addMoreGameServers(ctx, gsSet, numServersToAdd); err != nil {
			loggerForGameServerSet(c.baseLogger, gsSet).WithError(err).Warning("error adding game servers")
			return errors.Wrap(err, "error adding game servers")
		}
	}

	if len(toDelete) > 0 {
		if err := c.deleteGameServers(ctx, gsSet, toDelete); err != nil {
			loggerForGameServerSet(c.baseLogger, gsSet).WithError(err).Warning("error deleting game servers")
			return errors.Wrap(err, "error deleting game servers")
		}
	}

	return c.syncGameServerSetStatus(ctx, gsSet, list)
}

// computeReconciliationAction computes the action to take to reconcile a game server set set given
// the list of game servers that were found and target replica count.
func computeReconciliationAction(strategy apis.SchedulingStrategy, list []*agonesv1.GameServer,
	counts map[string]gameservers.NodeCount, targetReplicaCount int, maxCreations int, maxDeletions int,
	maxPending int, priorities []agonesv1.Priority) (int, []*agonesv1.GameServer, bool) {
	var upCount int     // up == Ready or will become ready
	var deleteCount int // number of gameservers to delete

	// track the number of pods that are being created at any given moment by the GameServerSet
	// so we can limit it at a throughput that Kubernetes can handle
	var podPendingCount int // podPending == "up" but don't have a Pod running yet

	var potentialDeletions []*agonesv1.GameServer
	var toDelete []*agonesv1.GameServer

	scheduleDeletion := func(gs *agonesv1.GameServer) {
		toDelete = append(toDelete, gs)
		deleteCount--
	}

	handleGameServerUp := func(gs *agonesv1.GameServer) {
		if upCount >= targetReplicaCount {
			deleteCount++
		} else {
			upCount++
		}

		// Track gameservers that could be potentially deleted
		potentialDeletions = append(potentialDeletions, gs)
	}

	// pass 1 - count allocated/reserved servers only, since those can't be touched
	for _, gs := range list {
		if !gs.IsDeletable() {
			upCount++
		}
	}

	// pass 2 - handle all other statuses
	for _, gs := range list {
		if !gs.IsDeletable() {
			// already handled above
			continue
		}

		// GS being deleted don't count.
		if gs.IsBeingDeleted() {
			continue
		}

		switch gs.Status.State {
		case agonesv1.GameServerStatePortAllocation:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateCreating:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateStarting:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateScheduled:
			podPendingCount++
			handleGameServerUp(gs)
		case agonesv1.GameServerStateRequestReady:
			handleGameServerUp(gs)
		case agonesv1.GameServerStateReady:
			handleGameServerUp(gs)
		case agonesv1.GameServerStateReserved:
			handleGameServerUp(gs)

		// GameServerStateShutdown - already handled above
		// GameServerStateAllocated - already handled above
		case agonesv1.GameServerStateError:
			if !shouldDeleteErroredGameServer(gs) {
				// The GameServer is in an Error state and should not be deleted yet.
				// To stop an ever-increasing number of GameServers from being created,
				// consider the Error state GameServers as up and pending. This stops high
				// churn rate that can negatively impact Kubernetes.
				podPendingCount++
				handleGameServerUp(gs)
			} else {
				scheduleDeletion(gs)
			}

		case agonesv1.GameServerStateUnhealthy:
			scheduleDeletion(gs)
		default:
			// unrecognized state, assume it's up.
			handleGameServerUp(gs)
		}
	}

	var partialReconciliation bool
	var numServersToAdd int

	if upCount < targetReplicaCount {
		numServersToAdd = targetReplicaCount - upCount
		originalNumServersToAdd := numServersToAdd

		if numServersToAdd > maxCreations {
			numServersToAdd = maxCreations
		}

		if numServersToAdd+podPendingCount > maxPending {
			numServersToAdd = maxPending - podPendingCount
			if numServersToAdd < 0 {
				numServersToAdd = 0
			}
		}

		if originalNumServersToAdd != numServersToAdd {
			partialReconciliation = true
		}
	}

	if deleteCount > 0 {
		potentialDeletions = SortGameServersByStrategy(strategy, potentialDeletions, counts, priorities)
		toDelete = append(toDelete, potentialDeletions[0:deleteCount]...)
	}

	if len(toDelete) > maxDeletions {
		toDelete = toDelete[0:maxDeletions]
		partialReconciliation = true
	}

	return numServersToAdd, toDelete, partialReconciliation
}

func shouldDeleteErroredGameServer(gs *agonesv1.GameServer) bool {
	erroredAtStr := gs.Annotations[agonesv1.GameServerErroredAtAnnotation]
	if erroredAtStr == "" {
		return true
	}

	erroredAt, err := time.Parse(time.RFC3339, erroredAtStr)
	if err != nil {
		// The annotation is in the wrong format, delete the GameServer.
		return true
	}

	if time.Since(erroredAt) >= gameServerErrorDeletionDelay {
		return true
	}
	return false
}

// addMoreGameServers adds diff more GameServers to the set
func (c *Controller) addMoreGameServers(ctx context.Context, gsSet *agonesv1.GameServerSet, count int) (err error) {
	loggerForGameServerSet(c.baseLogger, gsSet).WithField("count", count).Debug("Adding more gameservers")
	latency := c.newMetrics(ctx)
	latency.setRequest(count)

	defer func() {
		if err != nil {
			latency.setError("error")
		}
		latency.record()

	}()

	return parallelize(newGameServersChannel(count, gsSet), c.maxCreationParallelism, func(gs *agonesv1.GameServer) error {
		gs, err := c.gameServerGetter.GameServers(gs.Namespace).Create(ctx, gs, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error creating gameserver for gameserverset %s", gsSet.ObjectMeta.Name)
		}

		c.stateCache.forGameServerSet(gsSet).created(gs)
		c.recorder.Eventf(gsSet, corev1.EventTypeNormal, "SuccessfulCreate", "Created gameserver: %s", gs.ObjectMeta.Name)
		return nil
	})
}

func (c *Controller) deleteGameServers(ctx context.Context, gsSet *agonesv1.GameServerSet, toDelete []*agonesv1.GameServer) error {
	loggerForGameServerSet(c.baseLogger, gsSet).WithField("diff", len(toDelete)).Debug("Deleting gameservers")

	return parallelize(gameServerListToChannel(toDelete), c.maxDeletionParallelism, func(gs *agonesv1.GameServer) error {
		// We should not delete the gameservers directly buy set their state to shutdown and let the gameserver controller to delete
		gsCopy := gs.DeepCopy()
		gsCopy.Status.State = agonesv1.GameServerStateShutdown
		_, err := c.gameServerGetter.GameServers(gs.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating gameserver %s from status %s to Shutdown status", gs.ObjectMeta.Name, gs.Status.State)
		}

		c.stateCache.forGameServerSet(gsSet).deleted(gs)
		c.recorder.Eventf(gsSet, corev1.EventTypeNormal, "SuccessfulDelete", "Deleted gameserver in state %s: %v", gs.Status.State, gs.ObjectMeta.Name)
		return nil
	})
}

func newGameServersChannel(n int, gsSet *agonesv1.GameServerSet) chan *agonesv1.GameServer {
	gameServers := make(chan *agonesv1.GameServer)
	go func() {
		defer close(gameServers)

		for i := 0; i < n; i++ {
			gameServers <- gsSet.GameServer()
		}
	}()

	return gameServers
}

func gameServerListToChannel(list []*agonesv1.GameServer) chan *agonesv1.GameServer {
	gameServers := make(chan *agonesv1.GameServer)
	go func() {
		defer close(gameServers)

		for _, gs := range list {
			gameServers <- gs
		}
	}()

	return gameServers
}

// parallelize processes a channel of game server objects, invoking the provided callback for items in the channel with the specified degree of parallelism up to a limit.
// Returns nil if all callbacks returned nil or one of the error responses, not necessarily the first one.
func parallelize(gameServers chan *agonesv1.GameServer, parallelism int, work func(gs *agonesv1.GameServer) error) error {
	errch := make(chan error, parallelism)

	var wg sync.WaitGroup

	for i := 0; i < parallelism; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for it := range gameServers {
				err := work(it)
				if err != nil {
					errch <- err
					break
				}
			}
		}()
	}
	wg.Wait()
	close(errch)

	for range gameServers {
		// drain any remaining game servers in the channel, in case we did not consume them all
		continue
	}

	// return first error from the channel, or nil if all successful.
	return <-errch
}

// syncGameServerSetStatus synchronises the GameServerSet State with active GameServer counts
func (c *Controller) syncGameServerSetStatus(ctx context.Context, gsSet *agonesv1.GameServerSet, list []*agonesv1.GameServer) error {
	return c.updateStatusIfChanged(ctx, gsSet, computeStatus(gsSet, list))
}

// updateStatusIfChanged updates GameServerSet status if it's different than provided.
func (c *Controller) updateStatusIfChanged(ctx context.Context, gsSet *agonesv1.GameServerSet, status agonesv1.GameServerSetStatus) error {
	if !cmp.Equal(gsSet.Status, status) {
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Status = status
		_, err := c.gameServerSetGetter.GameServerSets(gsSet.ObjectMeta.Namespace).UpdateStatus(ctx, gsSetCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status on GameServerSet %s", gsSet.ObjectMeta.Name)
		}
	}
	return nil
}

// computeStatus computes the status of the game server set.
func computeStatus(gsSet *agonesv1.GameServerSet, list []*agonesv1.GameServer) agonesv1.GameServerSetStatus {
	var status agonesv1.GameServerSetStatus

	// Initialize list status with empty lists from spec
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		status.Lists = createInitialListStatus(gsSet)
		status.Counters = createInitialCounterStatus(gsSet)
	}
	for _, gs := range list {
		if gs.IsBeingDeleted() {
			// don't count GS that are being deleted
			status.ShutdownReplicas++
			continue
		}

		status.Replicas++
		switch gs.Status.State {
		case agonesv1.GameServerStateReady:
			status.ReadyReplicas++
		case agonesv1.GameServerStateAllocated:
			status.AllocatedReplicas++
		case agonesv1.GameServerStateReserved:
			status.ReservedReplicas++
		}

		// Drop Counters and Lists status if the feature flag has been set to false
		if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			if len(status.Counters) != 0 || len(status.Lists) != 0 {
				status.Counters = map[string]agonesv1.AggregatedCounterStatus{}
				status.Lists = map[string]agonesv1.AggregatedListStatus{}
			}
		}
		// Aggregates all Counters and Lists only for GameServer all states (except IsBeingDeleted)
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			status.Counters = aggregateCounters(status.Counters, gs.Status.Counters, gs.Status.State)
			status.Lists = aggregateLists(status.Lists, gs.Status.Lists, gs.Status.State)
		}
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		// to make this code simpler, while the feature gate is in place,
		// we will loop around the gs list twice.
		status.Players = &agonesv1.AggregatedPlayerStatus{}
		// TODO: integrate this extra loop into the above for loop when PlayerTracking moves to GA
		for _, gs := range list {
			if gs.ObjectMeta.DeletionTimestamp.IsZero() &&
				(gs.Status.State == agonesv1.GameServerStateReady ||
					gs.Status.State == agonesv1.GameServerStateReserved ||
					gs.Status.State == agonesv1.GameServerStateAllocated) {
				if gs.Status.Players != nil {
					status.Players.Capacity += gs.Status.Players.Capacity
					status.Players.Count += gs.Status.Players.Count
				}
			}
		}
	}

	return status
}

func createInitialListStatus(gsSet *agonesv1.GameServerSet) map[string]agonesv1.AggregatedListStatus {
	list := make(map[string]agonesv1.AggregatedListStatus)
	for name := range gsSet.Spec.Template.Spec.Lists {
		list[name] = agonesv1.AggregatedListStatus{}
	}
	return list
}

func createInitialCounterStatus(gsSet *agonesv1.GameServerSet) map[string]agonesv1.AggregatedCounterStatus {
	counters := make(map[string]agonesv1.AggregatedCounterStatus)
	for name := range gsSet.Spec.Template.Spec.Counters {
		counters[name] = agonesv1.AggregatedCounterStatus{}
	}
	return counters
}

// aggregateCounters adds the contents of a CounterStatus map to an AggregatedCounterStatus map.
func aggregateCounters(aggCounterStatus map[string]agonesv1.AggregatedCounterStatus,
	counterStatus map[string]agonesv1.CounterStatus,
	gsState agonesv1.GameServerState) map[string]agonesv1.AggregatedCounterStatus {

	if aggCounterStatus == nil {
		aggCounterStatus = make(map[string]agonesv1.AggregatedCounterStatus)
	}

	for key, val := range counterStatus {
		// If the Counter exists in both maps, aggregate the values.
		if counter, ok := aggCounterStatus[key]; ok {
			// Aggregate for all game server statuses (expected IsBeingDeleted)
			counter.Count = agonesv1.SafeAdd(counter.Count, val.Count)
			counter.Capacity = agonesv1.SafeAdd(counter.Capacity, val.Capacity)

			// Aggregate for Allocated game servers only
			if gsState == agonesv1.GameServerStateAllocated {
				counter.AllocatedCount = agonesv1.SafeAdd(counter.AllocatedCount, val.Count)
				counter.AllocatedCapacity = agonesv1.SafeAdd(counter.AllocatedCapacity, val.Capacity)
			}
			aggCounterStatus[key] = counter
		} else {
			tmp := val.DeepCopy()
			allocatedCount := int64(0)
			allocatedCapacity := int64(0)
			if gsState == agonesv1.GameServerStateAllocated {
				allocatedCount = tmp.Count
				allocatedCapacity = tmp.Capacity
			}
			aggCounterStatus[key] = agonesv1.AggregatedCounterStatus{
				AllocatedCount:    allocatedCount,
				AllocatedCapacity: allocatedCapacity,
				Capacity:          tmp.Capacity,
				Count:             tmp.Count,
			}
		}
	}

	return aggCounterStatus
}

// aggregateLists adds the contents of a ListStatus map to an AggregatedListStatus map.
func aggregateLists(aggListStatus map[string]agonesv1.AggregatedListStatus,
	listStatus map[string]agonesv1.ListStatus,
	gsState agonesv1.GameServerState) map[string]agonesv1.AggregatedListStatus {

	if aggListStatus == nil {
		aggListStatus = make(map[string]agonesv1.AggregatedListStatus)
	}

	for key, val := range listStatus {
		// If the List exists in both maps, aggregate the values.
		if list, ok := aggListStatus[key]; ok {
			list.Capacity += val.Capacity
			// We do include duplicates in the Count.
			list.Count += int64(len(val.Values))
			if gsState == agonesv1.GameServerStateAllocated {
				list.AllocatedCount += int64(len(val.Values))
				list.AllocatedCapacity += val.Capacity
			}
			aggListStatus[key] = list
		} else {
			tmp := val.DeepCopy()
			allocatedCount := int64(0)
			allocatedCapacity := int64(0)
			if gsState == agonesv1.GameServerStateAllocated {
				allocatedCount = int64(len(tmp.Values))
				allocatedCapacity = tmp.Capacity
			}
			aggListStatus[key] = agonesv1.AggregatedListStatus{
				AllocatedCount:    allocatedCount,
				AllocatedCapacity: allocatedCapacity,
				Capacity:          tmp.Capacity,
				Count:             int64(len(tmp.Values)),
			}
		}
	}

	return aggListStatus
}

// newMetrics creates a new gss latency recorder.
func (c *Controller) newMetrics(ctx context.Context) *metrics {
	ctx, err := tag.New(ctx, latencyTags...)
	if err != nil {
		c.baseLogger.WithError(err).Warn("failed to tag latency recorder.")
	}
	return &metrics{
		ctx:              ctx,
		gameServerLister: c.gameServerLister,
		logger:           c.baseLogger,
		start:            time.Now(),
	}
}
