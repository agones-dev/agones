// Copyright 2025 Google LLC All Rights Reserved.
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

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/clientset/versioned/scheme"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// SucceededController changes the state of a GameServer to Shutdown
// when its Pod has a backing state of Succeeded.
type SucceededController struct {
	baseLogger       *logrus.Entry
	podSynced        cache.InformerSynced
	podLister        corelisterv1.PodLister
	gameServerSynced cache.InformerSynced
	gameServerGetter getterv1.GameServersGetter
	gameServerLister listerv1.GameServerLister
	workerqueue      *workerqueue.WorkerQueue
	recorder         record.EventRecorder
}

// NewSucceededController creates a new SucceededController and sets up event handlers.
func NewSucceededController(health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *SucceededController {
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	gameServers := agonesInformerFactory.Agones().V1().GameServers()

	c := &SucceededController{
		podSynced:        podInformer.HasSynced,
		podLister:        kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerSynced: gameServers.Informer().HasSynced,
		gameServerGetter: agonesClient.AgonesV1(),
		gameServerLister: gameServers.Lister(),
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncGameServer, c.baseLogger, logfields.GameServerKey, agones.GroupName+".SucceededController")
	health.AddLivenessCheck("gameserver-succeeded-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "succeeded-controller"})

	_, _ = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			if isGameServerPod(pod) && pod.Status.Phase == corev1.PodSucceeded {
				c.workerqueue.Enqueue(pod)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if isGameServerPod(pod) && pod.Status.Phase == corev1.PodSucceeded {
				c.workerqueue.Enqueue(pod)
			}
		},
	})

	// Recovery path: if a pod Succeeded event was missed (e.g. during controller restart),
	// the GameServer informer resync will re-check the pod phase and re-enqueue.
	_, _ = gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)
			if _, isDev := gs.GetDevAddress(); isDev {
				return
			}
			if gs.IsBeingDeleted() || agonesv1.TerminalGameServerStates[gs.Status.State] || isBeforePodCreated(gs) {
				return
			}
			pod, err := c.podLister.Pods(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name)
			if err == nil && isGameServerPod(pod) && pod.Status.Phase == corev1.PodSucceeded {
				c.workerqueue.Enqueue(pod)
			}
		},
	})

	return c
}

// Run starts the SucceededController worker queue after ensuring caches are synced.
func (c *SucceededController) Run(ctx context.Context, workers int) error {
	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.podSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(ctx, workers)
	return nil
}

func (c *SucceededController) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerKey, key)
}

// syncGameServer changes a GameServer to Shutdown state when its Pod is in Succeeded state
func (c *SucceededController) syncGameServer(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForGameServerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	// check if the pod exists and is in Succeeded state
	pod, err := c.podLister.Pods(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error retrieving Pod %s from namespace %s", name, namespace)
		}
		// If the pod doesn't exist, we don't need to do anything
		return nil
	}

	// If the pod exists but is not in Succeeded state or is being terminated, we don't need to do anything
	if !isGameServerPod(pod) || pod.Status.Phase != corev1.PodSucceeded || !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	c.loggerForGameServerKey(key).Debug("Pod is in Succeeded state. Moving GameServer to Shutdown.")

	gs, err := c.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForGameServerKey(key).Debug("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	// already on the way out, so no need to do anything.
	if gs.IsBeingDeleted() || agonesv1.TerminalGameServerStates[gs.Status.State] {
		c.loggerForGameServerKey(key).WithField("state", gs.Status.State).Debug("GameServer already being deleted/shutdown. Skipping.")
		return nil
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateShutdown
	gs, err = c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "error updating GameServer to Shutdown")
	}

	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Pod is in Succeeded state")
	return nil
}
