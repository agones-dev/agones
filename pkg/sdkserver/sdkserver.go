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

package sdkserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/mennanov/fmutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	k8sv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	typedv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listersv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
)

// Operation is a synchronisation action
type Operation string

var (
	// GameServerListMaxCapacity is the maximum capacity for List in the gamerserver spec and status CRDs.
	GameServerListMaxCapacity int64
)

const (
	updateState            Operation     = "updateState"
	updateLabel            Operation     = "updateLabel"
	updateAnnotation       Operation     = "updateAnnotation"
	updatePlayerCapacity   Operation     = "updatePlayerCapacity"
	updateConnectedPlayers Operation     = "updateConnectedPlayers"
	updateCounters         Operation     = "updateCounters"
	updateLists            Operation     = "updateLists"
	updatePeriod           time.Duration = time.Second
)

var (
	_ sdk.SDKServer   = &SDKServer{}
	_ alpha.SDKServer = &SDKServer{}
	_ beta.SDKServer  = &SDKServer{}
)

type counterUpdateRequest struct {
	// Capacity of the Counter as set by capacitySet.
	capacitySet *int64
	// Count of the Counter as set by countSet.
	countSet *int64
	// Tracks the sum of CountIncrement, CountDecrement, and/or CountSet requests from the client SDK.
	diff int64
	// Counter as retreived from the GameServer
	counter agonesv1.CounterStatus
}

type listUpdateRequest struct {
	// Capacity of the List as set by capacitySet.
	capacitySet *int64
	// String keys are the Values to remove from the List
	valuesToDelete map[string]bool
	// Values to add to the List
	valuesToAppend []string
}

// SDKServer is a gRPC server, that is meant to be a sidecar
// for a GameServer that will update the game server status on SDK requests
//
//nolint:govet // ignore fieldalignment, singleton
type SDKServer struct {
	logger              *logrus.Entry
	gameServerName      string
	namespace           string
	informerFactory     externalversions.SharedInformerFactory
	gameServerGetter    typedv1.GameServersGetter
	gameServerLister    listersv1.GameServerLister
	gameServerSynced    cache.InformerSynced
	connected           bool
	server              *http.Server
	clock               clock.Clock
	health              agonesv1.Health
	healthTimeout       time.Duration
	healthMutex         sync.RWMutex
	healthLastUpdated   time.Time
	healthFailureCount  int32
	healthChecksRunning sync.Once
	workerqueue         *workerqueue.WorkerQueue
	streamMutex         sync.RWMutex
	connectedStreams    []sdk.SDK_WatchGameServerServer
	ctx                 context.Context
	recorder            record.EventRecorder
	gsLabels            map[string]string
	gsAnnotations       map[string]string
	gsState             agonesv1.GameServerState
	gsStateChannel      chan agonesv1.GameServerState
	gsUpdateMutex       sync.RWMutex
	gsWaitForSync       sync.WaitGroup
	reserveTimer        *time.Timer
	gsReserveDuration   *time.Duration
	gsPlayerCapacity    int64
	gsConnectedPlayers  []string
	gsCounterUpdates    map[string]counterUpdateRequest
	gsListUpdates       map[string]listUpdateRequest
	gsCopy              *agonesv1.GameServer
}

// NewSDKServer creates a SDKServer that sets up an
// InClusterConfig for Kubernetes
func NewSDKServer(gameServerName, namespace string, kubeClient kubernetes.Interface,
	agonesClient versioned.Interface, logLevel logrus.Level, healthPort int, requestsRateLimit time.Duration) (*SDKServer, error) {
	mux := http.NewServeMux()
	resync := 0 * time.Second

	// limit the informer to only working with the gameserver that the sdk is attached to
	tweakListOptions := func(opts *metav1.ListOptions) {
		s1 := fields.OneTermEqualSelector("metadata.name", gameServerName)
		opts.FieldSelector = s1.String()
	}
	factory := externalversions.NewSharedInformerFactoryWithOptions(agonesClient, resync, externalversions.WithNamespace(namespace), externalversions.WithTweakListOptions(tweakListOptions))
	gameServers := factory.Agones().V1().GameServers()

	s := &SDKServer{
		gameServerName:   gameServerName,
		namespace:        namespace,
		gameServerGetter: agonesClient.AgonesV1(),
		gameServerLister: gameServers.Lister(),
		gameServerSynced: gameServers.Informer().HasSynced,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", healthPort),
			Handler: mux,
		},
		clock:              clock.RealClock{},
		healthMutex:        sync.RWMutex{},
		healthFailureCount: 0,
		streamMutex:        sync.RWMutex{},
		gsLabels:           map[string]string{},
		gsAnnotations:      map[string]string{},
		gsUpdateMutex:      sync.RWMutex{},
		gsWaitForSync:      sync.WaitGroup{},
		gsConnectedPlayers: []string{},
		gsStateChannel:     make(chan agonesv1.GameServerState, 2),
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		// Once FeatureCountsAndLists is in GA, move this into SDKServer creation above.
		s.gsCounterUpdates = map[string]counterUpdateRequest{}
		s.gsListUpdates = map[string]listUpdateRequest{}
	}

	s.informerFactory = factory
	s.logger = runtime.NewLoggerWithType(s).WithField("gsKey", namespace+"/"+gameServerName)
	s.logger.Logger.SetLevel(logLevel)

	_, _ = gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)
			s.sendGameServerUpdate(gs)
		},
	})

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(s.logger.Debugf)
	eventBroadcaster.StartRecordingToSink(&k8sv1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	s.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "gameserver-sidecar"})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			s.logger.WithError(err).Error("could not send ok response on healthz")
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/gshealthz", func(w http.ResponseWriter, _ *http.Request) {
		s.ensureHealthChecksRunning()
		if s.healthy() {
			_, err := w.Write([]byte("ok"))
			if err != nil {
				s.logger.WithError(err).Error("could not send ok response on gshealthz")
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	// we haven't synced yet
	s.gsWaitForSync.Add(1)
	s.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(
		s.syncGameServer,
		s.logger,
		logfields.GameServerKey,
		strings.Join([]string{agones.GroupName, s.namespace, s.gameServerName}, "."),
		workerqueue.ConstantRateLimiter(requestsRateLimit))

	s.logger.Info("Created GameServer sidecar")

	return s, nil
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (s *SDKServer) Run(ctx context.Context) error {
	s.informerFactory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), s.gameServerSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	// need this for streaming gRPC commands
	s.ctx = ctx
	// we have the gameserver details now
	s.gsWaitForSync.Done()

	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	s.health = gs.Spec.Health
	s.logger.WithField("health", s.health).Debug("Setting health configuration")
	s.healthTimeout = time.Duration(gs.Spec.Health.PeriodSeconds) * time.Second
	s.touchHealthLastUpdated()

	if gs.Status.State == agonesv1.GameServerStateReserved && gs.Status.ReservedUntil != nil {
		s.gsUpdateMutex.Lock()
		s.resetReserveAfter(context.Background(), time.Until(gs.Status.ReservedUntil.Time))
		s.gsUpdateMutex.Unlock()
	}

	// populate player tracking values
	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		s.gsUpdateMutex.Lock()
		if gs.Status.Players != nil {
			s.gsPlayerCapacity = gs.Status.Players.Capacity
			s.gsConnectedPlayers = gs.Status.Players.IDs
		}
		s.gsUpdateMutex.Unlock()
	}

	// then start the http endpoints
	s.logger.Debug("Starting SDKServer http health check...")
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				s.logger.WithError(err).Error("Health check: http server closed")
			} else {
				err = errors.Wrap(err, "Could not listen on :8080")
				runtime.HandleError(s.logger.WithError(err), err)
			}
		}
	}()
	defer s.server.Close() // nolint: errcheck

	s.workerqueue.Run(ctx, 1)
	return nil
}

