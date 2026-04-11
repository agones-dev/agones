// Copyright 2018 Google LLC All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/cloudproduct"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
)

const (
	sdkserverSidecarName  = "agones-gameserver-sidecar"
	grpcPortEnvVar        = "AGONES_SDK_GRPC_PORT"
	httpPortEnvVar        = "AGONES_SDK_HTTP_PORT"
	passthroughPortEnvVar = "PASSTHROUGH"
)

// Extensions struct contains what is needed to bind webhook handlers
type Extensions struct {
	baseLogger *logrus.Entry
	apiHooks   agonesv1.APIHooks
}

// Controller is a the main GameServer crd controller
//
//nolint:govet // ignore fieldalignment, singleton
type Controller struct {
	baseLogger               *logrus.Entry
	controllerHooks          cloudproduct.ControllerHooksInterface
	sidecarImage             string
	alwaysPullSidecarImage   bool
	sidecarCPURequest        resource.Quantity
	sidecarCPULimit          resource.Quantity
	sidecarMemoryRequest     resource.Quantity
	sidecarMemoryLimit       resource.Quantity
	sidecarRunAsUser         int
	sidecarRequestsRateLimit time.Duration
	sdkServiceAccount        string
	crdGetter                apiextclientv1.CustomResourceDefinitionInterface
	podGetter                typedcorev1.PodsGetter
	podLister                corelisterv1.PodLister
	podSynced                cache.InformerSynced
	gameServerGetter         getterv1.GameServersGetter
	gameServerLister         listerv1.GameServerLister
	gameServerSynced         cache.InformerSynced
	nodeLister               corelisterv1.NodeLister
	nodeSynced               cache.InformerSynced
	portAllocator            portallocator.Interface
	healthController         *HealthController
	migrationController      *MigrationController
	missingPodController     *MissingPodController
	succeededController      *SucceededController
	workerqueue              *workerqueue.WorkerQueue
	creationWorkerQueue      *workerqueue.WorkerQueue // handles creation only
	deletionWorkerQueue      *workerqueue.WorkerQueue // handles deletion only
	recorder                 record.EventRecorder
}

// NewController returns a new gameserver crd controller
func NewController(
	controllerHooks cloudproduct.ControllerHooksInterface,
	health healthcheck.Handler,
	portRanges map[string]portallocator.PortRange,
	sidecarImage string,
	alwaysPullSidecarImage bool,
	sidecarCPURequest resource.Quantity,
	sidecarCPULimit resource.Quantity,
	sidecarMemoryRequest resource.Quantity,
	sidecarMemoryLimit resource.Quantity,
	sidecarRunAsUser int,
	sidecarRequestsRateLimit time.Duration,
	sdkServiceAccount string,
	kubeClient kubernetes.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
) *Controller {

	pods := kubeInformerFactory.Core().V1().Pods()
	gameServers := agonesInformerFactory.Agones().V1().GameServers()
	gsInformer := gameServers.Informer()

	c := &Controller{
		controllerHooks:          controllerHooks,
		sidecarImage:             sidecarImage,
		sidecarCPULimit:          sidecarCPULimit,
		sidecarCPURequest:        sidecarCPURequest,
		sidecarMemoryLimit:       sidecarMemoryLimit,
		sidecarMemoryRequest:     sidecarMemoryRequest,
		sidecarRunAsUser:         sidecarRunAsUser,
		sidecarRequestsRateLimit: sidecarRequestsRateLimit,
		alwaysPullSidecarImage:   alwaysPullSidecarImage,
		sdkServiceAccount:        sdkServiceAccount,
		crdGetter:                extClient.ApiextensionsV1().CustomResourceDefinitions(),
		podGetter:                kubeClient.CoreV1(),
		podLister:                pods.Lister(),
		podSynced:                pods.Informer().HasSynced,
		gameServerGetter:         agonesClient.AgonesV1(),
		gameServerLister:         gameServers.Lister(),
		gameServerSynced:         gsInformer.HasSynced,
		nodeLister:               kubeInformerFactory.Core().V1().Nodes().Lister(),
		nodeSynced:               kubeInformerFactory.Core().V1().Nodes().Informer().HasSynced,
		portAllocator:            controllerHooks.NewPortAllocator(portRanges, kubeInformerFactory, agonesInformerFactory),
		healthController:         NewHealthController(health, kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory, controllerHooks.WaitOnFreePorts()),
		migrationController:      NewMigrationController(health, kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory, controllerHooks.SyncPodPortsToGameServer),
		missingPodController:     NewMissingPodController(health, kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory),
		succeededController:      NewSucceededController(health, kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory),
	}

	c.baseLogger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserver-controller"})

	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServer, c.baseLogger, logfields.GameServerKey, agones.GroupName+".GameServerController", fastRateLimiter())
	c.creationWorkerQueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServer, c.baseLogger.WithField("subqueue", "creation"), logfields.GameServerKey, agones.GroupName+".GameServerControllerCreation", fastRateLimiter())
	c.deletionWorkerQueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncGameServer, c.baseLogger.WithField("subqueue", "deletion"), logfields.GameServerKey, agones.GroupName+".GameServerControllerDeletion", fastRateLimiter())
	health.AddLivenessCheck("gameserver-workerqueue", healthcheck.Check(c.workerqueue.Healthy))
	health.AddLivenessCheck("gameserver-creation-workerqueue", healthcheck.Check(c.creationWorkerQueue.Healthy))
	health.AddLivenessCheck("gameserver-deletion-workerqueue", healthcheck.Check(c.deletionWorkerQueue.Healthy))

	_, _ = gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueGameServerBasedOnState,
		UpdateFunc: func(oldObj, newObj interface{}) {
			// no point in processing unless there is a State change
			oldGs := oldObj.(*agonesv1.GameServer)
			newGs := newObj.(*agonesv1.GameServer)
			if oldGs.Status.State != newGs.Status.State || !newGs.ObjectMeta.DeletionTimestamp.IsZero() {
				c.enqueueGameServerBasedOnState(newGs)
			}
		},
	})

	// track pod deletions, for when GameServers are deleted
	_, _ = pods.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			if isGameServerPod(oldPod) {
				newPod := newObj.(*corev1.Pod)
				//  node name has changed -- i.e. it has been scheduled
				if oldPod.Spec.NodeName != newPod.Spec.NodeName {
					owner := metav1.GetControllerOf(newPod)
					c.workerqueue.Enqueue(cache.ExplicitKey(newPod.ObjectMeta.Namespace + "/" + owner.Name))
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// Could be a DeletedFinalStateUnknown, in which case, just ignore it
			pod, ok := obj.(*corev1.Pod)
			if ok && isGameServerPod(pod) {
				owner := metav1.GetControllerOf(pod)
				c.workerqueue.Enqueue(cache.ExplicitKey(pod.ObjectMeta.Namespace + "/" + owner.Name))
			}
		},
	})

	return c
}

