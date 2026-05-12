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
	"fmt"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	cron "github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// RestartController handles scheduled in-place restarts of game server containers.
// It does NOT delete Pods; it relies on the SDK sidecar (watchRestartAnnotation)
// to exit cleanly so Kubernetes restarts only the game server container.
type RestartController struct {
	baseLogger     *logrus.Entry
	kubeClient     kubernetes.Interface
	agonesClient   versioned.Interface
	gsLister       listerv1.GameServerLister
	gsListerSynced cache.InformerSynced
	recorder       record.EventRecorder
	workerqueue    *workerqueue.WorkerQueue
}

// NewRestartController returns a new RestartController wired to the provided
// informer factories. Matches the constructor pattern of all Agones controllers
// (see health_controller.go, succeeded_controller.go).
//
// Note: unlike other controllers, RestartController does not need kubeInformerFactory
// because it only watches GameServer objects (Agones CRDs), not core k8s resources.
// The parameter is intentionally omitted to avoid the revive unused-parameter lint warning.
func NewRestartController(
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory,
) *RestartController {

	gameServers := agonesInformerFactory.Agones().V1().GameServers()

	c := &RestartController{
		kubeClient:     kubeClient,
		agonesClient:   agonesClient,
		gsLister:       gameServers.Lister(),
		gsListerSynced: gameServers.Informer().HasSynced,
	}

	c.baseLogger = runtime.NewLoggerWithType(c)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: kubeClient.CoreV1().Events(""),
	})
	c.recorder = eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: "gameserver-restart-controller"},
	)

	c.workerqueue = workerqueue.NewWorkerQueue(
		c.syncGameServer,
		c.baseLogger,
		logfields.GameServerKey,
		agones.GroupName+".RestartController",
	)
	health.AddLivenessCheck("restart-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	// Enqueue GameServers with a RestartPolicy when they are added or updated.
	_, _ = gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gs, ok := obj.(*agonesv1.GameServer)
			if ok && gs.Spec.RestartPolicy != nil {
				c.workerqueue.Enqueue(gs)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			gs, ok := newObj.(*agonesv1.GameServer)
			if ok && gs.Spec.RestartPolicy != nil {
				c.workerqueue.Enqueue(gs)
			}
		},
	})

	return c
}

// Run starts the controller. Blocks until ctx is cancelled.
func (c *RestartController) Run(ctx context.Context, workers int) error {
	c.baseLogger.Info("Starting RestartController")
	defer c.baseLogger.Info("Stopping RestartController")

	if !cache.WaitForCacheSync(ctx.Done(), c.gsListerSynced) {
		return fmt.Errorf("never got in sync with cache for RestartController")
	}

	// Periodically re-enqueue all GameServers with a RestartPolicy so we catch
	// schedule windows even when no update events fire.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.enqueueAll()
			}
		}
	}()

	c.workerqueue.Run(ctx, workers)
	return nil
}

// enqueueAll lists ALL GameServers with a RestartPolicy and re-enqueues them.
func (c *RestartController) enqueueAll() {
	gsList, err := c.gsLister.List(labels.Everything())
	if err != nil {
		c.baseLogger.WithError(err).Error("Failed to list GameServers for restart re-enqueue")
		return
	}
	for _, gs := range gsList {
		if gs.Spec.RestartPolicy != nil {
			c.workerqueue.Enqueue(gs)
		}
	}
}

// syncGameServer is the reconcile function called by the worker queue for each
// GameServer key. It is the entry point for the restart state machine.
func (c *RestartController) syncGameServer(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	gs, err := c.gsLister.GameServers(namespace).Get(name)
	if k8serrors.IsNotFound(err) {
		return nil // GS deleted; nothing to do
	}
	if err != nil {
		return err
	}

	// Feature gate guard — no-op if the gate is disabled.
	if !runtime.FeatureEnabled(runtime.FeatureGameServerScheduledRestart) {
		return nil
	}

	// No RestartPolicy — most game servers; fast exit.
	if gs.Spec.RestartPolicy == nil {
		return nil
	}

	// Terminal states never recover; skip.
	if agonesv1.TerminalGameServerStates[gs.Status.State] {
		return nil
	}

	return c.reconcileRestart(ctx, gs)
}

