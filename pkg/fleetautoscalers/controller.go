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

package fleetautoscalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/apis/autoscaling"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedagonesv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	typedautoscalingv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/autoscaling/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listeragonesv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	listerautoscalingv1 "agones.dev/agones/pkg/client/listers/autoscaling/v1"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	extism "github.com/extism/go-sdk"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
)

// fasState is the state for each Fleet Autoscaler instance
// expand this structure if extra state is needed for other autoscaler
// strategies.
type fasState struct {
	wasmPlugin *extism.Plugin
	httpClient *http.Client
}

// fasThread is used for tracking each Fleet's autoscaling jobs
type fasThread struct {
	cancel     context.CancelFunc
	state      *fasState
	generation int64
}

// close cancels the context and cleans up any resources
func (ft *fasThread) close(ctx context.Context) {
	ft.cancel()
	if ft.state.wasmPlugin != nil {
		_ = ft.state.wasmPlugin.Close(ctx)
	}
	if ft.state.httpClient != nil {
		ft.state.httpClient.CloseIdleConnections()
	}
}

// Extensions struct contains what is needed to bind webhook handlers
type Extensions struct {
	baseLogger *logrus.Entry
}

// FasLogger helps log and record events related to FleetAutoscaler.
type FasLogger struct {
	fas            *autoscalingv1.FleetAutoscaler
	baseLogger     *logrus.Entry
	recorder       record.EventRecorder
	currChainEntry *autoscalingv1.FleetAutoscalerPolicyType
}

// Controller is the FleetAutoscaler controller
//
//nolint:govet // ignore fieldalignment, singleton
type Controller struct {
	baseLogger            *logrus.Entry
	clock                 clock.WithTickerAndDelayedExecution
	counter               *gameservers.PerNodeCounter
	crdGetter             apiextclientv1.CustomResourceDefinitionInterface
	fasThreads            map[types.UID]fasThread
	fasThreadMutex        sync.Mutex
	fleetGetter           typedagonesv1.FleetsGetter
	fleetLister           listeragonesv1.FleetLister
	fleetSynced           cache.InformerSynced
	fleetAutoscalerGetter typedautoscalingv1.FleetAutoscalersGetter
	fleetAutoscalerLister listerautoscalingv1.FleetAutoscalerLister
	fleetAutoscalerSynced cache.InformerSynced
	workerqueue           *workerqueue.WorkerQueue
	recorder              record.EventRecorder
	gameServerLister      listeragonesv1.GameServerLister
}

// NewController returns a controller for a FleetAutoscaler
func NewController(
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
	counter *gameservers.PerNodeCounter) *Controller {

	autoscaler := agonesInformerFactory.Autoscaling().V1().FleetAutoscalers()
	fleetInformer := agonesInformerFactory.Agones().V1().Fleets()
	gameServers := agonesInformerFactory.Agones().V1().GameServers()

	c := &Controller{
		clock:                 clock.RealClock{},
		counter:               counter,
		crdGetter:             extClient.ApiextensionsV1().CustomResourceDefinitions(),
		fasThreads:            map[types.UID]fasThread{},
		fasThreadMutex:        sync.Mutex{},
		fleetGetter:           agonesClient.AgonesV1(),
		fleetLister:           fleetInformer.Lister(),
		fleetSynced:           fleetInformer.Informer().HasSynced,
		fleetAutoscalerGetter: agonesClient.AutoscalingV1(),
		fleetAutoscalerLister: autoscaler.Lister(),
		fleetAutoscalerSynced: autoscaler.Informer().HasSynced,
		gameServerLister:      gameServers.Lister(),
	}
	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncFleetAutoscaler, c.baseLogger, logfields.FleetAutoscalerKey, autoscaling.GroupName+".FleetAutoscalerController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("fleetautoscaler-workerqueue", c.workerqueue.Healthy)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleetautoscaler-controller"})

	ctx := context.Background()
	_, _ = autoscaler.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.addFasThread(obj.(*autoscalingv1.FleetAutoscaler), true)
		},
		UpdateFunc: func(_, newObj interface{}) {
			c.updateFasThread(ctx, newObj.(*autoscalingv1.FleetAutoscaler))
		},
		DeleteFunc: func(obj interface{}) {
			// Could be a DeletedFinalStateUnknown, in which case, just ignore it
			fas, ok := obj.(*autoscalingv1.FleetAutoscaler)
			if !ok {
				return
			}
			c.deleteFasThread(ctx, fas, true)
		},
	})

	return c
}