// NewExtensions binds the handlers to the webhook outside the initialization of the controller
// initializes a new logger for extensions.
func NewExtensions(apiHooks agonesv1.APIHooks, wh *webhooks.WebHook) *Extensions {
	ext := &Extensions{apiHooks: apiHooks}

	ext.baseLogger = runtime.NewLoggerWithType(ext)

	wh.AddHandler("/mutate", agonesv1.Kind("GameServer"), admissionv1.Create, ext.creationMutationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("GameServer"), admissionv1.Create, ext.creationValidationHandler)
	wh.AddHandler("/mutate", corev1.SchemeGroupVersion.WithKind("Pod").GroupKind(), admissionv1.Create, ext.creationMutationHandlerPod)

	return ext
}

func (c *Controller) enqueueGameServerBasedOnState(item interface{}) {
	gs := item.(*agonesv1.GameServer)

	switch gs.Status.State {
	case agonesv1.GameServerStatePortAllocation,
		agonesv1.GameServerStateCreating:
		c.creationWorkerQueue.Enqueue(gs)

	case agonesv1.GameServerStateShutdown:
		c.deletionWorkerQueue.Enqueue(gs)

	default:
		c.workerqueue.Enqueue(gs)
	}
}

// fastRateLimiter returns a fast rate limiter, without exponential back-off.
func fastRateLimiter() workqueue.TypedRateLimiter[any] {
	const numFastRetries = 5
	const fastDelay = 20 * time.Millisecond  // first few retries up to 'numFastRetries' are fast
	const slowDelay = 500 * time.Millisecond // subsequent retries are slow

	return workqueue.NewTypedItemFastSlowRateLimiter[any](fastDelay, slowDelay, numFastRetries)
}

// creationMutationHandler is the handler for the mutating webhook that sets the
// the default values on the GameServer
// Should only be called on gameserver create operations.
// nolint:dupl
func (ext *Extensions) creationMutationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	gs := &agonesv1.GameServer{}
	err := json.Unmarshal(obj.Raw, gs)
	if err != nil {
		// If the JSON is invalid during mutation, fall through to validation. This allows OpenAPI schema validation
		// to proceed, resulting in a more user friendly error message.
		return review, nil
	}

	// This is the main logic of this function
	// the rest is really just json plumbing
	gs.ApplyDefaults()

	newGS, err := json.Marshal(gs)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied GameServer %s to json", gs.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newGS)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for GameServer %s", gs.ObjectMeta.Name)
	}

	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for GameServer %s", gs.ObjectMeta.Name)
	}

	pt := admissionv1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsonPatch

	return review, nil
}