// WaitForConnection attempts a GameServer GET every 3s until the client responds.
// This is a workaround for the informer hanging indefinitely on first LIST due
// to a flaky network to the Kubernetes service endpoint.
func (s *SDKServer) WaitForConnection(ctx context.Context) error {
	// In normal operaiton, waitForConnection is called exactly once in Run().
	// In unit tests, waitForConnection() can be called before Run() to ensure
	// that connected is true when Run() is called, otherwise the List() below
	// may race with a test that changes a mock. (Despite the fact that we drop
	// the data on the ground, the Go race detector will pereceive a data race.)
	if s.connected {
		return nil
	}

	try := 0
	return wait.PollUntilContextCancel(ctx, 4*time.Second, true, func(ctx context.Context) (bool, error) {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// Specifically use gameServerGetter since it's the raw client (gameServerLister is the informer).
		// We use List here to avoid needing permission to Get().
		_, err := s.gameServerGetter.GameServers(s.namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", s.gameServerName).String(),
		})
		if err != nil {
			s.logger.WithField("try", try).WithError(err).Info("Connection to Kubernetes service failed")
			try++
			return false, nil
		}
		s.logger.WithField("try", try).Info("Connection to Kubernetes service established")
		s.connected = true
		return true, nil
	})
}

// syncGameServer synchronises the GameServer with the requested operations.
// The format of the key is {operation}. To prevent old operation data from
// overwriting the new one, the operation data is persisted in SDKServer.
func (s *SDKServer) syncGameServer(ctx context.Context, key string) error {
	switch Operation(key) {
	case updateState:
		return s.updateState(ctx)
	case updateLabel:
		return s.updateLabels(ctx)
	case updateAnnotation:
		return s.updateAnnotations(ctx)
	case updatePlayerCapacity:
		return s.updatePlayerCapacity(ctx)
	case updateConnectedPlayers:
		return s.updateConnectedPlayers(ctx)
	case updateCounters:
		return s.updateCounter(ctx)
	case updateLists:
		return s.updateList(ctx)
	}

	return errors.Errorf("could not sync game server key: %s", key)
}

// updateState sets the GameServer Status's state to the one persisted in SDKServer,
// i.e. SDKServer.gsState.
func (s *SDKServer) updateState(ctx context.Context) error {
	s.gsUpdateMutex.RLock()
	s.logger.WithField("state", s.gsState).Debug("Updating state")
	if len(s.gsState) == 0 {
		s.gsUpdateMutex.RUnlock()
		return errors.Errorf("could not update GameServer %s/%s to empty state", s.namespace, s.gameServerName)
	}
	s.gsUpdateMutex.RUnlock()

	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	// If we are currently in shutdown/being deleted, there is no escaping.
	if gs.IsBeingDeleted() {
		s.logger.Debug("GameServerState being shutdown. Skipping update.")

		// Explicitly update gsStateChannel if current state is Shutdown since sendGameServerUpdate will not triggered.
		if s.gsState == agonesv1.GameServerStateShutdown && gs.Status.State != agonesv1.GameServerStateShutdown {
			go func() {
				s.gsStateChannel <- agonesv1.GameServerStateShutdown
			}()
		}

		return nil
	}

	// If the state is currently unhealthy, you can't go back to Ready.
	if gs.Status.State == agonesv1.GameServerStateUnhealthy {
		s.logger.Debug("GameServerState already unhealthy. Skipping update.")
		return nil
	}

	s.gsUpdateMutex.RLock()
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = s.gsState

	// If we are setting the Reserved status, check for the duration, and set that too.
	if gsCopy.Status.State == agonesv1.GameServerStateReserved && s.gsReserveDuration != nil {
		n := metav1.NewTime(time.Now().Add(*s.gsReserveDuration))
		gsCopy.Status.ReservedUntil = &n
	} else {
		gsCopy.Status.ReservedUntil = nil
	}
	s.gsUpdateMutex.RUnlock()

	// If we are setting the Allocated status, set the last-allocated annotation as well.
	if gsCopy.Status.State == agonesv1.GameServerStateAllocated {
		ts, err := s.clock.Now().MarshalText()
		if err != nil {
			return err
		}
		if gsCopy.ObjectMeta.Annotations == nil {
			gsCopy.ObjectMeta.Annotations = map[string]string{}
		}
		gsCopy.ObjectMeta.Annotations[gameserverallocations.LastAllocatedAnnotationKey] = string(ts)
	}

	gs, err = s.patchGameServer(ctx, gs, gsCopy)
	if err != nil {
		return errors.Wrapf(err, "could not update GameServer %s/%s to state %s", s.namespace, s.gameServerName, gsCopy.Status.State)
	}

	message := "SDK state change"
	level := corev1.EventTypeNormal
	// post state specific work here
	switch gs.Status.State {
	case agonesv1.GameServerStateUnhealthy:
		level = corev1.EventTypeWarning
		message = "Health check failure"
	case agonesv1.GameServerStateReserved:
		s.gsUpdateMutex.Lock()
		if s.gsReserveDuration != nil {
			message += fmt.Sprintf(", for %s", s.gsReserveDuration)
			s.resetReserveAfter(context.Background(), *s.gsReserveDuration)
		}
		s.gsUpdateMutex.Unlock()
	}

	s.recorder.Event(gs, level, string(gs.Status.State), message)

	return nil
}