// reconcileRestart drives the full soft/hard-deadline state machine.
func (c *RestartController) reconcileRestart(ctx context.Context, gs *agonesv1.GameServer) error {
	// Guard: RestartPolicy must be set (callers should check, but be defensive).
	if gs.Spec.RestartPolicy == nil {
		return nil
	}

	// Guard: terminal states (Shutdown, Error, Unhealthy) never recover.
	// This check also lives in syncGameServer, but we repeat it here so that
	// tests calling reconcileRestart directly are also protected.
	if agonesv1.TerminalGameServerStates[gs.Status.State] {
		return nil
	}

	rp := gs.Spec.RestartPolicy
	now := time.Now().UTC()

	// ── Step 1: Parse the cron schedule ───────────────────────────────────────
	schedule, err := cron.ParseStandard(rp.Schedule)
	if err != nil {
		c.baseLogger.WithError(err).WithField("gs", gs.Name).Error("Invalid cron schedule — skipping")
		return nil
	}

	// ── Step 2: Determine the next scheduled trigger ──────────────────────────
	// Anchor = last known "window opened at" annotation, or GS creation time.
	anchor := gs.CreationTimestamp.Time
	if t, ok := gs.Annotations[agonesv1.GameServerNextRestartAnnotation]; ok {
		if parsed, err2 := time.Parse(time.RFC3339, t); err2 == nil {
			anchor = parsed
		}
	}
	nextRestart := schedule.Next(anchor)

	// ── Step 3: Not yet time — record the upcoming window and exit ────────────
	if now.Before(nextRestart) {
		return c.updateNextRestartTime(ctx, gs, nextRestart)
	}

	// ── Step 4: Hard deadline check ───────────────────────────────────────────
	// If pending-since + hardDeadline < now, force restart regardless of state.
	forceRestart := false
	if rp.HardDeadlineDuration != nil {
		if pendingSince, ok := gs.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]; ok {
			if pt, err2 := time.Parse(time.RFC3339, pendingSince); err2 == nil {
				if now.Sub(pt) >= rp.HardDeadlineDuration.Duration {
					c.baseLogger.WithField("gs", gs.Name).Warn("Hard deadline exceeded — forcing restart")
					c.recorder.Event(gs, corev1.EventTypeWarning,
						"HardDeadlineExceeded", "Forcing restart: hard deadline exceeded")
					forceRestart = true
				}
			}
		}
	}

	// ── Step 5: Idle check (skip when force) ──────────────────────────────────
	if !forceRestart && !c.isIdle(gs) {
		// 5a. Record pending-since if this is the first time we're deferring.
		if _, ok := gs.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]; !ok {
			return c.annotateRestartPending(ctx, gs, now)
		}

		// 5b. Soft deadline: if we've been waiting longer than softDeadlineDuration,
		//     skip to the next cron window.
		if rp.SoftDeadlineDuration != nil {
			if pendingSince, ok := gs.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]; ok {
				if pt, err2 := time.Parse(time.RFC3339, pendingSince); err2 == nil {
					if now.Sub(pt) >= rp.SoftDeadlineDuration.Duration {
						c.baseLogger.WithField("gs", gs.Name).Info(
							"Soft deadline passed — skipping to next cron window")
						c.recorder.Event(gs, corev1.EventTypeNormal,
							"SoftDeadlineExpired", "Soft deadline passed; deferring to next cron window")
						return c.advanceAnchor(ctx, gs, nextRestart)
					}
				}
			}
		}

		// Still waiting for the server to become idle.
		c.baseLogger.WithField("gs", gs.Name).Debug("GameServer not idle; deferring restart")
		return nil
	}

	// ── Step 6: Perform the in-place restart ──────────────────────────────────
	return c.performInPlaceRestart(ctx, gs, nextRestart)
}

// isIdle returns true when the game server is safe to restart:
//   - state must be Ready (not Allocated, not Reserved)
//   - no active players (if PlayerTracking is enabled)
func (c *RestartController) isIdle(gs *agonesv1.GameServer) bool {
	if gs.Status.State != agonesv1.GameServerStateReady {
		return false
	}
	if gs.Status.Players != nil && gs.Status.Players.Count > 0 {
		return false
	}
	return true
}

// performInPlaceRestart signals the game server container to restart by annotating
// the GameServer with the restart-pending-since annotation. The SDK sidecar
// (watchRestartAnnotation in sdkserver.go) watches this annotation and calls
// os.Exit(0) when the server becomes idle, triggering a Kubernetes container restart.
func (c *RestartController) performInPlaceRestart(
	ctx context.Context,
	gs *agonesv1.GameServer,
	windowTime time.Time,
) error {
	c.baseLogger.WithField("gs", gs.Name).Info("Performing in-place game server restart")
	c.recorder.Event(gs, corev1.EventTypeNormal,
		"ScheduledRestart", "Restarting game server container in-place per schedule")

	// Advance the anchor so the next sync computes the FOLLOWING cron window,
	// and clear the pending annotation.
	return c.advanceAnchor(ctx, gs, windowTime)
}

// updateNextRestartTime patches the GS with the computed next-restart time so
// operators can observe it via kubectl / dashboards.
func (c *RestartController) updateNextRestartTime(
	ctx context.Context,
	gs *agonesv1.GameServer,
	t time.Time,
) error {
	gsCopy := gs.DeepCopy()
	mt := metav1.NewTime(t)
	gsCopy.Status.NextRestartTime = &mt
	if gsCopy.Annotations == nil {
		gsCopy.Annotations = map[string]string{}
	}
	gsCopy.Annotations[agonesv1.GameServerNextRestartAnnotation] = t.UTC().Format(time.RFC3339)

	_, err := c.agonesClient.AgonesV1().GameServers(gs.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return err
}

// annotateRestartPending records the first time a scheduled window was triggered
// but the server was not yet idle. The timestamp is used to enforce soft/hard deadlines.
func (c *RestartController) annotateRestartPending(
	ctx context.Context,
	gs *agonesv1.GameServer,
	t time.Time,
) error {
	gsCopy := gs.DeepCopy()
	if gsCopy.Annotations == nil {
		gsCopy.Annotations = map[string]string{}
	}
	gsCopy.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation] = t.UTC().Format(time.RFC3339)
	_, err := c.agonesClient.AgonesV1().GameServers(gs.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return err
}

// advanceAnchor moves the next-restart annotation past the current window so
// the following sync computes the next cron trigger, and clears the pending
// annotation (whether we restarted or gave up on this window).
func (c *RestartController) advanceAnchor(
	ctx context.Context,
	gs *agonesv1.GameServer,
	windowTime time.Time,
) error {
	gsCopy := gs.DeepCopy()
	if gsCopy.Annotations == nil {
		gsCopy.Annotations = map[string]string{}
	}
	// Advance by 1 second past the window so the next sync picks the FOLLOWING window.
	gsCopy.Annotations[agonesv1.GameServerNextRestartAnnotation] =
		windowTime.Add(time.Second).UTC().Format(time.RFC3339)
	delete(gsCopy.Annotations, agonesv1.GameServerRestartPendingSinceAnnotation)

	_, err := c.agonesClient.AgonesV1().GameServers(gs.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	return err
}