func loggerForGameServerKey(key string, logger *logrus.Entry) *logrus.Entry {
	return logfields.AugmentLogEntry(logger, logfields.GameServerKey, key)
}

func loggerForGameServer(gs *agonesv1.GameServer, logger *logrus.Entry) *logrus.Entry {
	gsName := logfields.NilGameServer
	if gs != nil {
		gsName = gs.Namespace + "/" + gs.Name
	}
	return loggerForGameServerKey(gsName, logger).WithField("gs", gs)
}

// creationValidationHandler that validates a GameServer when it is created
// Should only be called on gameserver create operations.
func (ext *Extensions) creationValidationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	gs := &agonesv1.GameServer{}
	err := json.Unmarshal(obj.Raw, gs)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling GameServer json after schema validation: %s", obj.Raw)
	}

	loggerForGameServer(gs, ext.baseLogger).WithField("review", review).Debug("creationValidationHandler")

	if errs := gs.Validate(ext.apiHooks); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: review.Request.Kind.Group,
			Kind:  review.Request.Kind.Kind,
		}
		statusErr := k8serrors.NewInvalid(kind, review.Request.Name, errs)
		review.Response.Allowed = false
		review.Response.Result = &statusErr.ErrStatus
		loggerForGameServer(gs, ext.baseLogger).WithField("review", review).Debug("Invalid GameServer")
	}
	return review, nil
}

// creationMutationHandlerPod that mutates a GameServer pod when it is created
// Should only be called on gameserver pod create operations.
func (ext *Extensions) creationMutationHandlerPod(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	obj := review.Request.Object
	pod := &corev1.Pod{}
	err := json.Unmarshal(obj.Raw, pod)
	if err != nil {
		// If the JSON is invalid during mutation, fall through to validation. This allows OpenAPI schema validation
		// to proceed, resulting in a more user friendly error message.
		return review, nil
	}

	ext.baseLogger.WithField("pod.Name", pod.Name).Debug("creationMutationHandlerPod")

	annotation, ok := pod.ObjectMeta.Annotations[agonesv1.PassthroughPortAssignmentAnnotation]
	if !ok {
		ext.baseLogger.WithField("pod.Name", pod.Name).Info("the agones.dev/container-passthrough-port-assignment annotation is empty and it's unexpected")
		return review, nil
	}

	passthroughPortAssignmentMap := make(map[string][]int)
	if err := json.Unmarshal([]byte(annotation), &passthroughPortAssignmentMap); err != nil {
		return review, errors.Wrapf(err, "could not unmarshal annotation %v (value %q)", passthroughPortAssignmentMap, annotation)
	}

	for _, container := range pod.Spec.InitContainers {
		if container.RestartPolicy == nil || *container.RestartPolicy != corev1.ContainerRestartPolicyAlways {
			continue
		}
		for _, portIdx := range passthroughPortAssignmentMap[container.Name] {
			container.Ports[portIdx].ContainerPort = container.Ports[portIdx].HostPort
		}
	}
	for _, container := range pod.Spec.Containers {
		for _, portIdx := range passthroughPortAssignmentMap[container.Name] {
			container.Ports[portIdx].ContainerPort = container.Ports[portIdx].HostPort
		}
	}

	newPod, err := json.Marshal(pod)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling changes applied Pod %s to json", pod.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newPod)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for Pod %s", pod.ObjectMeta.Name)
	}

	jsonPatch, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for Pod %s", pod.ObjectMeta.Name)
	}

	pt := admissionv1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsonPatch

	return review, nil
}

// Run the GameServer controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(ctx context.Context, workers int) error {
	err := crd.WaitForEstablishedCRD(ctx, c.crdGetter, "gameservers.agones.dev", c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.podSynced, c.nodeSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// Run the Port Allocator
	if err = c.portAllocator.Run(ctx); err != nil {
		return errors.Wrap(err, "error running the port allocator")
	}

	// Run the Health Controller
	go func() {
		if err := c.healthController.Run(ctx, workers); err != nil {
			c.baseLogger.WithError(err).Error("error running health controller")
		}
	}()

	// Run the Migration Controller
	go func() {
		if err := c.migrationController.Run(ctx, workers); err != nil {
			c.baseLogger.WithError(err).Error("error running migration controller")
		}
	}()

	// Run the Missing Pod Controller
	go func() {
		if err := c.missingPodController.Run(ctx, workers); err != nil {
			c.baseLogger.WithError(err).Error("error running missing pod controller")
		}
	}()

	// Run the Succeeded Controller
	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		go func() {
			if err := c.succeededController.Run(ctx, workers); err != nil {
				c.baseLogger.WithError(err).Error("error running succeeded controller")
			}
		}()
	}

	// start work queues
	var wg sync.WaitGroup

	startWorkQueue := func(wq *workerqueue.WorkerQueue) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wq.Run(ctx, workers)
		}()
	}

	startWorkQueue(c.workerqueue)
	startWorkQueue(c.creationWorkerQueue)
	startWorkQueue(c.deletionWorkerQueue)
	wg.Wait()
	return nil
}