// Gets the GameServer from the cache, or from the local SDKServer if that version is more recent.
func (s *SDKServer) gameServer() (*agonesv1.GameServer, error) {
	// this ensure that if we get requests for the gameserver before the cache has been synced,
	// they will block here until it's ready
	s.gsWaitForSync.Wait()
	gs, err := s.gameServerLister.GameServers(s.namespace).Get(s.gameServerName)
	if err != nil {
		return gs, errors.Wrapf(err, "could not retrieve GameServer %s/%s", s.namespace, s.gameServerName)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	if s.gsCopy != nil && gs.ObjectMeta.Generation < s.gsCopy.Generation {
		return s.gsCopy, nil
	}
	return gs, nil
}

// patchGameServer is a helper function to create and apply a patch update, so the changes in
// gsCopy are applied to the original gs.
func (s *SDKServer) patchGameServer(ctx context.Context, gs, gsCopy *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	patch, err := gs.Patch(gsCopy)
	if err != nil {
		return nil, err
	}

	gs, err = s.gameServerGetter.GameServers(s.namespace).Patch(ctx, gs.GetObjectMeta().GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
	// if the test operation fails, no reason to error log
	if err != nil && k8serrors.IsInvalid(err) {
		err = workerqueue.NewTraceError(err)
	}
	return gs, errors.Wrapf(err, "error attempting to patch gameserver: %s/%s", gsCopy.ObjectMeta.Namespace, gsCopy.ObjectMeta.Name)
}

// updateLabels updates the labels on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsLabels, with the prefix of "agones.dev/sdk-"
func (s *SDKServer) updateLabels(ctx context.Context) error {
	s.logger.WithField("labels", s.gsLabels).Debug("Updating label")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	if len(s.gsLabels) > 0 && gsCopy.ObjectMeta.Labels == nil {
		gsCopy.ObjectMeta.Labels = map[string]string{}
	}
	for k, v := range s.gsLabels {
		gsCopy.ObjectMeta.Labels[metadataPrefix+k] = v
	}
	s.gsUpdateMutex.RUnlock()

	_, err = s.patchGameServer(ctx, gs, gsCopy)
	return err
}

// updateAnnotations updates the Annotations on this GameServer to the ones persisted in SDKServer,
// i.e. SDKServer.gsAnnotations, with the prefix of "agones.dev/sdk-"
func (s *SDKServer) updateAnnotations(ctx context.Context) error {
	s.logger.WithField("annotations", s.gsAnnotations).Debug("Updating annotation")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	if len(s.gsAnnotations) > 0 && gsCopy.ObjectMeta.Annotations == nil {
		gsCopy.ObjectMeta.Annotations = map[string]string{}
	}
	for k, v := range s.gsAnnotations {
		gsCopy.ObjectMeta.Annotations[metadataPrefix+k] = v
	}
	s.gsUpdateMutex.RUnlock()

	_, err = s.patchGameServer(ctx, gs, gsCopy)
	return err
}

// enqueueState enqueue a State change request into the
// workerqueue
func (s *SDKServer) enqueueState(state agonesv1.GameServerState) {
	s.gsUpdateMutex.Lock()
	// Update cached state, but prevent transitions out of `Unhealthy` by the SDK.
	if s.gsState != agonesv1.GameServerStateUnhealthy {
		s.gsState = state
	}
	s.gsUpdateMutex.Unlock()
	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateState)))
}

// Ready enters the RequestReady state change for this GameServer into
// the workqueue so it can be updated
func (s *SDKServer) Ready(_ context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Debug("Received Ready request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateRequestReady)
	return e, nil
}

// Allocate enters an Allocate state change into the workqueue, so it can be updated
func (s *SDKServer) Allocate(_ context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateAllocated)
	return e, nil
}

// Shutdown enters the Shutdown state change for this GameServer into
// the workqueue so it can be updated. If gracefulTermination feature is enabled,
// Shutdown will block on GameServer being shutdown.
func (s *SDKServer) Shutdown(_ context.Context, e *sdk.Empty) (*sdk.Empty, error) {
	s.logger.Debug("Received Shutdown request, adding to queue")
	s.stopReserveTimer()
	s.enqueueState(agonesv1.GameServerStateShutdown)

	return e, nil
}

// Health receives each health ping, and tracks the last time the health
// check was received, to track if a GameServer is healthy
func (s *SDKServer) Health(stream sdk.SDK_HealthServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			s.logger.Debug("Health stream closed.")
			return stream.SendAndClose(&sdk.Empty{})
		}
		if err != nil {
			return errors.Wrap(err, "Error with Health check")
		}
		s.logger.Debug("Health Ping Received")
		s.touchHealthLastUpdated()
	}
}

// SetLabel adds the Key/Value to be used to set the label with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetLabel(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Debug("Adding SetLabel to queue")

	if kv == nil {
		return nil, status.Error(codes.InvalidArgument, "label key/value cannot be nil")
	}

	if errs := validation.IsQualifiedName(kv.Key); len(errs) > 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid label key %q: %s",
			kv.Key,
			strings.Join(errs, ", "),
		)
	}

	if errs := validation.IsValidLabelValue(kv.Value); len(errs) > 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid label value %q for key %q: %s",
			kv.Value,
			kv.Key,
			strings.Join(errs, ", "),
		)
	}

	s.gsUpdateMutex.Lock()
	s.gsLabels[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateLabel)))
	return &sdk.Empty{}, nil
}