// NewExtensions binds the handlers to the webhook outside the initialization of the controller
// initializes a new logger for extensions.
func NewExtensions(wh *webhooks.WebHook) *Extensions {
	ext := &Extensions{}

	ext.baseLogger = runtime.NewLoggerWithType(ext)

	kind := autoscalingv1.Kind("FleetAutoscaler")
	wh.AddHandler("/mutate", kind, admissionv1.Create, ext.mutationHandler)
	wh.AddHandler("/mutate", kind, admissionv1.Update, ext.mutationHandler)
	wh.AddHandler("/validate", kind, admissionv1.Create, ext.validationHandler)
	wh.AddHandler("/validate", kind, admissionv1.Update, ext.validationHandler)

	return ext
}

// Run the FleetAutoscaler controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(ctx context.Context, workers int) error {
	err := crd.WaitForEstablishedCRD(ctx, c.crdGetter, "fleetautoscalers."+autoscaling.GroupName, c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.fleetSynced, c.fleetAutoscalerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	go func() {
		// clean all go routines when ctx is Done
		<-ctx.Done()
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		for _, thread := range c.fasThreads {
			thread.close(ctx)
		}
	}()

	c.workerqueue.Run(ctx, workers)
	return nil
}

func loggerForFleetAutoscalerKey(key string, logger *logrus.Entry) *logrus.Entry {
	return logfields.AugmentLogEntry(logger, logfields.FleetAutoscalerKey, key)
}

func loggerForFleetAutoscaler(fas *autoscalingv1.FleetAutoscaler, logger *logrus.Entry) *logrus.Entry {
	fasName := "NilFleetAutoScaler"
	if fas != nil {
		fasName = fas.Namespace + "/" + fas.Name
	}
	return loggerForFleetAutoscalerKey(fasName, logger).WithField("fas", fas)
}

func (c *Controller) loggerForFleetAutoscalerKey(key string) *logrus.Entry {
	return loggerForFleetAutoscalerKey(key, c.baseLogger)
}

func (c *Controller) loggerForFleetAutoscaler(fas *autoscalingv1.FleetAutoscaler) *logrus.Entry {
	return loggerForFleetAutoscaler(fas, c.baseLogger)
}

// creationMutationHandler is the handler for the mutating webhook that sets
// the default values on the FleetAutoscaler
func (ext *Extensions) mutationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	fas := &autoscalingv1.FleetAutoscaler{}
	err := json.Unmarshal(obj.Raw, fas)
	if err != nil {
		// If the JSON is invalid during mutation, fall through to validation. This allows OpenAPI schema validation
		// to proceed, resulting in a more user friendly error message.
		return review, nil
	}

	fas.ApplyDefaults()

	newFas, err := json.Marshal(fas)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied FleetAutoscaler %s to json", fas.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFas)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for FleetAutoscaler %s", fas.ObjectMeta.Name)
	}

	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for FleetAutoScaler %s", fas.ObjectMeta.Name)
	}

	pt := admissionv1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsonPatch

	return review, nil
}

// validationHandler will intercept when a FleetAutoscaler is created, and
// validate its settings.
func (ext *Extensions) validationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	fas := &autoscalingv1.FleetAutoscaler{}
	err := json.Unmarshal(obj.Raw, fas)
	if err != nil {
		ext.baseLogger.WithField("review", review).WithError(err).Error("validationHandler")
		return review, errors.Wrapf(err, "error unmarshalling FleetAutoscaler json after schema validation: %s", obj.Raw)
	}
	fas.ApplyDefaults()

	if errs := fas.Validate(); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: review.Request.Kind.Group,
			Kind:  review.Request.Kind.Kind,
		}
		statusErr := k8serrors.NewInvalid(kind, review.Request.Name, errs)
		review.Response.Allowed = false
		review.Response.Result = &statusErr.ErrStatus
		loggerForFleetAutoscaler(fas, ext.baseLogger).WithField("review", review).Debug("Invalid FleetAutoscaler")
	}

	return review, nil
}