// syncGameServer synchronises the Pods for the GameServers.
// and reacts to status changes that can occur through the client SDK
func (c *Controller) syncGameServer(ctx context.Context, key string) error {
	loggerForGameServerKey(key, c.baseLogger).Debug("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(loggerForGameServerKey(key, c.baseLogger), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	gs, err := c.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			loggerForGameServerKey(key, c.baseLogger).Debug("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	if gs, err = c.syncGameServerDeletionTimestamp(ctx, gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerPortAllocationState(ctx, gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerCreatingState(ctx, gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerStartingState(ctx, gs); err != nil {
		return err
	}
	if gs, err = c.syncGameServerRequestReadyState(ctx, gs); err != nil {
		return err
	}
	if gs, err = c.syncDevelopmentGameServer(ctx, gs); err != nil {
		return err
	}
	if err := c.syncGameServerShutdownState(ctx, gs); err != nil {
		return err
	}

	return nil
}

// syncGameServerDeletionTimestamp if the deletion timestamp is non-zero
// then do one of two things:
// - if the GameServer has Pods running, delete them
// - if there no pods, remove the finalizer
func (c *Controller) syncGameServerDeletionTimestamp(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	if gs.ObjectMeta.DeletionTimestamp.IsZero() {
		return gs, nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("Syncing with Deletion Timestamp")

	pod, err := c.gameServerPod(gs)
	if err != nil && !k8serrors.IsNotFound(err) {
		return gs, err
	}

	_, isDev := gs.GetDevAddress()
	if pod != nil && !isDev {
		// only need to do this once
		if pod.ObjectMeta.DeletionTimestamp.IsZero() {
			err = c.podGetter.Pods(pod.ObjectMeta.Namespace).Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{})
			if err != nil {
				return gs, errors.Wrapf(err, "error deleting pod for GameServer. Name: %s, Namespace: %s", gs.ObjectMeta.Name, pod.ObjectMeta.Namespace)
			}
			c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), fmt.Sprintf("Deleting Pod %s", pod.ObjectMeta.Name))
		}

		// but no removing finalizers until it's truly gone
		return gs, nil
	}

	gsCopy := gs.DeepCopy()
	// remove the finalizer for this controller
	var fin []string
	for _, f := range gsCopy.ObjectMeta.Finalizers {
		if f != agones.GroupName && f != agonesv1.FinalizerName {
			fin = append(fin, f)
		}
	}
	gsCopy.ObjectMeta.Finalizers = fin
	loggerForGameServer(gsCopy, c.baseLogger).Debugf("No pods found, removing finalizer %s", agonesv1.FinalizerName)
	gs, err = c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return gs, errors.Wrapf(err, "error removing finalizer for GameServer %s", gsCopy.ObjectMeta.Name)
}

// syncGameServerPortAllocationState gives a port to a dynamically allocating GameServer
func (c *Controller) syncGameServerPortAllocationState(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	if !(gs.Status.State == agonesv1.GameServerStatePortAllocation && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}

	gsCopy := c.portAllocator.Allocate(gs.DeepCopy())

	gsCopy.Status.State = agonesv1.GameServerStateCreating
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Port allocated")

	loggerForGameServer(gsCopy, c.baseLogger).Debug("Syncing Port Allocation GameServerState")
	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		// if the GameServer doesn't get updated with the port data, then put the port
		// back in the pool, as it will get retried on the next pass
		c.portAllocator.DeAllocate(gsCopy)
		return gs, errors.Wrapf(err, "error updating GameServer %s to default values", gs.Name)
	}

	return gs, nil
}

// syncGameServerCreatingState checks if the GameServer is in the Creating state, and if so
// creates a Pod for the GameServer and moves the state to Starting
func (c *Controller) syncGameServerCreatingState(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	if !(gs.Status.State == agonesv1.GameServerStateCreating && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}
	if _, isDev := gs.GetDevAddress(); isDev {
		return gs, nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("Syncing Create State")

	// Maybe something went wrong, and the pod was created, but the state was never moved to Starting, so let's check
	_, err := c.gameServerPod(gs)
	if k8serrors.IsNotFound(err) {

		for i := range gs.Spec.Ports {
			if gs.Spec.Ports[i].PortPolicy == agonesv1.Static && gs.Spec.Ports[i].Protocol == agonesv1.ProtocolTCPUDP {
				name := gs.Spec.Ports[i].Name
				gs.Spec.Ports[i].Name = name + "-tcp"
				gs.Spec.Ports[i].Protocol = corev1.ProtocolTCP

				// Add separate UDP port configuration
				gs.Spec.Ports = append(gs.Spec.Ports, agonesv1.GameServerPort{
					PortPolicy:    agonesv1.Static,
					Name:          name + "-udp",
					ContainerPort: gs.Spec.Ports[i].ContainerPort,
					HostPort:      gs.Spec.Ports[i].HostPort,
					Protocol:      corev1.ProtocolUDP,
					Container:     gs.Spec.Ports[i].Container,
				})
			}
		}
		gs, err = c.createGameServerPod(ctx, gs)
		if err != nil || gs.Status.State == agonesv1.GameServerStateError {
			return gs, err
		}
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateStarting
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to Starting state", gs.Name)
	}
	return gs, nil
}

// syncDevelopmentGameServer manages advances a development gameserver to Ready status and registers its address and ports.
func (c *Controller) syncDevelopmentGameServer(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	// do not sync if the server is deleting.
	if !(gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}
	// Get the development IP address
	devIPAddress, isDevGs := gs.GetDevAddress()
	if !isDevGs {
		return gs, nil
	}

	// Only move from Creating -> Ready or RequestReady -> Ready.
	// Shutdown -> Delete will still be handled normally by syncGameServerShutdownState.
	// Other manual state changes are up to the end user.
	if gs.Status.State != agonesv1.GameServerStateCreating && gs.Status.State != agonesv1.GameServerStateRequestReady {
		return gs, nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("GS is a development game server and will not be managed by Agones.")
	gsCopy := gs.DeepCopy()

	gsCopy.Status.State = agonesv1.GameServerStateReady

	if gs.Status.State == agonesv1.GameServerStateCreating {
		var ports []agonesv1.GameServerStatusPort
		for _, p := range gs.Spec.Ports {
			ports = append(ports, p.Status())
		}

		gsCopy.Status.Ports = ports
		gsCopy.Status.Address = devIPAddress
		gsCopy.Status.Addresses = []corev1.NodeAddress{{Address: devIPAddress, Type: "InternalIP"}}
		gsCopy.Status.NodeName = devIPAddress
	}

	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to %v status", gs.Name, gs.Status)
	}
	return gs, nil
}

// createGameServerPod creates the backing Pod for a given GameServer
func (c *Controller) createGameServerPod(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	sidecar := c.sidecar(gs)
	pod, err := gs.Pod(c.controllerHooks, sidecar)
	if err != nil {
		// this shouldn't happen, but if it does.
		loggerForGameServer(gs, c.baseLogger).WithError(err).Error("error creating pod from Game Server")
		gs, err = c.moveToErrorState(ctx, gs, err.Error())
		return gs, err
	}

	// if the service account is not set, then you are in the "opinionated"
	// mode. If the user sets the service account, we assume they know what they are
	// doing, and don't disable the gameserver container.
	if pod.Spec.ServiceAccountName == "" {
		pod.Spec.ServiceAccountName = c.sdkServiceAccount
		err = gs.DisableServiceAccount(pod)
		if err != nil {
			return gs, err
		}
	}

	err = c.addGameServerHealthCheck(gs, pod)
	if err != nil {
		return gs, err
	}
	c.addSDKServerEnvVars(gs, pod)

	loggerForGameServer(gs, c.baseLogger).WithField("pod", pod).Debug("Creating Pod for GameServer")
	pod, err = c.podGetter.Pods(gs.ObjectMeta.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		switch {
		case k8serrors.IsAlreadyExists(err):
			c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Pod already exists, reused")
			return gs, nil
		case k8serrors.IsInvalid(err):
			loggerForGameServer(gs, c.baseLogger).WithField("pod", pod).Errorf("Pod created is invalid")
			gs, err = c.moveToErrorState(ctx, gs, err.Error())
			return gs, err
		case k8serrors.IsForbidden(err):
			loggerForGameServer(gs, c.baseLogger).WithField("pod", pod).Errorf("Pod created is forbidden")
			gs, err = c.moveToErrorState(ctx, gs, err.Error())
			return gs, err
		default:
			c.recorder.Eventf(gs, corev1.EventTypeWarning, string(gs.Status.State), "error creating Pod for GameServer %s", gs.Name)
			return gs, errors.Wrapf(err, "error creating Pod for GameServer %s", gs.Name)
		}
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State),
		fmt.Sprintf("Pod %s created", pod.ObjectMeta.Name))

	return gs, nil
}

// sidecar creates the sidecar container for a given GameServer
func (c *Controller) sidecar(gs *agonesv1.GameServer) corev1.Container {
	sidecar := corev1.Container{
		Name:  sdkserverSidecarName,
		Image: c.sidecarImage,
		Env: []corev1.EnvVar{
			{
				Name:  "GAMESERVER_NAME",
				Value: gs.ObjectMeta.Name,
			},
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name:  "FEATURE_GATES",
				Value: runtime.EncodeFeatures(),
			},
			{
				Name:  "LOG_LEVEL",
				Value: string(gs.Spec.SdkServer.LogLevel),
			},
			{
				Name:  "REQUESTS_RATE_LIMIT",
				Value: c.sidecarRequestsRateLimit.String(),
			},
		},
		Resources: corev1.ResourceRequirements{},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8080),
				},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       3,
		},
	}

	if gs.Spec.SdkServer.GRPCPort != 0 {
		sidecar.Args = append(sidecar.Args, fmt.Sprintf("--grpc-port=%d", gs.Spec.SdkServer.GRPCPort))
	}

	if gs.Spec.SdkServer.HTTPPort != 0 {
		sidecar.Args = append(sidecar.Args, fmt.Sprintf("--http-port=%d", gs.Spec.SdkServer.HTTPPort))
	}

	requests := corev1.ResourceList{}
	if !c.sidecarCPURequest.IsZero() {
		requests[corev1.ResourceCPU] = c.sidecarCPURequest
	}
	if !c.sidecarMemoryRequest.IsZero() {
		requests[corev1.ResourceMemory] = c.sidecarMemoryRequest
	}
	sidecar.Resources.Requests = requests

	limits := corev1.ResourceList{}
	if !c.sidecarCPULimit.IsZero() {
		limits[corev1.ResourceCPU] = c.sidecarCPULimit
	}
	if !c.sidecarMemoryLimit.IsZero() {
		limits[corev1.ResourceMemory] = c.sidecarMemoryLimit
	}
	sidecar.Resources.Limits = limits

	if c.alwaysPullSidecarImage {
		sidecar.ImagePullPolicy = corev1.PullAlways
	}

	sidecar.SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To(int64(c.sidecarRunAsUser)),
	}

	return sidecar
}