// SetAnnotation adds the Key/Value to be used to set the annotations with the metadataPrefix to the `GameServer`
// metdata
func (s *SDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	s.logger.WithField("values", kv).Debug("Adding SetAnnotation to queue")

	if kv == nil {
		return nil, status.Error(codes.InvalidArgument, "annotation key/value cannot be nil")
	}
	// TODO(k8s-1.35): Replace validation.IsQualifiedName with the new
	// qualified name validation function once we bump apimachinery to v0.35+
	if errs := validation.IsQualifiedName(kv.Key); len(errs) > 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid annotation key %q: %s",
			kv.Key,
			strings.Join(errs, ", "),
		)
	}
	s.gsUpdateMutex.Lock()
	s.gsAnnotations[kv.Key] = kv.Value
	s.gsUpdateMutex.Unlock()

	s.workerqueue.Enqueue(cache.ExplicitKey(string(updateAnnotation)))
	return &sdk.Empty{}, nil
}

// GetGameServer returns the current GameServer configuration and state from the backing GameServer CRD
func (s *SDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	s.logger.Debug("Received GetGameServer request")
	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}
	return convert(gs), nil
}

// WatchGameServer sends events through the stream when changes occur to the
// backing GameServer configuration / status
func (s *SDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	s.logger.Debug("Received WatchGameServer request, adding stream to connectedStreams")

	gs, err := s.GetGameServer(context.Background(), &sdk.Empty{})
	if err != nil {
		return err
	}

	if err := stream.Send(gs); err != nil {
		return err
	}

	s.streamMutex.Lock()
	s.connectedStreams = append(s.connectedStreams, stream)
	s.streamMutex.Unlock()
	// don't exit until we shutdown, because that will close the stream
	<-s.ctx.Done()
	return nil
}

// Reserve moves this GameServer to the Reserved state for the Duration specified
func (s *SDKServer) Reserve(_ context.Context, d *sdk.Duration) (*sdk.Empty, error) {
	s.stopReserveTimer()

	e := &sdk.Empty{}

	// 0 is forever.
	if d.Seconds > 0 {
		duration := time.Duration(d.Seconds) * time.Second
		s.gsUpdateMutex.Lock()
		s.gsReserveDuration = &duration
		s.gsUpdateMutex.Unlock()
	}

	s.logger.Debug("Received Reserve request, adding to queue")
	s.enqueueState(agonesv1.GameServerStateReserved)

	return e, nil
}

// resetReserveAfter will move the GameServer back to being ready after the specified duration.
// This function should be wrapped in a s.gsUpdateMutex lock when being called.
func (s *SDKServer) resetReserveAfter(ctx context.Context, duration time.Duration) {
	if s.reserveTimer != nil {
		s.reserveTimer.Stop()
	}

	s.reserveTimer = time.AfterFunc(duration, func() {
		if _, err := s.Ready(ctx, &sdk.Empty{}); err != nil {
			s.logger.WithError(errors.WithStack(err)).Error("error returning to Ready after reserved")
		}
	})
}

// stopReserveTimer stops the reserve timer. This is a no-op and safe to call if the timer is nil
func (s *SDKServer) stopReserveTimer() {
	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	if s.reserveTimer != nil {
		s.reserveTimer.Stop()
	}
	s.gsReserveDuration = nil
}

// PlayerConnect should be called when a player connects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) PlayerConnect(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("playerID", id.PlayerID).Debug("Player Connected")

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// the player is already connected, return false.
	for _, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			return &alpha.Bool{Bool: false}, nil
		}
	}

	if int64(len(s.gsConnectedPlayers)) >= s.gsPlayerCapacity {
		return &alpha.Bool{Bool: false}, errors.New("players are already at capacity")
	}

	// let's retain the original order, as it should be a smaller patch on data change
	s.gsConnectedPlayers = append(s.gsConnectedPlayers, id.PlayerID)
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), updatePeriod)

	return &alpha.Bool{Bool: true}, nil
}

// PlayerDisconnect should be called when a player disconnects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) PlayerDisconnect(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("playerID", id.PlayerID).Debug("Player Disconnected")

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	found := -1
	for i, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			found = i
			break
		}
	}
	if found == -1 {
		return &alpha.Bool{Bool: false}, nil
	}

	// let's retain the original order, as it should be a smaller patch on data change
	s.gsConnectedPlayers = append(s.gsConnectedPlayers[:found], s.gsConnectedPlayers[found+1:]...)
	s.workerqueue.EnqueueAfter(cache.ExplicitKey(string(updateConnectedPlayers)), updatePeriod)

	return &alpha.Bool{Bool: true}, nil
}

// IsPlayerConnected returns if the playerID is currently connected to the GameServer.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) IsPlayerConnected(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	result := &alpha.Bool{Bool: false}

	for _, playerID := range s.gsConnectedPlayers {
		if playerID == id.PlayerID {
			result.Bool = true
			break
		}
	}

	return result, nil
}

// GetConnectedPlayers returns the list of the currently connected player ids.
// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetConnectedPlayers(_ context.Context, _ *alpha.Empty) (*alpha.PlayerIDList, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	return &alpha.PlayerIDList{List: s.gsConnectedPlayers}, nil
}

// GetPlayerCount returns the current player count.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetPlayerCount(_ context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	return &alpha.Count{Count: int64(len(s.gsConnectedPlayers))}, nil
}

// SetPlayerCapacity to change the game server's player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) SetPlayerCapacity(_ context.Context, count *alpha.Count) (*alpha.Empty, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.Lock()
	s.gsPlayerCapacity = count.Count
	s.gsUpdateMutex.Unlock()
	s.workerqueue.Enqueue(cache.ExplicitKey(string(updatePlayerCapacity)))

	return &alpha.Empty{}, nil
}

// GetPlayerCapacity returns the current player capacity, as set by SDK.SetPlayerCapacity()
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (s *SDKServer) GetPlayerCapacity(_ context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	return &alpha.Count{Count: s.gsPlayerCapacity}, nil
}