// syncFleetAutoscaler syncs FleetAutoScale according to different sync type
func (c *Controller) syncFleetAutoscaler(ctx context.Context, key string) error {
	c.loggerForFleetAutoscalerKey(key).Debug("Synchronising")

	fas, err := c.getFleetAutoscalerByKey(key)
	if err != nil {
		return err
	}

	if fas == nil {
		return c.cleanFasThreads(ctx, key)
	}

	// The state is protected by the workerqueue ensuring that we don't process the same queue item concurrently,
	// and we're only going to get here if the fasThread for this FleetAutoscaler exists,
	// so we can safely read the state without a lock over the whole function.
	c.fasThreadMutex.Lock()
	thread, ok := c.fasThreads[fas.ObjectMeta.UID]
	c.fasThreadMutex.Unlock()
	if !ok {
		return errors.New("There should be a fasThread for the FleetAutoscaler, but it was not found")
	}

	// Retrieve the fleet by spec name
	fleet, err := c.fleetLister.Fleets(fas.Namespace).Get(fas.Spec.FleetName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForFleetAutoscaler(fas).Debug("Could not find fleet for autoscaler. Skipping.")

			c.recorder.Eventf(fas, corev1.EventTypeWarning, "FailedGetFleet",
				"could not fetch fleet: %s", fas.Spec.FleetName)

			// don't retry. Pick it up next sync.
			err = nil
		}

		if err := c.updateStatusUnableToScale(ctx, fas); err != nil {
			return err
		}

		return err
	}

	// Don't do anything, the fleet is marked for deletion
	if !fleet.DeletionTimestamp.IsZero() {
		return nil
	}

	fasLog := FasLogger{
		fas:            fas,
		baseLogger:     c.baseLogger,
		recorder:       c.recorder,
		currChainEntry: &fas.Status.LastAppliedPolicy,
	}

	currentReplicas := fleet.Status.Replicas
	gameServerNamespacedLister := c.gameServerLister.GameServers(fleet.ObjectMeta.Namespace)
	desiredReplicas, scalingLimited, err := computeDesiredFleetSize(ctx, thread.state, fas.Spec.Policy, fleet, gameServerNamespacedLister, c.counter.Counts(), &fasLog)

	// If the err is not nil and not an inactive schedule error (ignorable in this case), then record the event
	if err != nil {
		if !errors.Is(err, InactiveScheduleError{}) {
			c.recorder.Eventf(fas, corev1.EventTypeWarning, "FleetAutoscaler",
				"Error calculating desired fleet size on FleetAutoscaler %s. Error: %s", fas.ObjectMeta.Name, err.Error())

			if err := c.updateStatusUnableToScale(ctx, fas); err != nil {
				return err
			}
		}
		return errors.Wrapf(err, "error calculating autoscaling fleet: %s", fleet.ObjectMeta.Name)
	}

	// Log the desired replicas and scaling status
	c.loggerForFleetAutoscalerKey(key).Debugf("Computed desired fleet size: %d, Scaling limited: %v", desiredReplicas, scalingLimited)

	// Scale the fleet to the new size
	if err = c.scaleFleet(ctx, fas, fleet, desiredReplicas); err != nil {
		return errors.Wrapf(err, "error autoscaling fleet %s to %d replicas", fas.Spec.FleetName, desiredReplicas)
	}

	return c.updateStatus(ctx, fas, currentReplicas, desiredReplicas, desiredReplicas != fleet.Spec.Replicas, scalingLimited, *fasLog.currChainEntry)
}

// getFleetAutoscalerByKey gets the Fleet Autoscaler by key
// a nil FleetAutoscaler returned indicates that an attempt to sync should not be retried, e.g.  if the FleetAutoscaler no longer exists.
func (c *Controller) getFleetAutoscalerByKey(key string) (*autoscalingv1.FleetAutoscaler, error) {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForFleetAutoscalerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil, nil
	}
	fas, err := c.fleetAutoscalerLister.FleetAutoscalers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForFleetAutoscalerKey(key).Debug(fmt.Sprintf("FleetAutoscaler %s from namespace %s is no longer available for syncing", name, namespace))
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error retrieving FleetAutoscaler %s from namespace %s", name, namespace)
	}
	return fas, nil
}