// addGameServerHealthCheck adds the http health check to the GameServer container
func (c *Controller) addGameServerHealthCheck(gs *agonesv1.GameServer, pod *corev1.Pod) error {
	if gs.Spec.Health.Disabled {
		return nil
	}

	return gs.ApplyToPodContainer(pod, gs.Spec.Container, func(c corev1.Container) corev1.Container {
		if c.LivenessProbe == nil {
			c.LivenessProbe = &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/gshealthz",
						Port: intstr.FromInt(8080),
					},
				},
				// The sidecar relies on kubelet to delay by InitialDelaySeconds after the
				// container is started (after image pull, etc).
				InitialDelaySeconds: gs.Spec.Health.InitialDelaySeconds,
				PeriodSeconds:       gs.Spec.Health.PeriodSeconds,

				// By the time /gshealthz returns unhealthy, the sidecar has already evaluated
				// {FailureThreshold in a row} failed health checks, so in theory on the kubelet
				// side, one failure is sufficient to know the game server is unhealthy. However,
				// with only one failure, if the sidecar doesn't come up at all, we unnecessarily
				// restart the game server. So use FailureThreshold as startup wiggle-room as well.
				//
				// Note that in general, FailureThreshold could also be infinite - the controller
				// and sidecar are responsible for health management.
				FailureThreshold: gs.Spec.Health.FailureThreshold,
			}
		}

		return c
	})
}