// GetCounter returns a Counter. Returns error if the counter does not exist.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) GetCounter(_ context.Context, in *beta.GetCounterRequest) (*beta.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	s.logger.WithField("name", in.Name).Debug("Getting Counter")

	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	counter, ok := gs.Status.Counters[in.Name]
	if !ok {
		return nil, errors.Errorf("counter not found: %s", in.Name)
	}
	s.logger.WithField("Get Counter", counter).Debugf("Got Counter %s", in.Name)
	protoCounter := &beta.Counter{Name: in.Name, Count: counter.Count, Capacity: counter.Capacity}
	// If there are batched changes that have not yet been applied, apply them to the Counter.
	// This does NOT validate batched the changes.
	if counterUpdate, ok := s.gsCounterUpdates[in.Name]; ok {
		if counterUpdate.capacitySet != nil {
			protoCounter.Capacity = *counterUpdate.capacitySet
		}
		if counterUpdate.countSet != nil {
			protoCounter.Count = *counterUpdate.countSet
		}
		protoCounter.Count += counterUpdate.diff
		if protoCounter.Count < 0 {
			protoCounter.Count = 0
			s.logger.Debug("truncating Count in Get Counter request to 0")
		}
		if protoCounter.Count > protoCounter.Capacity {
			protoCounter.Count = protoCounter.Capacity
			s.logger.Debug("truncating Count in Get Counter request to Capacity")
		}
		s.logger.WithField("Get Counter", counter).Debugf("Applied Batched Counter Updates %v", counterUpdate)
	}

	return protoCounter, nil
}

// UpdateCounter collapses all UpdateCounterRequests for a given Counter into a single request.
// UpdateCounterRequest must be one and only one of Capacity, Count, or CountDiff.
// Returns error if the Counter does not exist (name cannot be updated).
// Returns error if the Count is out of range [0,Capacity].
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) UpdateCounter(_ context.Context, in *beta.UpdateCounterRequest) (*beta.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	if in.CounterUpdateRequest == nil {
		return nil, errors.Errorf("invalid argument. CounterUpdateRequest: %v cannot be nil", in.CounterUpdateRequest)
	}
	if in.CounterUpdateRequest.CountDiff == 0 && in.CounterUpdateRequest.Count == nil && in.CounterUpdateRequest.Capacity == nil {
		return nil, errors.Errorf("invalid argument. Malformed CounterUpdateRequest: %v", in.CounterUpdateRequest)
	}

	s.logger.WithField("name", in.CounterUpdateRequest.Name).Debug("Update Counter Request")

	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// Check if we already have a batch request started for this Counter. If not, add new request to
	// the gsCounterUpdates map.
	name := in.CounterUpdateRequest.Name
	batchCounter := s.gsCounterUpdates[name]

	counter, ok := gs.Status.Counters[name]
	// We didn't find the Counter named key in the gameserver.
	if !ok {
		return nil, errors.Errorf("counter not found: %s", name)
	}

	batchCounter.counter = *counter.DeepCopy()

	// Updated based on if client call is CapacitySet
	if in.CounterUpdateRequest.Capacity != nil {
		if in.CounterUpdateRequest.Capacity.GetValue() < 0 {
			return nil, errors.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d", in.CounterUpdateRequest.Capacity.GetValue())
		}
		capacitySet := in.CounterUpdateRequest.Capacity.GetValue()
		batchCounter.capacitySet = &capacitySet
	}

	// Update based on if Client call is CountSet
	if in.CounterUpdateRequest.Count != nil {
		// Verify that 0 <= Count >= Capacity
		countSet := in.CounterUpdateRequest.Count.GetValue()
		capacity := batchCounter.counter.Capacity
		if batchCounter.capacitySet != nil {
			capacity = *batchCounter.capacitySet
		}
		if countSet < 0 || countSet > capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", countSet, capacity)
		}
		batchCounter.countSet = &countSet
		// Clear any previous CountIncrement or CountDecrement requests, and add the CountSet as the first item.
		batchCounter.diff = 0
	}

	// Update based on if Client call is CountIncrement or CountDecrement
	if in.CounterUpdateRequest.CountDiff != 0 {
		count := batchCounter.counter.Count
		if batchCounter.countSet != nil {
			count = *batchCounter.countSet
		}
		count += batchCounter.diff + in.CounterUpdateRequest.CountDiff
		// Verify that 0 <= Count >= Capacity
		capacity := batchCounter.counter.Capacity
		if batchCounter.capacitySet != nil {
			capacity = *batchCounter.capacitySet
		}
		if count < 0 || count > capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", count, capacity)
		}
		batchCounter.diff += in.CounterUpdateRequest.CountDiff
	}

	s.gsCounterUpdates[name] = batchCounter

	// Queue up the Update for later batch processing by updateCounters.
	s.workerqueue.Enqueue(cache.ExplicitKey(updateCounters))
	return projectCounterState(name, batchCounter), nil
}

// projectCounterState calculates the final expected Counter state after applying batched updates.
func projectCounterState(name string, batchCounter counterUpdateRequest) *beta.Counter {
	currentCapacity := batchCounter.counter.Capacity
	if batchCounter.capacitySet != nil {
		currentCapacity = *batchCounter.capacitySet
	}
	currentCount := batchCounter.counter.Count
	if batchCounter.countSet != nil {
		currentCount = *batchCounter.countSet
	}
	currentCount += batchCounter.diff
	if currentCount < 0 {
		currentCount = 0
	}
	if currentCount > currentCapacity {
		currentCount = currentCapacity
	}
	return &beta.Counter{
		Name:     name,
		Count:    currentCount,
		Capacity: currentCapacity,
	}
}

// updateCounter updates the Counters in the GameServer's Status with the batched update requests.
func (s *SDKServer) updateCounter(ctx context.Context) error {
	gs, err := s.gameServer()
	if err != nil {
		return err
	}
	gsCopy := gs.DeepCopy()

	s.logger.WithField("batchCounterUpdates", s.gsCounterUpdates).Debug("Batch updating Counter(s)")
	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	names := []string{}

	for name, ctrReq := range s.gsCounterUpdates {
		counter, ok := gsCopy.Status.Counters[name]
		if !ok {
			continue
		}
		// Changes may have been made to the Counter since we validated the incoming changes in
		// UpdateCounter, and we need to verify if the batched changes can be fully applied, partially
		// applied, or cannot be applied.
		if ctrReq.capacitySet != nil {
			counter.Capacity = *ctrReq.capacitySet
		}
		if ctrReq.countSet != nil {
			counter.Count = *ctrReq.countSet
		}
		newCnt := counter.Count + ctrReq.diff
		if newCnt < 0 {
			newCnt = 0
			s.logger.Debug("truncating Count in Update Counter request to 0")
		}
		if newCnt > counter.Capacity {
			newCnt = counter.Capacity
			s.logger.Debug("truncating Count in Update Counter request to Capacity")
		}
		counter.Count = newCnt
		gsCopy.Status.Counters[name] = counter
		names = append(names, name)
	}

	gs, err = s.patchGameServer(ctx, gs, gsCopy)
	if err != nil {
		return err
	}

	// Record an event per update Counter
	for _, name := range names {
		s.recorder.Event(gs, corev1.EventTypeNormal, "UpdateCounter",
			fmt.Sprintf("Counter %s updated to Count:%d Capacity:%d",
				name, gs.Status.Counters[name].Count, gs.Status.Counters[name].Capacity))
	}

	// Cache a copy of the successfully updated gameserver
	s.gsCopy = gs
	// Clear the gsCounterUpdates
	s.gsCounterUpdates = map[string]counterUpdateRequest{}

	return nil
}