// scaleFleet scales the fleet of the autoscaler to a new number of replicas
func (c *Controller) scaleFleet(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, f *agonesv1.Fleet, replicas int32) error {
	if replicas != f.Spec.Replicas {
		fCopy := f.DeepCopy()
		fCopy.Spec.Replicas = replicas
		fCopy, err := c.fleetGetter.Fleets(f.ObjectMeta.Namespace).Update(ctx, fCopy, metav1.UpdateOptions{})
		if err != nil {
			c.recorder.Eventf(fas, corev1.EventTypeWarning, "AutoScalingFleetError",
				"Error on scaling fleet %s from %d to %d. Error: %s", fCopy.ObjectMeta.Name, f.Spec.Replicas, fCopy.Spec.Replicas, err.Error())
			return errors.Wrapf(err, "error updating replicas for fleet %s", f.ObjectMeta.Name)
		}

		c.recorder.Eventf(fas, corev1.EventTypeNormal, "AutoScalingFleet",
			"Scaling fleet %s from %d to %d", fCopy.ObjectMeta.Name, f.Spec.Replicas, fCopy.Spec.Replicas)
	}

	return nil
}

// updateStatus updates the status of the given FleetAutoscaler
func (c *Controller) updateStatus(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, currentReplicas int32, desiredReplicas int32, scaled bool, scalingLimited bool, chainEntry autoscalingv1.FleetAutoscalerPolicyType) error {
	fasCopy := fas.DeepCopy()
	fasCopy.Status.AbleToScale = true
	fasCopy.Status.ScalingLimited = scalingLimited
	fasCopy.Status.CurrentReplicas = currentReplicas
	fasCopy.Status.DesiredReplicas = desiredReplicas

	if fas.Spec.Policy.Type == autoscalingv1.ChainPolicyType {
		fasCopy.Status.LastAppliedPolicy = chainEntry
	} else {
		fasCopy.Status.LastAppliedPolicy = fas.Spec.Policy.Type
	}

	if scaled {
		now := metav1.NewTime(time.Now())
		fasCopy.Status.LastScaleTime = &now
	}

	if !apiequality.Semantic.DeepEqual(fas.Status, fasCopy.Status) {
		if scalingLimited {
			// scalingLimited indicates that the calculated scale would be above or below the range defined by MinReplicas and MaxReplicas
			msg := "Scaling fleet %s was limited to minimum size of %d"
			if currentReplicas > desiredReplicas {
				msg = "Scaling fleet %s was limited to maximum size of %d"
			}

			c.recorder.Eventf(fas, corev1.EventTypeWarning, "ScalingLimited", msg, fas.Spec.FleetName, desiredReplicas)
		}

		_, err := c.fleetAutoscalerGetter.FleetAutoscalers(fas.ObjectMeta.Namespace).UpdateStatus(ctx, fasCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status for fleetautoscaler %s", fas.ObjectMeta.Name)
		}
	}

	return nil
}

// updateStatus updates the status of the given FleetAutoscaler in the case we're not able to scale
func (c *Controller) updateStatusUnableToScale(ctx context.Context, fas *autoscalingv1.FleetAutoscaler) error {
	fasCopy := fas.DeepCopy()
	fasCopy.Status.AbleToScale = false
	fasCopy.Status.ScalingLimited = false
	fasCopy.Status.CurrentReplicas = 0
	fasCopy.Status.DesiredReplicas = 0
	fasCopy.Status.LastAppliedPolicy = autoscalingv1.FleetAutoscalerPolicyType("")

	if !apiequality.Semantic.DeepEqual(fas.Status, fasCopy.Status) {
		_, err := c.fleetAutoscalerGetter.FleetAutoscalers(fas.ObjectMeta.Namespace).UpdateStatus(ctx, fasCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status for fleetautoscaler %s", fas.ObjectMeta.Name)
		}
	}

	return nil
}

