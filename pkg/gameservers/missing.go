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

// MissingPodController makes sure that any GameServer that isn't in a
// Scheduled or Unhealthy state and is either missing a Pod, or has a Pod in a
// Failed state, is moved to Unhealthy.
//
// It's possible that a GameServer is missing its associated pod due to
// unexpected controller downtime or if the Pod is deleted with no subsequent Delete event.
// Similarly, a Pod can reach the Failed phase (e.g. non-zero exit with SidecarContainers
// enabled) without a corresponding Unhealthy transition being triggered.
//
// Since resync on the controller is every 30 seconds, even if there is some time in which a GameServer
// is in a broken state, it will eventually move to Unhealthy, and get replaced (if in a Fleet).
type MissingPodController struct {
	baseLogger       *logrus.Entry
	podSynced        cache.InformerSynced
	podLister        corelisterv1.PodLister
	gameServerSynced cache.InformerSynced
	gameServerGetter getterv1.GameServersGetter
	gameServerLister listerv1.GameServerLister
	workerqueue      *workerqueue.WorkerQueue
	recorder         record.EventRecorder
}

// NewMissingPodController returns a MissingPodController
func NewMissingPodController(health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) *MissingPodController {
	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	gameServers := agonesInformerFactory.Agones().V1().GameServers()

	c := &MissingPodController{
		podSynced:        podInformer.HasSynced,
		podLister:        kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerSynced: gameServers.Informer().HasSynced,
		gameServerGetter: agonesClient.AgonesV1(),
		gameServerLister: gameServers.Lister(),
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueue(c.syncGameServer, c.baseLogger, logfields.GameServerKey, agones.GroupName+".MissingPodController")
	health.AddLivenessCheck("gameserver-missing-pod-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "missing-pod-controller"})

	_, _ = gameServers.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			gs := newObj.(*agonesv1.GameServer)

			if _, isDev := gs.GetDevAddress(); !isDev && !isBeforePodCreated(gs) && !gs.IsBeingDeleted() &&
				!(gs.Status.State == agonesv1.GameServerStateUnhealthy) && !(gs.Status.State == agonesv1.GameServerStateError) {

				// Enqueue if the pod is missing, not a game server pod, or has failed.
				// If there was an error accessing the Kubernetes control plane then enqueue it, so it can be rechecked when the control plane comes back up.
				if pod, err := c.podLister.Pods(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name); err != nil || !isGameServerPod(pod) || pod.Status.Phase == corev1.PodFailed {
					c.workerqueue.Enqueue(gs)
				}
			}
		},
	})

	return c
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (c *MissingPodController) Run(ctx context.Context, workers int) error {
	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.podSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(ctx, workers)
	return nil
}

func (c *MissingPodController) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerKey, key)
}

// syncGameServer checks if a GameServer has a backing Pod, and if not,
// moves it to Unhealthy
func (c *MissingPodController) syncGameServer(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(c.loggerForGameServerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	// check if the pod exists and is healthy
	podFailed := false
	if pod, err := c.podLister.Pods(namespace).Get(name); err != nil {
		if !k8serrors.IsNotFound(err) {
			return errors.Wrapf(err, "error retrieving Pod %s from namespace %s", name, namespace)
		}
	} else if isGameServerPod(pod) {
		if pod.Status.Phase != corev1.PodFailed {
			// pod exists and is not failed - all is well
			return nil
		}
		podFailed = true
	}
	if podFailed {
		c.loggerForGameServerKey(key).Debug("Pod has failed. Moving GameServer to Unhealthy.")
	} else {
		c.loggerForGameServerKey(key).Debug("Pod is missing. Moving GameServer to Unhealthy.")
	}

	gs, err := c.gameServerLister.GameServers(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.loggerForGameServerKey(key).Debug("GameServer is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", name, namespace)
	}

	// already on the way out, so no need to do anything.
	if gs.IsBeingDeleted() || gs.Status.State == agonesv1.GameServerStateUnhealthy {
		c.loggerForGameServerKey(key).WithField("state", gs.Status.State).Debug("GameServer already being deleted/unhealthy. Skipping.")
		return nil
	}

	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateUnhealthy
	gs, err = c.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "error updating GameServer to Unhealthy")
	}

	eventMessage := "Pod is missing"
	if podFailed {
		eventMessage = "Pod has failed"
	}
	c.recorder.Event(gs, corev1.EventTypeWarning, string(gs.Status.State), eventMessage)
	return nil
}