// GetList returns a List. Returns not found if the List does not exist.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) GetList(_ context.Context, in *beta.GetListRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	if in == nil {
		return nil, errors.Errorf("GetListRequest cannot be nil")
	}
	s.logger.WithField("name", in.Name).Debug("Getting List")

	gs, err := s.gameServer()
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()

	list, ok := gs.Status.Lists[in.Name]
	if !ok {
		return nil, errors.Errorf("list not found: %s", in.Name)
	}

	s.logger.WithField("Get List", list).Debugf("Got List %s", in.Name)
	protoList := beta.List{Name: in.Name, Values: list.Values, Capacity: list.Capacity}
	// If there are batched changes that have not yet been applied, apply them to the List.
	// This does NOT validate batched the changes, and does NOT modify the List.
	if listUpdate, ok := s.gsListUpdates[in.Name]; ok {
		if listUpdate.capacitySet != nil {
			protoList.Capacity = *listUpdate.capacitySet
		}
		if len(listUpdate.valuesToDelete) != 0 {
			protoList.Values = deleteValues(protoList.Values, listUpdate.valuesToDelete)
		}
		if len(listUpdate.valuesToAppend) != 0 {
			protoList.Values = agonesv1.MergeRemoveDuplicates(protoList.Values, listUpdate.valuesToAppend)
		}
		// Truncates Values to less than or equal to Capacity
		if len(protoList.Values) > int(protoList.Capacity) {
			protoList.Values = append([]string{}, protoList.Values[:protoList.Capacity]...)
		}
		s.logger.WithField("Get List", list).Debugf("Applied Batched List Updates %v", listUpdate)
	}

	return &protoList, nil
}

// UpdateList collapses all update capacity requests for a given List into a single UpdateList request.
// This function currently only updates the Capacity of a List.
// Returns error if the List does not exist (name cannot be updated).
// Returns error if the List update capacity is out of range [0,1000].
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) UpdateList(ctx context.Context, in *beta.UpdateListRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	if in == nil {
		return nil, errors.Errorf("UpdateListRequest cannot be nil")
	}
	if in.List == nil || in.UpdateMask == nil {
		return nil, errors.Errorf("invalid argument. List: %v and UpdateMask %v cannot be nil", in.List, in.UpdateMask)
	}
	if !in.UpdateMask.IsValid(in.List.ProtoReflect().Interface()) {
		return nil, errors.Errorf("invalid argument. Field Mask Path(s): %v are invalid for List. Use valid field name(s): %v", in.UpdateMask.GetPaths(), in.List.ProtoReflect().Descriptor().Fields())
	}

	if GameServerListMaxCapacity == 0 {
		err := s.GsListsMaxItems(ctx)
		if err != nil {
			return nil, errors.Errorf("%v", err)
		}
	}
	if in.List.Capacity < 0 || in.List.Capacity > GameServerListMaxCapacity {
		return nil, errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", in.List.Capacity)
	}

	list, err := s.GetList(ctx, &beta.GetListRequest{Name: in.List.Name})
	if err != nil {

		return nil, errors.Errorf("not found. %s List not found", list.Name)
	}

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// Removes any fields from the request object that are not included in the FieldMask Paths.
	fmutils.Filter(in.List, in.UpdateMask.Paths)

	// The list will allow the current list to be overwritten
	batchList := listUpdateRequest{}

	// Only set the capacity if its included in the update mask paths
	if slices.Contains(in.UpdateMask.Paths, "capacity") {
		batchList.capacitySet = &in.List.Capacity
	}

	// Only change the values if its included in the update mask paths
	if slices.Contains(in.UpdateMask.Paths, "values") {
		currList := list

		// Find values to remove from the current list
		valuesToDelete := map[string]bool{}
		for _, value := range currList.Values {
			valueFound := false
			for _, element := range in.List.Values {
				if value == element {
					valueFound = true
				}
			}

			if !valueFound {
				valuesToDelete[value] = true
			}
		}
		batchList.valuesToDelete = valuesToDelete

		// Find values that need to be added to the current list from the incomming list
		valuesToAdd := []string{}
		for _, value := range in.List.Values {
			valueFound := false
			for _, element := range currList.Values {
				if value == element {
					valueFound = true
				}
			}

			if !valueFound {
				valuesToAdd = append(valuesToAdd, value)
			}
		}
		batchList.valuesToAppend = valuesToAdd
	}

	// Queue up the Update for later batch processing by updateLists.
	s.gsListUpdates[list.Name] = batchList
	s.workerqueue.Enqueue(cache.ExplicitKey(updateLists))
	return &beta.List{}, nil

}

// AddListValue collapses all append a value to the end of a List requests into a single UpdateList request.
// Returns not found if the List does not exist.
// Returns already exists if the value is already in the List.
// Returns out of range if the List is already at Capacity.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) AddListValue(ctx context.Context, in *beta.AddListValueRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	if in == nil {
		return nil, errors.Errorf("AddListValueRequest cannot be nil")
	}
	s.logger.WithField("name", in.Name).Debug("Add List Value")

	list, err := s.GetList(ctx, &beta.GetListRequest{Name: in.Name})
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// Verify room to add another value
	if int(list.Capacity) <= len(list.Values) {
		return nil, errors.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", list.Capacity, len(list.Values))
	}
	// Verify value does not already exist in the list
	for _, val := range list.Values {
		if in.Value == val {
			return nil, errors.Errorf("already exists. Value: %s already in List: %s", in.Value, in.Name)
		}
	}
	list.Values = append(list.Values, in.Value)
	batchList := s.gsListUpdates[in.Name]
	batchList.valuesToAppend = append(batchList.valuesToAppend, in.Value)
	s.gsListUpdates[in.Name] = batchList
	// Queue up the Update for later batch processing by updateLists.
	s.workerqueue.Enqueue(cache.ExplicitKey(updateLists))
	return list, nil
}