func (c *Controller) addSDKServerEnvVars(gs *agonesv1.GameServer, pod *corev1.Pod) {
	sdkEnvVars := sdkEnvironmentVariables(gs)
	if sdkEnvVars == nil {
		// If a gameserver was created before 1.1 when we started defaulting the grpc and http ports,
		// don't change any container spec.
		return
	}

	for i := range pod.Spec.InitContainers {
		c := &pod.Spec.InitContainers[i]
		if c.Name != sdkserverSidecarName {
			addEnvVarsToContainer(c, sdkEnvVars)
			pod.Spec.InitContainers[i] = *c
		}
	}

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		if c.Name != sdkserverSidecarName {
			addEnvVarsToContainer(c, sdkEnvVars)
			pod.Spec.Containers[i] = *c
		}
	}
}

func addEnvVarsToContainer(c *corev1.Container, sdkEnvVars []corev1.EnvVar) {
	// Filter out environment variables that have reserved names.
	// From https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	env := c.Env[:0]
	for _, e := range c.Env {
		if !reservedEnvironmentVariableName(e.Name) {
			env = append(env, e)
		}
	}
	env = append(env, sdkEnvVars...)
	c.Env = env
}

func reservedEnvironmentVariableName(name string) bool {
	return name == grpcPortEnvVar || name == httpPortEnvVar
}