// addFasThread creates a ticker that enqueues the FleetAutoscaler for it's configured interval.
// If `lock` is set to true, the function will do appropriate locking for this operation. If set to `false`
// make sure to lock the operation with the c.fasThreadMutex for the execution of this command.
func (c *Controller) addFasThread(fas *autoscalingv1.FleetAutoscaler, lock bool) {
	log := c.loggerForFleetAutoscaler(fas)
	log.WithField("seconds", fas.Spec.Sync.FixedInterval.Seconds).Debug("Thread for Autoscaler created")

	duration := time.Duration(fas.Spec.Sync.FixedInterval.Seconds) * time.Second

	// store against the UID, as there is no guarantee the name is unique over time.
	ctx, cancel := context.WithCancel(context.Background())
	thread := fasThread{
		cancel:     cancel,
		generation: fas.Generation,
		state:      &fasState{},
	}

	if lock {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
	}

	// Seems unlikely that concurrent events could fire at the same time for the same UID,
	// but just in case, let's check.
	if _, ok := c.fasThreads[fas.ObjectMeta.UID]; ok {
		return
	}
	c.fasThreads[fas.ObjectMeta.UID] = thread

	// do immediate enqueue on addition to have an autoscale fire on addition.
	c.workerqueue.Enqueue(fas)
	// Add to queue for each duration period, until cancellation occurs.
	// Workerqueue will handle if multiple attempts are made to add an existing item to the queue, and retries on failure
	// etc.
	go func() {
		wait.Until(func() {
			c.workerqueue.Enqueue(fas)
		}, duration, ctx.Done())
	}()
}

// updateFasThread will replace the queueing thread if the generation has changes on the FleetAutoscaler.
func (c *Controller) updateFasThread(ctx context.Context, fas *autoscalingv1.FleetAutoscaler) {
	c.fasThreadMutex.Lock()
	defer c.fasThreadMutex.Unlock()

	thread, ok := c.fasThreads[fas.ObjectMeta.UID]
	if !ok {
		// maybe the controller crashed and we are only getting update events at this point, so let's add
		// the thread back in
		c.addFasThread(fas, false)
		return
	}

	if fas.Generation != thread.generation {
		c.loggerForFleetAutoscaler(fas).WithField("generation", thread.generation).
			Debug("Fleet autoscaler generation updated, recreating thread")
		c.deleteFasThread(ctx, fas, false)
		c.addFasThread(fas, false)
	}
}

// deleteFasThread removes a FleetAutoScaler sync routine.
// If `lock` is set to true, the function will do appropriate locking for this operation. If set to `false`
// make sure to lock the operation with the c.fasThreadMutex for the execution of this command.
func (c *Controller) deleteFasThread(ctx context.Context, fas *autoscalingv1.FleetAutoscaler, lock bool) {
	c.loggerForFleetAutoscaler(fas).Debug("Thread for Autoscaler removed")

	if lock {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
	}

	if thread, ok := c.fasThreads[fas.ObjectMeta.UID]; ok {
		thread.close(ctx)
		delete(c.fasThreads, fas.ObjectMeta.UID)
	}
}

// cleanFasThreads will delete any fasThread that no longer
// can be tied to a FleetAutoscaler instance.
func (c *Controller) cleanFasThreads(ctx context.Context, key string) error {
	c.baseLogger.WithField("key", key).Debug("Doing full autoscaler thread cleanup")
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return errors.Wrap(err, "attempting to clean all fleet autoscaler threads")
	}

	fasList, err := c.fleetAutoscalerLister.FleetAutoscalers(namespace).List(labels.Everything())
	if err != nil {
		return errors.Wrap(err, "attempting to clean all fleet autoscaler threads")
	}

	c.fasThreadMutex.Lock()
	defer c.fasThreadMutex.Unlock()

	keys := map[types.UID]bool{}
	for k := range c.fasThreads {
		keys[k] = true
	}

	for _, fas := range fasList {
		delete(keys, fas.ObjectMeta.UID)
	}

	// any key that doesn't match to an existing UID, stop it.
	for k := range keys {
		thread := c.fasThreads[k]
		thread.close(ctx)
		delete(c.fasThreads, k)
	}

	return nil
}