// RemoveListValue collapses all remove a value from a List requests into a single UpdateList request.
// Returns not found if the List does not exist.
// Returns not found if the value is not in the List.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (s *SDKServer) RemoveListValue(ctx context.Context, in *beta.RemoveListValueRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}
	if in == nil {
		return nil, errors.Errorf("RemoveListValueRequest cannot be nil")
	}
	s.logger.WithField("name", in.Name).WithField("value", in.Value).Debug("Remove List Value")

	list, err := s.GetList(ctx, &beta.GetListRequest{Name: in.Name})
	if err != nil {
		return nil, err
	}

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	// Track this removal for batch persistence to K8s
	batchList := s.gsListUpdates[in.Name]

	removedFromBatch := false
	if len(batchList.valuesToAppend) > 0 {
		newAppend := make([]string, 0, len(batchList.valuesToAppend))
		for _, v := range batchList.valuesToAppend {
			if v == in.Value {
				removedFromBatch = true
				continue // skip value
			}
			newAppend = append(newAppend, v)
		}
		batchList.valuesToAppend = newAppend
	}
	if !removedFromBatch {
		found := false
		newValues := make([]string, 0, len(list.Values))
		for _, val := range list.Values {
			if val == in.Value {
				found = true
				continue
			}
			newValues = append(newValues, val)
		}
		if !found {
			return nil, fmt.Errorf("not found: value %s not in list %s", in.Value, in.Name)
		}
		list.Values = newValues
		// Track deletions
		if batchList.valuesToDelete == nil {
			batchList.valuesToDelete = make(map[string]bool)
		}
		batchList.valuesToDelete[in.Value] = true
	}
	s.gsListUpdates[in.Name] = batchList
	// Queue up the Update for later batch processing by updateLists.
	s.workerqueue.Enqueue(cache.ExplicitKey(updateLists))
	return list, nil
}

// updateList updates the Lists in the GameServer's Status with the batched update list requests.
// Includes all SetCapacity, AddValue, and RemoveValue requests in the batched request.
func (s *SDKServer) updateList(ctx context.Context) error {
	gs, err := s.gameServer()
	if err != nil {
		return err
	}
	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.Lock()
	defer s.gsUpdateMutex.Unlock()

	s.logger.WithField("batchListUpdates", s.gsListUpdates).Debug("Batch updating List(s)")

	names := []string{}

	for name, listReq := range s.gsListUpdates {
		list, ok := gsCopy.Status.Lists[name]
		if !ok {
			continue
		}
		if listReq.capacitySet != nil {
			list.Capacity = *listReq.capacitySet
		}
		if len(listReq.valuesToDelete) != 0 {
			list.Values = deleteValues(list.Values, listReq.valuesToDelete)
		}
		if len(listReq.valuesToAppend) != 0 {
			list.Values = agonesv1.MergeRemoveDuplicates(list.Values, listReq.valuesToAppend)
		}

		if int64(len(list.Values)) > list.Capacity {
			s.logger.Debugf("truncating Values in Update List request to List Capacity %d", list.Capacity)
			list.Values = append([]string{}, list.Values[:list.Capacity]...)
		}
		gsCopy.Status.Lists[name] = list
		names = append(names, name)
	}

	gs, err = s.patchGameServer(ctx, gs, gsCopy)
	if err != nil {
		return err
	}

	// Record an event per List update
	for _, name := range names {
		s.recorder.Event(gs, corev1.EventTypeNormal, "UpdateList", fmt.Sprintf("List %s updated", name))
		s.logger.Debugf("List %s updated to List Capacity: %d, Values: %v",
			name, gs.Status.Lists[name].Capacity, gs.Status.Lists[name].Values)
	}

	// Cache a copy of the successfully updated gameserver
	s.gsCopy = gs
	// Clear the gsListUpdates
	s.gsListUpdates = map[string]listUpdateRequest{}

	return nil
}

// Returns a new string list with the string keys in toDeleteValues removed from valuesList.
func deleteValues(valuesList []string, toDeleteValues map[string]bool) []string {
	newValuesList := []string{}
	for _, value := range valuesList {
		if _, ok := toDeleteValues[value]; ok {
			continue
		}
		newValuesList = append(newValuesList, value)
	}
	return newValuesList
}

// sendGameServerUpdate sends a watch game server event
func (s *SDKServer) sendGameServerUpdate(gs *agonesv1.GameServer) {
	s.logger.Debug("Sending GameServer Event to connectedStreams")

	s.streamMutex.Lock()
	defer s.streamMutex.Unlock()

	// Filter the slice of streams sharing the same backing array and capacity as the original
	// so that storage is reused and no memory allocations are made. This modifies the original
	// slice.
	//
	// See https://go.dev/wiki/SliceTricks#filtering-without-allocating
	remainingStreams := s.connectedStreams[:0]
	for _, stream := range s.connectedStreams {
		select {
		case <-stream.Context().Done():
			s.logger.Debug("Dropping stream")

			err := stream.Context().Err()
			switch {
			case err != nil:
				s.logger.WithError(errors.WithStack(err)).Error("stream closed with error")
			default:
				s.logger.Debug("Stream closed")
			}
		default:
			s.logger.Debug("Keeping stream")
			remainingStreams = append(remainingStreams, stream)

			if err := stream.Send(convert(gs)); err != nil {
				s.logger.WithError(errors.WithStack(err)).
					Error("error sending game server update event")
			}
		}
	}
	s.connectedStreams = remainingStreams

	if gs.Status.State == agonesv1.GameServerStateShutdown {
		// Wrap this in a go func(), just in case pushing to this channel deadlocks since there is only one instance of
		// a receiver. In theory, This could leak goroutines a bit, but since we're shuttling down everything anyway,
		// it shouldn't matter.
		go func() {
			s.gsStateChannel <- agonesv1.GameServerStateShutdown
		}()
	}
}

// checkHealthUpdateState checks the health as part of the /gshealthz hook, and if not
// healthy will push the Unhealthy state into the queue so it can be updated.
func (s *SDKServer) checkHealthUpdateState() {
	s.checkHealth()
	if !s.healthy() {
		s.logger.WithField("gameServerName", s.gameServerName).Warn("GameServer has failed health check")
		s.enqueueState(agonesv1.GameServerStateUnhealthy)
	}
}