func sdkEnvironmentVariables(gs *agonesv1.GameServer) []corev1.EnvVar {
	var env []corev1.EnvVar
	if gs.Spec.SdkServer.GRPCPort != 0 {
		env = append(env, corev1.EnvVar{
			Name:  grpcPortEnvVar,
			Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort)),
		})
	}
	if gs.Spec.SdkServer.HTTPPort != 0 {
		env = append(env, corev1.EnvVar{
			Name:  httpPortEnvVar,
			Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort)),
		})
	}
	return env
}

// syncGameServerStartingState looks for a pod that has been scheduled for this GameServer
// and then sets the Status > Address and Ports values.
func (c *Controller) syncGameServerStartingState(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	if !(gs.Status.State == agonesv1.GameServerStateStarting && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return gs, nil
	}
	if _, isDev := gs.GetDevAddress(); isDev {
		return gs, nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("Syncing Starting GameServerState")

	// there should be a pod (although it may not have a scheduled container),
	// so if there is an error of any kind, then move this to queue backoff
	pod, err := c.gameServerPod(gs)
	if err != nil {
		// expected to happen, so don't log it.
		if k8serrors.IsNotFound(err) {
			return nil, workerqueue.NewTraceError(err)
		}

		// do log if it's something other than NotFound, since that's weird.
		return nil, err
	}
	if pod.Spec.NodeName == "" {
		return gs, workerqueue.NewTraceError(errors.Errorf("node not yet populated for Pod %s", pod.ObjectMeta.Name))
	}

	// Ensure the pod IPs are populated
	if len(pod.Status.PodIPs) == 0 {
		return gs, workerqueue.NewTraceError(errors.Errorf("pod IPs not yet populated for Pod %s", pod.ObjectMeta.Name))
	}

	node, err := c.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		return gs, errors.Wrapf(err, "error retrieving node %s for Pod %s", pod.Spec.NodeName, pod.ObjectMeta.Name)
	}
	gsCopy := gs.DeepCopy()
	gsCopy, err = applyGameServerAddressAndPort(gsCopy, node, pod, c.controllerHooks.SyncPodPortsToGameServer)
	if err != nil {
		return gs, err
	}

	gsCopy.Status.State = agonesv1.GameServerStateScheduled
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return gs, errors.Wrapf(err, "error updating GameServer %s to Scheduled state", gs.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Address and port populated")

	return gs, nil
}

// syncGameServerRequestReadyState checks if the Game Server is Requesting to be ready,
// and then adds the IP and Port information to the Status and marks the GameServer
// as Ready
func (c *Controller) syncGameServerRequestReadyState(ctx context.Context, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	if !(gs.Status.State == agonesv1.GameServerStateRequestReady && gs.ObjectMeta.DeletionTimestamp.IsZero()) ||
		gs.Status.State == agonesv1.GameServerStateUnhealthy {
		return gs, nil
	}
	if _, isDev := gs.GetDevAddress(); isDev {
		return gs, nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("Syncing RequestReady State")

	gsCopy := gs.DeepCopy()

	pod, err := c.gameServerPod(gs)
	// NotFound should never happen, and if it does -- something bad happened,
	// so go into workerqueue backoff.
	if err != nil {
		return nil, err
	}

	// if the address hasn't been populated, and the Ready request comes
	// before the controller has had a chance to do it, then
	// do it here instead
	addressPopulated := false
	if gs.Status.NodeName == "" {
		addressPopulated = true
		if pod.Spec.NodeName == "" {
			return gs, workerqueue.NewTraceError(errors.Errorf("node not yet populated for Pod %s", pod.ObjectMeta.Name))
		}
		node, err := c.nodeLister.Get(pod.Spec.NodeName)
		if err != nil {
			return gs, errors.Wrapf(err, "error retrieving node %s for Pod %s", pod.Spec.NodeName, pod.ObjectMeta.Name)
		}
		gsCopy, err = applyGameServerAddressAndPort(gsCopy, node, pod, c.controllerHooks.SyncPodPortsToGameServer)
		if err != nil {
			return gs, err
		}
	}

	gsCopy, err = c.applyGameServerReadyContainerIDAnnotation(ctx, gsCopy, pod)
	if err != nil {
		return gs, err
	}

	gsCopy.Status.State = agonesv1.GameServerStateReady
	gs, err = c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return gs, errors.Wrapf(err, "error setting Ready, Port and address on GameServer %s Status", gs.ObjectMeta.Name)
	}

	if addressPopulated {
		c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Address and port populated")
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "SDK.Ready() complete")
	return gs, nil
}

// applyGameServerReadyContainerIDAnnotation updates the GameServer and its corresponding Pod with an annotation
// indicating the ID of the container that is running the game server, once it's in a Running state.
func (c *Controller) applyGameServerReadyContainerIDAnnotation(ctx context.Context, gsCopy *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, error) {
	// if there is a sidecar container, we escape right away. On move to stable, this method can be deleted.
	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		return gsCopy, nil
	}

	// track the ready gameserver container, so we can determine that after this point, we should move to Unhealthy
	// if there is a container crash/restart after we move to Ready
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == gsCopy.Spec.Container {
			if _, ok := gsCopy.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation]; !ok {
				// check to make sure this container is actually running. If there was a recent crash, the cache may
				// not yet have the newer, running container.
				if cs.State.Running == nil {
					return nil, workerqueue.NewTraceError(fmt.Errorf("game server container for GameServer %s in namespace %s is not currently running, try again", gsCopy.ObjectMeta.Name, gsCopy.ObjectMeta.Namespace))
				}
				gsCopy.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = cs.ContainerID
			}
			break
		}
	}
	// Verify that we found the game server container - we may have a stale cache where pod is missing ContainerStatuses.
	if _, ok := gsCopy.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation]; !ok {
		return nil, workerqueue.NewTraceError(fmt.Errorf("game server container for GameServer %s in namespace %s not present in pod status, try again", gsCopy.ObjectMeta.Name, gsCopy.ObjectMeta.Namespace))
	}

	// Also update the pod with the same annotation, so we can check if the Pod data is up-to-date, now and also in the HealthController.
	// But if it is already set, then ignore it, since we only need to do this one time.
	if _, ok := pod.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation]; !ok {
		podCopy := pod.DeepCopy()
		if podCopy.ObjectMeta.Annotations == nil {
			podCopy.ObjectMeta.Annotations = map[string]string{}
		}

		podCopy.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = gsCopy.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation]
		if _, err := c.podGetter.Pods(pod.ObjectMeta.Namespace).Update(ctx, podCopy, metav1.UpdateOptions{}); err != nil {
			return nil, errors.Wrapf(err, "error updating ready annotation on Pod: %s", pod.ObjectMeta.Name)
		}
	}
	return gsCopy, nil
}

