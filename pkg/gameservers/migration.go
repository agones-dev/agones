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
	"strings"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
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
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisterv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// MigrationController watches for if a Pod is migrated/a maintenance
// event happens on a node, and a Pod is recreated with a new Address for a
// GameServer
type MigrationController struct {
	baseLogger               *logrus.Entry
	podSynced                cache.InformerSynced
	podLister                corelisterv1.PodLister
	gameServerSynced         cache.InformerSynced
	gameServerGetter         getterv1.GameServersGetter
	gameServerLister         listerv1.GameServerLister
	nodeLister               corelisterv1.NodeLister
	nodeSynced               cache.InformerSynced
	workerqueue              *workerqueue.WorkerQueue
	recorder                 record.EventRecorder
	syncPodPortsToGameServer func(*agonesv1.GameServer, *corev1.Pod) error
}

// NewMigrationController returns a MigrationController
func NewMigrationController(health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	agonesClient versioned.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory,
	syncPodPortsToGameServer func(*agonesv1.GameServer, *corev1.Pod) error,
) *MigrationController {

	podInformer := kubeInformerFactory.Core().V1().Pods().Informer()
	gameserverInformer := agonesInformerFactory.Agones().V1().GameServers()
	mc := &MigrationController{
		podSynced:                podInformer.HasSynced,
		podLister:                kubeInformerFactory.Core().V1().Pods().Lister(),
		gameServerSynced:         gameserverInformer.Informer().HasSynced,
		gameServerGetter:         agonesClient.AgonesV1(),
		gameServerLister:         gameserverInformer.Lister(),
		nodeLister:               kubeInformerFactory.Core().V1().Nodes().Lister(),
		nodeSynced:               kubeInformerFactory.Core().V1().Nodes().Informer().HasSynced,
		syncPodPortsToGameServer: syncPodPortsToGameServer,
	}

	mc.baseLogger = runtime.NewLoggerWithType(mc)
	mc.workerqueue = workerqueue.NewWorkerQueue(mc.syncGameServer, mc.baseLogger, logfields.GameServerKey, agones.GroupName+".MigrationController")
	health.AddLivenessCheck("gameserver-migration-workerqueue", healthcheck.Check(mc.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(mc.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	mc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "migration-controller"})

	_, _ = podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			if _, _, ok, err := mc.isMigratingGameServerPod(pod); err != nil || ok {
				mc.workerqueue.Enqueue(pod)
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			pod := newObj.(*corev1.Pod)
			if _, _, ok, err := mc.isMigratingGameServerPod(pod); err != nil || ok {
				mc.workerqueue.Enqueue(pod)
			}
		},
	})
	return mc
}

// Run processes the rate limited queue.
// Will block until stop is closed
func (mc *MigrationController) Run(ctx context.Context, workers int) error {
	mc.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), mc.nodeSynced, mc.gameServerSynced, mc.podSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	mc.workerqueue.Run(ctx, workers)
	return nil
}

func (mc *MigrationController) loggerForGameServerKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(mc.baseLogger, logfields.GameServerKey, key)
}

func (mc *MigrationController) loggerForGameServer(gs *agonesv1.GameServer) *logrus.Entry {
	gsName := logfields.NilGameServer
	if gs != nil {
		gsName = gs.Namespace + "/" + gs.Name
	}
	return mc.loggerForGameServerKey(gsName).WithField("gs", gs)
}

func (mc *MigrationController) isMigratingGameServerPod(pod *k8sv1.Pod) (*agonesv1.GameServer, *k8sv1.Node, bool, error) {
	if pod.Spec.NodeName == "" || !pod.ObjectMeta.DeletionTimestamp.IsZero() || !isGameServerPod(pod) {
		return nil, nil, false, nil
	}

	key := pod.Namespace + "/" + pod.Name
	gs, err := mc.gameServerLister.GameServers(pod.ObjectMeta.Namespace).Get(pod.ObjectMeta.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			mc.loggerForGameServerKey(key).Debug("GameServer is no longer available for syncing")
			return nil, nil, false, nil
		}
		return nil, nil, false, errors.Wrapf(err, "error retrieving GameServer %s from namespace %s", pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
	}

	// Either the address has not been set, or we're being deleted already
	if gs.Status.NodeName == "" || gs.IsBeingDeleted() || gs.Status.State == agonesv1.GameServerStateUnhealthy {
		return nil, nil, false, nil
	}

	if pod.Spec.NodeName == "" {
		return nil, nil, false, workerqueue.NewTraceError(errors.Errorf("node not yet populated for Pod %s", pod.ObjectMeta.Name))
	}

	node, err := mc.nodeLister.Get(pod.Spec.NodeName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			mc.loggerForGameServerKey(key).WithField("node", pod.Spec.NodeName).Debug("Node is no longer available for syncing")
			return nil, nil, false, nil
		}
		return nil, nil, false, errors.Wrapf(err, "error retrieving node %s for Pod %s", pod.Spec.NodeName, pod.ObjectMeta.Name)
	}

	// if the node is being terminated, then also escape, because the Pod is going to be Terminated if it hasn't been
	// already.
	if !node.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil, nil, false, nil
	}

	// if the nodes match, and the default GameServer Address matches one of the node addresses - escape, since
	// migration isn't happening.
	if pod.Spec.NodeName == gs.Status.NodeName && mc.anyAddressMatch(node, gs) {
		return nil, nil, false, nil
	}

	return gs, node, true, nil
}

// syncGameServer will check if the Pod for the GameServer
// has been migrated to a new node (or a node with the same name, but different address)
// and will either update it, or move it to Unhealthy, depending on its State
func (mc *MigrationController) syncGameServer(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(mc.loggerForGameServerKey(key), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	pod, err := mc.podLister.Pods(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			mc.loggerForGameServerKey(key).Debug("Pod is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving Pod %s from namespace %s", name, namespace)
	}

	gs, node, ok, err := mc.isMigratingGameServerPod(pod)
	// if there is an error, retry, but if not migrating then escape and continue on with your life doing other things.
	if err != nil || !ok {
		return err
	}

	// If the GameServer has yet to become ready, we will reapply the Address and Port
	// otherwise, we move it to Unhealthy so that a new GameServer will be recreated.
	gsCopy := gs.DeepCopy()
	var eventMsg string
	if gsCopy.IsBeforeReady() {
		gsCopy, err = applyGameServerAddressAndPort(gsCopy, node, pod, mc.syncPodPortsToGameServer)
		if err != nil {
			return err
		}
		eventMsg = "Address updated due to Node migration"
	} else {
		gsCopy.Status.State = agonesv1.GameServerStateUnhealthy
		eventMsg = "Node migration occurred"
	}

	if gs, err = mc.gameServerGetter.GameServers(gsCopy.ObjectMeta.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{}); err != nil {
		return err
	}

	mc.loggerForGameServer(gs).Debug("GameServer migration occurred")
	mc.recorder.Event(gs, corev1.EventTypeWarning, string(gsCopy.Status.State), eventMsg)

	return nil
}

func (mc *MigrationController) anyAddressMatch(node *k8sv1.Node, gs *agonesv1.GameServer) bool {
	var nodeAddresses []string
	for _, a := range node.Status.Addresses {
		if a.Address == gs.Status.Address {
			return true
		}
		nodeAddresses = append(nodeAddresses, a.Address)
	}
	mc.loggerForGameServer(gs).
		WithField("gs", gs.Name).
		WithField("gs.Status.Address", gs.Status.Address).
		WithField("node.Status.Addresses", strings.Join(nodeAddresses, ",")).
		Warn("GameServer/Node address mismatch")
	return false
}