// touchHealthLastUpdated sets the healthLastUpdated
// value to now in UTC
func (s *SDKServer) touchHealthLastUpdated() {
	s.healthMutex.Lock()
	defer s.healthMutex.Unlock()
	s.healthLastUpdated = s.clock.Now().UTC()
	s.healthFailureCount = 0
}

func (s *SDKServer) ensureHealthChecksRunning() {
	if s.health.Disabled {
		return
	}
	s.healthChecksRunning.Do(func() {
		// start health checking running
		s.logger.Debug("Starting GameServer health checking")
		go wait.Until(s.checkHealthUpdateState, s.healthTimeout, s.ctx.Done())
	})
}

// checkHealth checks the healthLastUpdated value
// and if it is outside the timeout value, logger and
// count a failure
func (s *SDKServer) checkHealth() {
	s.healthMutex.Lock()
	defer s.healthMutex.Unlock()

	timeout := s.healthLastUpdated.Add(s.healthTimeout)
	if timeout.Before(s.clock.Now().UTC()) {
		s.healthFailureCount++
		s.logger.WithField("failureCount", s.healthFailureCount).Warn("GameServer Health Check failed")
	}
}

// healthy returns if the GameServer is
// currently healthy or not based on the configured
// failure count vs failure threshold
func (s *SDKServer) healthy() bool {
	if s.health.Disabled {
		return true
	}

	s.healthMutex.RLock()
	defer s.healthMutex.RUnlock()
	return s.healthFailureCount < s.health.FailureThreshold
}

// updatePlayerCapacity updates the Player Capacity field in the GameServer's Status.
func (s *SDKServer) updatePlayerCapacity(ctx context.Context) error {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	s.logger.WithField("capacity", s.gsPlayerCapacity).Debug("updating player capacity")
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()

	s.gsUpdateMutex.RLock()
	gsCopy.Status.Players.Capacity = s.gsPlayerCapacity
	s.gsUpdateMutex.RUnlock()

	gs, err = s.patchGameServer(ctx, gs, gsCopy)
	if err == nil {
		s.recorder.Event(gs, corev1.EventTypeNormal, "PlayerCapacity", fmt.Sprintf("Set to %d", gs.Status.Players.Capacity))
	}

	return err
}

// updateConnectedPlayers updates the Player IDs and Count fields in the GameServer's Status.
func (s *SDKServer) updateConnectedPlayers(ctx context.Context) error {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	gs, err := s.gameServer()
	if err != nil {
		return err
	}

	gsCopy := gs.DeepCopy()
	same := false
	s.gsUpdateMutex.RLock()
	s.logger.WithField("playerIDs", s.gsConnectedPlayers).Debug("updating connected players")
	same = apiequality.Semantic.DeepEqual(gsCopy.Status.Players.IDs, s.gsConnectedPlayers)
	gsCopy.Status.Players.IDs = s.gsConnectedPlayers
	gsCopy.Status.Players.Count = int64(len(s.gsConnectedPlayers))
	s.gsUpdateMutex.RUnlock()
	// if there is no change, then don't update
	// since it's possible this could fire quite a lot, let's reduce the
	// amount of requests as much as possible.
	if same {
		return nil
	}

	gs, err = s.patchGameServer(ctx, gs, gsCopy)
	if err == nil {
		s.recorder.Event(gs, corev1.EventTypeNormal, "PlayerCount", fmt.Sprintf("Set to %d", gs.Status.Players.Count))
	}

	return err
}

// NewSDKServerContext returns a Context that cancels when SIGTERM or os.Interrupt
// is received and the GameServer's Status is shutdown
func (s *SDKServer) NewSDKServerContext(ctx context.Context) context.Context {
	sdkCtx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()

		keepWaiting := true
		s.gsUpdateMutex.RLock()
		if len(s.gsState) > 0 {
			s.logger.WithField("state", s.gsState).Info("SDK server shutdown requested, waiting for game server shutdown")
		} else {
			s.logger.Info("SDK server state never updated by game server, shutting down sdk server without waiting")
			keepWaiting = false
		}
		s.gsUpdateMutex.RUnlock()

		for keepWaiting {
			gsState := <-s.gsStateChannel
			if gsState == agonesv1.GameServerStateShutdown {
				keepWaiting = false
			}
		}

		cancel()
	}()
	return sdkCtx
}

func (s *SDKServer) gsListUpdatesLen() int {
	s.gsUpdateMutex.RLock()
	defer s.gsUpdateMutex.RUnlock()
	return len(s.gsListUpdates)
}

// GsListsMaxItems lists all GameServers belonging to a specific fleet and attempts
// to retrieve the maximum capacity defined for a given listName in the GameServer's Spec.
// It returns the capacity of the first GameServer found that defines the list.
func (s *SDKServer) GsListsMaxItems(ctx context.Context) error {

	var labelsName string
	if s.namespace == "default" {
		labelsName = "fleet-example"
	} else {
		gs, _ := s.gameServerGetter.GameServers(s.namespace).Get(
			ctx,
			s.gameServerName,
			metav1.GetOptions{},
		)
		labelsName = gs.ObjectMeta.Labels[agonesv1.FleetNameLabel]
	}
	selector := labels.Set{
		agonesv1.FleetNameLabel: labelsName,
	}

	labelSelector := selector.String()
	gsList, err := s.gameServerGetter.GameServers(s.namespace).List(
		ctx,
		metav1.ListOptions{
			LabelSelector: labelSelector,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list GameServers for fleet %s: %w", "fleet-example", err)
	}

	if len(gsList.Items) == 0 {
		return fmt.Errorf("no gameservers for fleet")
	}
	for i := range gsList.Items {
		// Get a pointer to the listed item (only for accessing the name)
		listedGS := &gsList.Items[i]
		// Fetch the current GameServer resource
		gs, err := s.gameServerGetter.GameServers(s.namespace).Get(
			ctx,
			listedGS.Name, // Use the name from the listed item
			metav1.GetOptions{},
		)
		if err != nil {
			return fmt.Errorf("failed to get GameServer %s: %w", listedGS.Name, err)
		}
		if playersList, found := gs.Spec.Lists["players"]; found {
			GameServerListMaxCapacity = playersList.Capacity
		}
	}
	return nil
}