// syncGameServerShutdownState deletes the GameServer (and therefore the backing Pod) if it is in shutdown state
func (c *Controller) syncGameServerShutdownState(ctx context.Context, gs *agonesv1.GameServer) error {
	if !(gs.Status.State == agonesv1.GameServerStateShutdown && gs.ObjectMeta.DeletionTimestamp.IsZero()) {
		return nil
	}

	loggerForGameServer(gs, c.baseLogger).Debug("Syncing Shutdown State")
	// be explicit about where to delete.
	p := metav1.DeletePropagationBackground
	err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{PropagationPolicy: &p})
	if err != nil {
		return errors.Wrapf(err, "error deleting Game Server %s", gs.ObjectMeta.Name)
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Deletion started")
	return nil
}

// moveToErrorState moves the GameServer to the error state
func (c *Controller) moveToErrorState(ctx context.Context, gs *agonesv1.GameServer, msg string) (*agonesv1.GameServer, error) {
	gsCopy := gs.DeepCopy()
	if gsCopy.Annotations == nil {
		gsCopy.Annotations = make(map[string]string, 1)
	}
	gsCopy.Annotations[agonesv1.GameServerErroredAtAnnotation] = time.Now().Format(time.RFC3339)
	gsCopy.Status.State = agonesv1.GameServerStateError

	gs, err := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return gs, errors.Wrapf(err, "error moving GameServer %s to Error State", gs.ObjectMeta.Name)
	}

	c.recorder.Event(gs, corev1.EventTypeWarning, string(gs.Status.State), msg)
	return gs, nil
}

// gameServerPod returns the Pod for this Game Server, or an error if there are none,
// or it cannot be determined (there are more than one, which should not happen)
func (c *Controller) gameServerPod(gs *agonesv1.GameServer) (*corev1.Pod, error) {
	// If the game server is a dev server we do not create a pod for it, return an empty pod.
	if _, isDev := gs.GetDevAddress(); isDev {
		return &corev1.Pod{}, nil
	}

	pod, err := c.podLister.Pods(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name)

	// if not found, propagate this error up, so we can use it in checks
	if k8serrors.IsNotFound(err) {
		return nil, err
	}

	if !metav1.IsControlledBy(pod, gs) {
		return nil, k8serrors.NewNotFound(corev1.Resource("pod"), gs.ObjectMeta.Name)
	}

	return pod, errors.Wrapf(err, "error retrieving pod for GameServer %s", gs.ObjectMeta.Name)
}
