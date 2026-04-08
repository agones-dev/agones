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

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	e2eframework "agones.dev/agones/test/e2e/framework"
)

const (
	fakeIPAddress = "192.1.1.2"
)

func TestCreateConnect(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, len(readyGs.Status.Ports), 1)
	assert.NotEmpty(t, readyGs.Status.Ports[0].Port)
	assert.NotEmpty(t, readyGs.Status.Address)
	assert.NotEmpty(t, readyGs.Status.Addresses)

	var hasPodIPAddress bool
	for i, addr := range readyGs.Status.Addresses {
		if addr.Type == agonesv1.NodePodIP {
			assert.NotEmpty(t, readyGs.Status.Addresses[i].Address)
			hasPodIPAddress = true
		}
	}
	assert.True(t, hasPodIPAddress)

	assert.NotEmpty(t, readyGs.Status.NodeName)
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello World !")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

func TestHostName(t *testing.T) {
	t.Parallel()

	pods := framework.KubeClient.CoreV1().Pods(framework.Namespace)

	fixtures := map[string]struct {
		setup func(gs *agonesv1.GameServer)
		test  func(gs *agonesv1.GameServer, pod *corev1.Pod)
	}{
		"standard hostname": {
			setup: func(_ *agonesv1.GameServer) {},
			test: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				assert.Equal(t, gs.ObjectMeta.Name, pod.Spec.Hostname)
			},
		},
		"a . in the name": {
			setup: func(gs *agonesv1.GameServer) {
				gs.ObjectMeta.GenerateName = "game-server-1.0-"
			},
			test: func(_ *agonesv1.GameServer, pod *corev1.Pod) {
				expected := "game-server-1-0-"
				// since it's a generated name, we just check the beginning.
				assert.Equal(t, expected, pod.Spec.Hostname[:len(expected)])
			},
		},
		// generated name will automatically truncate to 63 chars.
		"generated with > 63 chars": {
			setup: func(gs *agonesv1.GameServer) {
				gs.ObjectMeta.GenerateName = "game-server-" + strings.Repeat("n", 100)
			},
			test: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				assert.Equal(t, gs.ObjectMeta.Name, pod.Spec.Hostname)
			},
		},
		// Note: no need to test for a gs.ObjectMeta.Name > 63 chars, as it will be rejected as invalid
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			gs := framework.DefaultGameServer(framework.Namespace)
			gs.Spec.Template.Spec.Subdomain = "default"
			v.setup(gs)
			readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
			require.NoError(t, err)
			pod, err := pods.Get(context.Background(), readyGs.ObjectMeta.Name, metav1.GetOptions{})
			require.NoError(t, err)
			v.test(readyGs, pod)
		})
	}
}

// nolint:dupl
func TestSDKSetLabel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "LABEL")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: LABEL\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollUntilContextTimeout(ctx, time.Second, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return gs.ObjectMeta.Labels != nil, nil
	})

	if assert.NoError(t, err) {
		assert.NotEmpty(t, gs.ObjectMeta.Labels["agones.dev/sdk-timestamp"])
	}
}

func TestHealthCheckDisable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Health = agonesv1.Health{
		Disabled:            true,
		FailureThreshold:    1,
		InitialDelaySeconds: 1,
		PeriodSeconds:       1,
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	_, err = framework.SendGameServerUDP(t, readyGs, "UNHEALTHY")
	if err != nil {
		t.Fatalf("Could not ping GameServer: %v", err)
	}

	time.Sleep(10 * time.Second)

	gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	if err != nil {
		assert.FailNow(t, "gameserver get failed", err.Error())
	}

	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

// nolint:dupl
func TestSDKSetAnnotation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	annotation := "agones.dev/sdk-timestamp"
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "ANNOTATION")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ANNOTATION\n", reply)

	// the label is set in a queue, so it may take a moment
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		_, ok := gs.ObjectMeta.Annotations[annotation]
		return ok, nil
	})

	logrus.WithField("annotations", gs.ObjectMeta.Annotations).Info("annotation information")

	if !assert.Nil(t, err) {
		assert.FailNow(t, "error waiting on annotation to be set")
	}
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[annotation])
	assert.NotEmpty(t, gs.ObjectMeta.Annotations[agonesv1.VersionAnnotation])
}

func TestUnhealthyGameServerAfterHealthCheckFail(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Health.FailureThreshold = 1

	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Failed to create a gameserver", err.Error())
	}

	reply, err := framework.SendGameServerUDP(t, gs, "UNHEALTHY")
	if err != nil {
		assert.FailNow(t, "Failed to send a message to a gameserver", err.Error())
	}
	assert.Equal(t, "ACK: UNHEALTHY\n", reply)

	_, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateUnhealthy, time.Minute)
	assert.NoError(t, err)
}

func TestUnhealthyGameServersWithoutFreePorts(t *testing.T) {
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Static PortPolicy")
	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		t.SkipNow()
	}

	t.Parallel()
	ctx := context.Background()
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		assert.FailNow(t, "Failed to list nodes", err.Error())
	}
	assert.True(t, len(nodes.Items) > 0)

	template := framework.DefaultGameServer(framework.Namespace)
	// choose port out of the minport/maxport range
	// also rand it, just in case there are still previous static GameServers floating around.
	template.Spec.Ports[0].HostPort = int32(rand.IntnRange(4000, 5000))
	template.Spec.Ports[0].PortPolicy = agonesv1.Static

	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	// one successful static port GameServer
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, template.DeepCopy())
	require.NoError(t, err)

	// now let's create the same one, but this time, require it be on the same node.
	newGs := template.DeepCopy()

	if newGs.Spec.Template.Spec.NodeSelector == nil {
		newGs.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	newGs.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = gs.Status.NodeName
	newGs, err = gameServers.Create(ctx, newGs, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 5*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerUnhealthyAfterDeletingPod(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	logrus.WithField("gsKey", readyGs.ObjectMeta.Name).Info("GameServer Ready")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	podClient := framework.KubeClient.CoreV1().Pods(framework.Namespace)

	defer gsClient.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	pod, err := podClient.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, metav1.IsControlledBy(pod, readyGs))

	err = podClient.Delete(ctx, pod.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)

	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
	require.NoError(t, err)
}

func TestGameServerRestartBeforeReadyCrash(t *testing.T) {
	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		t.SkipNow()
	}

	t.Parallel()
	ctx := context.Background()
	logger := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)
	// give some buffer with gameservers crashing and coming back
	gs.Spec.Health.PeriodSeconds = 60 * 60
	gs.Spec.Template.Spec.Containers[0].Env = append(gs.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "READY", Value: "FALSE"})
	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	newGs, err := gsClient.Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		assert.Fail(t, "could not create the gameserver", err.Error())
	}
	defer gsClient.Delete(ctx, newGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	logger.Info("Waiting for us to have an address to send things to")
	newGs, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateScheduled, framework.WaitForState)
	if err != nil {
		assert.FailNow(t, "Failed schedule a pod", err.Error())
	}

	logger.WithField("gs", newGs.ObjectMeta.Name).Info("GameServer created")

	address := fmt.Sprintf("%s:%d", newGs.Status.Address, newGs.Status.Ports[0].Port)
	logger.WithField("address", address).Info("Dialing UDP message to address")

	messageAndWait := func(gs *agonesv1.GameServer, msg string, check func(gs *agonesv1.GameServer, pod *corev1.Pod) bool) error {
		return wait.PollUntilContextTimeout(context.Background(), 200*time.Millisecond, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			gs, err := gsClient.Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get gameserver")
				return true, err
			}
			pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Get(ctx, newGs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				logger.WithError(err).Warn("could not get pod for gameserver")
				return true, err
			}

			if check(gs, pod) {
				return true, nil
			}

			// create a connection each time, as weird stuff happens if the receiver isn't up and running.
			conn, err := net.Dial("udp", address)
			if err != nil {
				logger.WithError(err).Warn("could not create connection")
				return true, err
			}
			defer conn.Close() // nolint: errcheck
			// doing this last, so that there is a short delay between the msg being sent, and the check.
			logger.WithField("gs", gs.ObjectMeta.Name).WithField("msg", msg).
				WithField("state", gs.Status.State).Info("sending message")
			if _, err = conn.Write([]byte(msg)); err != nil {
				logger.WithError(err).WithField("gs", gs.ObjectMeta.Name).
					WithField("state", gs.Status.State).Info("error sending packet")
			}
			return false, nil
		})
	}

	logger.Info("crashing, and waiting to see restart")
	err = messageAndWait(newGs, "CRASH", func(_ *agonesv1.GameServer, pod *corev1.Pod) bool {
		for _, c := range pod.Status.ContainerStatuses {
			if c.Name == newGs.Spec.Container && c.RestartCount > 0 {
				logger.Info("successfully crashed. Moving on!")
				return true
			}
		}
		return false
	})
	assert.NoError(t, err)

	// check that the GameServer is not in an unhealthy state. If it does happen, it should happen pretty quick.
	// We wait an extra 5s to close the kubelet race in #2445.
	newGs, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 10*time.Second)
	// should be an error, as the state should not occur
	if !assert.Error(t, err) {
		assert.FailNow(t, "GameServer should not be Unhealthy")
	}
	assert.Contains(t, err.Error(), "waiting for GameServer")

	// ping READY until it doesn't fail anymore - since it may take a while
	// for this to come back up -- or we could get a delayed CRASH, so we have to
	// wait for the process to restart again to fire the SDK.Ready()
	logger.Info("marking GameServer as ready")
	err = messageAndWait(newGs, "READY", func(gs *agonesv1.GameServer, _ *corev1.Pod) bool {
		if gs.Status.State == agonesv1.GameServerStateReady {
			logger.Info("ready! Moving On!")
			return true
		}
		return false
	})
	assert.NoError(t, err)

	// now crash, should be unhealthy, since it's after being Ready
	logger.Info("crashing again, should be unhealthy")
	// retry on crash, as with the restarts, sometimes Go takes a moment to send this through.
	err = messageAndWait(newGs, "CRASH", func(gs *agonesv1.GameServer, _ *corev1.Pod) bool {
		logger.WithField("gs", gs.ObjectMeta.Name).WithField("state", gs.Status.State).
			Info("checking final crash state")
		if gs.Status.State == agonesv1.GameServerStateUnhealthy {
			logger.Info("Unhealthy! We are done!")
			return true
		}
		return false
	})
	assert.NoError(t, err)
}

func TestGameServerUnhealthyAfterReadyCrash(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	log := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	log.WithField("gs", readyGs.ObjectMeta.Name).Info("GameServer created")

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	defer gsClient.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	address := fmt.Sprintf("%s:%d", readyGs.Status.Address, readyGs.Status.Ports[0].Port)

	// keep crashing, until we move to Unhealthy. Solves potential issues with controller Informer cache
	// race conditions in which it has yet to see a GameServer is Ready before the crash.
	var stop int32
	defer func() {
		atomic.StoreInt32(&stop, 1)
	}()
	go func() {
		for {
			if atomic.LoadInt32(&stop) > 0 {
				log.Info("UDP Crash stop signal received. Stopping.")
				return
			}
			var writeErr error
			func() {
				conn, err := net.Dial("udp", address)
				assert.NoError(t, err)
				defer conn.Close() // nolint: errcheck
				_, writeErr = conn.Write([]byte("CRASH"))
			}()
			if writeErr != nil {
				log.WithError(err).Warn("error sending udp packet. Stopping.")
				return
			}
			log.WithField("address", address).Info("sent UDP packet")
			time.Sleep(5 * time.Second)
		}
	}()
	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateUnhealthy, 3*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerPodCompletedAfterCleanExit(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		t.SkipNow()
	}

	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}

	address := fmt.Sprintf("%s:%d", readyGs.Status.Address, readyGs.Status.Ports[0].Port)
	conn, err := net.Dial("udp", address)
	mtx := &sync.Mutex{}
	require.NoError(t, err)
	defer func() {
		mtx.Lock()
		defer mtx.Unlock()
		if conn != nil {
			err = conn.Close()
			if err != nil {
				log.WithError(err).Warn("error closing udp connection")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				mtx.Lock()
				log.Info("Sending CRASH 0")
				_, err := conn.Write([]byte("CRASH 0"))
				mtx.Unlock()
				if err != nil {
					log.WithError(err).Warn("error sending udp packet. Stopping.")
					return
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()

	err = wait.PollUntilContextTimeout(ctx, 3*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		log.WithField("gs", readyGs.ObjectMeta.Name).WithField("state", gs.Status.State).WithError(err).Info("checking if GameServer exists")
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	require.NoError(t, err)
}

func TestDevelopmentGameServerLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	labels := map[string]string{"development": "true"}
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "udp-server",
			Namespace:    framework.Namespace,
			Annotations:  map[string]string{agonesv1.DevAddressAnnotation: fakeIPAddress},
			Labels:       labels,
		},
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				HostPort:      7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Static,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "placebo",
						Image: "this is ignored",
					}},
				},
			},
		},
	}
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs.DeepCopy())
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	require.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)

	// Set dev GS into RequestReady and confirm it goes back to Ready.
	gsCopy := readyGs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateRequestReady
	reqReadyGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Equal(t, agonesv1.GameServerStateRequestReady, reqReadyGs.Status.State)

	readyGs, err = framework.WaitForGameServerState(t, reqReadyGs, agonesv1.GameServerStateReady, framework.WaitForState)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready from request ready: %v", err)
	}
	require.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)

	// confirm delete works, because if the finalisers don't get removed, this won't work.
	err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)

	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	require.NoError(t, err)

	// let's make sure we can allocate a dev gameserver
	readyGs, err = framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs.DeepCopy())
	require.NoError(t, err)

	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{
				LabelSelector: metav1.LabelSelector{MatchLabels: labels},
			}},
		},
	}
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)

	require.Equal(t, readyGs.ObjectMeta.Name, gsa.Status.GameServerName)

	_, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	require.NoError(t, err)

	// Also confirm that delete works for Allocated state, it should be fine but let's be sure.
	err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)

	err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	require.NoError(t, err)
}

func TestGameServerSelfAllocate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	reply, err := framework.SendGameServerUDP(t, readyGs, "ALLOCATE")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	if assert.NoError(t, err) {
		assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	}
}

func TestGameServerReadyAllocateReady(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	logger := e2eframework.TestLogger(t)

	gs := framework.DefaultGameServer(framework.Namespace)

	logger.Info("Moving to Ready")
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err, "Could not get a GameServer ready")
	logger = logger.WithField("gs", readyGs.ObjectMeta.Name)

	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	require.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	logger.Info("Moving to Allocated")
	reply, err := framework.SendGameServerUDP(t, readyGs, "ALLOCATE")
	require.NoError(t, err, "Could not message GameServer")

	require.Equal(t, "ACK: ALLOCATE\n", reply)
	gs, err = framework.WaitForGameServerState(t, readyGs, agonesv1.GameServerStateAllocated, time.Minute)
	require.NoError(t, err)
	require.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)

	logger.Info("Moving to Ready again")
	reply, err = framework.SendGameServerUDP(t, readyGs, "READY")
	require.NoError(t, err, "Could not message GameServer")
	require.Equal(t, "ACK: READY\n", reply)
	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReady, time.Minute)
	require.NoError(t, err)
	require.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
}

func TestGameServerWithPortsMappedToMultipleContainers(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	firstContainerName := "udp-server"
	secondContainerName := "second-udp-server"
	firstPort := "gameport"
	secondPort := "second-gameport"
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: framework.Namespace},
		Spec: agonesv1.GameServerSpec{
			Container: firstContainerName,
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          firstPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}, {
				ContainerPort: 5000,
				Name:          secondPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
				Container:     &secondContainerName,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            firstContainerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
						{
							Name:            secondContainerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            []string{"-port", "5000"},
						},
					},
				},
			},
		},
	}

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	interval := 2 * time.Second
	timeOut := 60 * time.Second

	expectedMsg1 := "Ping 1"
	errPoll := wait.PollUntilContextTimeout(context.Background(), interval, timeOut, true, func(_ context.Context) (done bool, err error) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, firstPort, expectedMsg1)
		if err != nil {
			t.Logf("Could not message GameServer on %s: %v. Will try again...", firstPort, err)
		}
		return err == nil && strings.Contains(res, expectedMsg1), nil
	})
	if errPoll != nil {
		assert.FailNow(t, errPoll.Error(), "expected no errors after polling a port: %s", firstPort)
	}

	expectedMsg2 := "Ping 2"
	errPoll = wait.PollUntilContextTimeout(context.Background(), interval, timeOut, true, func(_ context.Context) (done bool, err error) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, secondPort, expectedMsg2)
		if err != nil {
			t.Logf("Could not message GameServer on %s: %v. Will try again...", secondPort, err)
		}
		return err == nil && strings.Contains(res, expectedMsg2), nil
	})

	assert.NoError(t, errPoll, "expected no errors after polling a port: %s", secondPort)
}

func TestGameServerWithPortsMappedToInitSidecarContainers(t *testing.T) {
	t.Skip("Skip test. Only works on some CNIs. The current e2e platform does not support ports on init containers.")

	t.Parallel()
	ctx := context.Background()

	containerName := "udp-server"
	sidecarContainerName := "sidecar-server"
	firstPort := "gameport"
	secondPort := "second-gameport"
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "udp-server", Namespace: framework.Namespace},
		Spec: agonesv1.GameServerSpec{
			Container: containerName,
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          firstPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}, {
				ContainerPort: 5000,
				Name:          secondPort,
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
				Container:     &sidecarContainerName,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            containerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            sidecarContainerName,
							Image:           framework.GameServerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            []string{"-port", "5000"},
							RestartPolicy:   ptr.To(corev1.ContainerRestartPolicyAlways),
						},
					},
				},
			},
		},
	}

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	interval := 2 * time.Second
	timeOut := 60 * time.Second

	expectedMsg1 := "Ping 1"
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, firstPort, expectedMsg1)
		require.NoError(c, err, "expected no errors after messaging GameServer a port: %s", firstPort)
		assert.Contains(c, res, expectedMsg1)
	}, timeOut, interval)

	expectedMsg2 := "Ping 2"
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		res, err := framework.SendGameServerUDPToPort(t, readyGs, secondPort, expectedMsg2)
		require.NoError(c, err, "expected no errors after messaging GameServer a port: %s", secondPort)
		assert.Contains(c, res, expectedMsg2)
	}, timeOut, interval)
}

func TestGameServerReserve(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// We are deliberately not trying to test the transition between Reserved -> Ready.
	//
	// We have found that trying to catch the GameServer in the Reserved state can be flaky,
	// as we can't control the speed in which the Kubernetes API is going to reply to request,
	// and we could sometimes miss when the GameServer is in the Reserved State before it goes to Ready.
	//
	// Therefore we are going to test for concrete states that we don't need to catch while
	// in a transitive state.

	gs := framework.DefaultGameServer(framework.Namespace)
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, gs, "RESERVE 0")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: RESERVE 0\n", reply)

	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReserved, 3*time.Minute)
	if err != nil {
		assert.FailNow(t, "Time out on waiting for gs in a Reserved state", err.Error())
	}

	reply, err = framework.SendGameServerUDP(t, gs, "ALLOCATE")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: ALLOCATE\n", reply)

	// put it in a totally different state, just to reset things.
	gs, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateAllocated, 3*time.Minute)
	if err != nil {
		assert.FailNow(t, "Time out on waiting for gs in an Allocated state", err.Error())
	}

	reply, err = framework.SendGameServerUDP(t, gs, "RESERVE 5s")
	if err != nil {
		assert.FailNow(t, "Could not message GameServer", err.Error())
	}
	assert.Equal(t, "ACK: RESERVE 5s\n", reply)

	// sleep, since we're going to wait for the Ready response.
	time.Sleep(5 * time.Second)
	_, err = framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateReady, 3*time.Minute)
	assert.NoError(t, err)
}

func TestGameServerShutdown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	reply, err := framework.SendGameServerUDP(t, readyGs, "EXIT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}

	assert.Equal(t, "ACK: EXIT\n", reply)

	err = wait.PollUntilContextTimeout(ctx, time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})

		if k8serrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})

	assert.NoError(t, err)
}

// TestGameServerEvicted test that if Gameserver would be evicted than it becomes Unhealthy
// Ephemeral Storage limit set to 0Mi
func TestGameServerEvicted(t *testing.T) {
	t.Parallel()
	log := e2eframework.TestLogger(t)

	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	newGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	log.WithField("name", newGs.ObjectMeta.Name).Info("GameServer created, waiting for being Evicted and Unhealthy")
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, newGs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck

	pods := framework.KubeClient.CoreV1().Pods(framework.Namespace)
	pod, err := pods.Get(ctx, newGs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)

	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
	}
	go func() {
		time.Sleep(3 * time.Second) // just make sure it comes in later
		log.WithField("name", eviction.ObjectMeta.Name).Info("Evicting pod!")
		err := pods.EvictV1(context.Background(), eviction)
		require.NoError(t, err)
	}()

	_, err = framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateUnhealthy, 10*time.Minute)
	require.NoError(t, err, fmt.Sprintf("waiting for [%v] GameServer Unhealthy state timed out (%v)", gs.Status.State, gs.Name))
}

func TestGameServerPassthroughPort(t *testing.T) {
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Passthrough PortPolicy")
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports[0] = agonesv1.GameServerPort{PortPolicy: agonesv1.Passthrough}
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "PASSTHROUGH", Value: "TRUE"}}
	// gate
	errs := gs.Validate(agtesting.FakeAPIHooks{})
	assert.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}

	port := readyGs.Spec.Ports[0]
	assert.Equal(t, agonesv1.Passthrough, port.PortPolicy)
	assert.NotEmpty(t, port.HostPort)
	assert.Equal(t, port.HostPort, port.ContainerPort)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello World !")
	if err != nil {
		t.Fatalf("Could ping GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", reply)
}

func TestGameServerPortPolicyNone(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePortPolicyNone) {
		t.SkipNow()
	}

	t.Parallel()

	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports[0] = agonesv1.GameServerPort{PortPolicy: agonesv1.None, ContainerPort: 7777}
	// gate
	errs := gs.Validate(agtesting.FakeAPIHooks{})
	assert.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}

	// To test sending UDP traffic directly to a pod when no hostPort is set, a product like Calico which uses BGP is needed
	// so this only tests that the port is set correctly.
	port := readyGs.Spec.Ports[0]
	assert.Equal(t, agonesv1.None, port.PortPolicy)
	assert.Empty(t, port.HostPort)
	assert.EqualValues(t, 7777, port.ContainerPort)
}

func TestGameServerTcpProtocol(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].Protocol = corev1.ProtocolTCP
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	replyTCP, err := e2eframework.SendGameServerTCP(readyGs, "Hello World !")
	require.NoError(t, err)
	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)
}

func TestGameServerTcpUdpProtocol(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].Protocol = agonesv1.ProtocolTCPUDP
	gs.Spec.Ports[0].Name = "gameserver"
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		assert.FailNow(t, "Could not get a GameServer ready", err.Error())
	}

	tcpPort := readyGs.Spec.Ports[0]
	assert.Equal(t, corev1.ProtocolTCP, tcpPort.Protocol)
	assert.NotEmpty(t, tcpPort.HostPort)
	assert.Equal(t, "gameserver-tcp", tcpPort.Name)

	udpPort := readyGs.Spec.Ports[1]
	assert.Equal(t, corev1.ProtocolUDP, udpPort.Protocol)
	assert.NotEmpty(t, udpPort.HostPort)
	assert.Equal(t, "gameserver-udp", udpPort.Name)

	assert.Equal(t, tcpPort.HostPort, udpPort.HostPort)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("GameServer created, sending UDP ping")

	replyUDP, err := framework.SendGameServerUDPToPort(t, readyGs, udpPort.Name, "Hello World !")
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("Could not ping UDP GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", replyUDP)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("UDP ping passed, sending TCP ping")

	replyTCP, err := e2eframework.SendGameServerTCPToPort(readyGs, tcpPort.Name, "Hello World !")
	if err != nil {
		t.Fatalf("Could not ping TCP GameServer: %v", err)
	}

	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)
}

// TestGameServerStaticTcpUdpProtocol checks UDP and TCP connection for TCPUDP Protocol of Static Portpolicy
func TestGameServerStaticTcpUdpProtocol(t *testing.T) {
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Static PortPolicy")
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports[0].PortPolicy = agonesv1.Static
	gs.Spec.Ports[0].Protocol = agonesv1.ProtocolTCPUDP
	gs.Spec.Ports[0].HostPort = 7000
	gs.Spec.Ports[0].Name = "gameserver"
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	tcpPort := readyGs.Spec.Ports[0]
	assert.Equal(t, corev1.ProtocolTCP, tcpPort.Protocol)
	assert.NotEmpty(t, tcpPort.HostPort)
	assert.Equal(t, "gameserver-tcp", tcpPort.Name)

	udpPort := readyGs.Spec.Ports[1]
	assert.Equal(t, corev1.ProtocolUDP, udpPort.Protocol)
	assert.NotEmpty(t, udpPort.HostPort)
	assert.Equal(t, "gameserver-udp", udpPort.Name)

	assert.Equal(t, tcpPort.HostPort, udpPort.HostPort)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("GameServer created, sending UDP ping")

	replyUDP, err := framework.SendGameServerUDPToPort(t, readyGs, udpPort.Name, "Hello World !")
	require.NoError(t, err)
	if err != nil {
		t.Fatalf("Could not ping UDP GameServer: %v", err)
	}

	assert.Equal(t, "ACK: Hello World !\n", replyUDP)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("UDP ping passed, sending TCP ping")

	replyTCP, err := e2eframework.SendGameServerTCPToPort(readyGs, tcpPort.Name, "Hello World !")
	if err != nil {
		t.Fatalf("Could not ping TCP GameServer: %v", err)
	}

	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)
}

// TestGameServerStaticTcpProtocol checks TCP connection for TCP Protocol of Static Portpolicy
func TestGameServerStaticTcpProtocol(t *testing.T) {
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Static PortPolicy")
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].PortPolicy = agonesv1.Static
	gs.Spec.Ports[0].Protocol = corev1.ProtocolTCP
	gs.Spec.Ports[0].HostPort = 7000
	gs.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "TCP", Value: "TRUE"}}

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("sending TCP ping")

	replyTCP, err := e2eframework.SendGameServerTCP(readyGs, "Hello World !")
	require.NoError(t, err)
	assert.Equal(t, "ACK TCP: Hello World !\n", replyTCP)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("TCP ping Passed")
}

// TestGameServerStaticUdpProtocol checks default(UDP) connection of Static Portpolicy
func TestGameServerStaticUdpProtocol(t *testing.T) {
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Static PortPolicy")
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Ports[0].PortPolicy = agonesv1.Static
	gs.Spec.Ports[0].HostPort = 7000

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("sending UDP ping")

	replyTCP, err := framework.SendGameServerUDP(t, readyGs, "Default UDP connection check")
	require.NoError(t, err)
	assert.Equal(t, "ACK: Default UDP connection check\n", replyTCP)

	logrus.WithField("name", readyGs.ObjectMeta.Name).Info("UDP ping Passed")
}

func TestGameServerWithoutPort(t *testing.T) {
	t.Parallel()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Ports = nil

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	assert.Len(t, errs, 0)

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)

	require.NoError(t, err, "Could not get a GameServer ready")
	assert.Empty(t, readyGs.Spec.Ports)
}

// TestGameServerResourceValidation - check that we are not able to use
// invalid PodTemplate for GameServer Spec with wrong Resource Requests and Limits
func TestGameServerResourceValidation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	mi64 := resource.MustParse("64Mi")
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = resource.MustParse("128Mi")

	errs := gs.Validate(agtesting.FakeAPIHooks{})
	assert.False(t, len(errs) == 0)

	gsClient := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)

	_, err := gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	assert.NotNil(t, err)
	statusErr, ok := err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.Len(t, statusErr.Status().Details.Causes, 1)
	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, statusErr.Status().Details.Causes[0].Type)
	assert.Equal(t, "spec.template.spec.containers[0].resources.requests", statusErr.Status().Details.Causes[0].Field)

	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("-50m")
	_, err = gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	assert.NotNil(t, err)
	statusErr, ok = err.(*k8serrors.StatusError)
	assert.True(t, ok)
	assert.Len(t, statusErr.Status().Details.Causes, 2)
	sort.Slice(statusErr.Status().Details.Causes, func(i, j int) bool {
		return statusErr.Status().Details.Causes[i].Field > statusErr.Status().Details.Causes[j].Field
	})
	assert.Equal(t, metav1.CauseTypeFieldValueInvalid, statusErr.Status().Details.Causes[0].Type)
	assert.Equal(t, "spec.template.spec.containers[0].resources.requests[cpu]", statusErr.Status().Details.Causes[0].Field)

	// test that values are still being set correctly
	m50 := resource.MustParse("50m")
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory] = mi64
	gs.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = m50
	gs.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = m50

	// confirm we have a valid GameServer before running the test
	errs = gs.Validate(agtesting.FakeAPIHooks{})
	require.Len(t, errs, 0)

	gsCopy, err := gsClient.Create(ctx, gs.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, mi64, gsCopy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory])
	assert.Equal(t, mi64, gsCopy.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory])
	assert.Equal(t, m50, gsCopy.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, m50, gsCopy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU])
}

func TestGameServerPodTemplateSpecFailSchemaValidation(t *testing.T) {
	t.Parallel()

	// The Kubernetes dynamic client skips schema validation (for reasons I'm not sure of), so the
	// best way I could determine to test schema validation is via calling kubectl directly.
	// The schema's from Kubernetes don't include anything like `pattern:` or `enum:` which would
	// potentially make this easier to test.

	gsYaml := `
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  name: "invalid-gameserver"
spec:
  ports:
    - name: default
      portPolicy: Dynamic
      containerPort: 7654
  template:
    spec:
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution: ERROR
      containers:
        - name: simple-game-server
          image: us-docker.pkg.dev/agones-images/examples/simple-game-server:0.40
`
	err := os.WriteFile("/tmp/invalid.yaml", []byte(gsYaml), 0o644)
	require.NoError(t, err)

	cmd := exec.Command("kubectl", "apply", "-f", "/tmp/invalid.yaml")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	logrus.WithField("stdout", stdout.String()).WithField("stderr", stderr.String()).WithError(err).Info("Ran command!")
	require.Error(t, err)
	assert.Contains(t, stderr.String(), "spec.template.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution")
}

func TestGameServerSetPlayerCapacity(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	t.Run("no initial capacity set", func(t *testing.T) {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		if err != nil {
			t.Fatalf("Could not get a GameServer ready: %v", err)
		}
		assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
		assert.Equal(t, int64(0), gs.Status.Players.Capacity)

		reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "0\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY 20")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "ACK: PLAYER_CAPACITY 20\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "20\n", reply)

		err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
			gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return gs.Status.Players.Capacity == 20, nil
		})
		assert.NoError(t, err)
	})

	t.Run("initial capacity set", func(t *testing.T) {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: 10}
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		if err != nil {
			t.Fatalf("Could not get a GameServer ready: %v", err)
		}
		assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
		assert.Equal(t, int64(10), gs.Status.Players.Capacity)

		reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "10\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY 20")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "ACK: PLAYER_CAPACITY 20\n", reply)

		reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CAPACITY")
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, "20\n", reply)

		err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
			gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			return gs.Status.Players.Capacity == 20, nil
		})
		assert.NoError(t, err)

		time.Sleep(30 * time.Second)
	})
}

func TestPlayerConnectWithCapacityZero(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()

	gs := framework.DefaultGameServer(framework.Namespace)
	playerCount := int64(0)
	gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: playerCount}
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
	assert.Equal(t, playerCount, gs.Status.Players.Capacity)

	// add a player
	msg := "PLAYER_CONNECT 1"
	logrus.WithField("msg", msg).Info("Sending Player Connect")
	_, err = framework.SendGameServerUDP(t, gs, msg)
	// expected error from the log.Fatalf("could not connect player: %v", err)
	if assert.Error(t, err) {
		_, err := framework.WaitForGameServerState(t, gs, agonesv1.GameServerStateUnhealthy, time.Minute)
		assert.NoError(t, err)
	}
}

func TestPlayerConnectAndDisconnect(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	gs := framework.DefaultGameServer(framework.Namespace)
	playerCount := int64(3)
	gs.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: playerCount}
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, gs.Status.State, agonesv1.GameServerStateReady)
	assert.Equal(t, playerCount, gs.Status.Players.Capacity)

	// add three players in quick succession
	for i := int64(1); i <= playerCount; i++ {
		msg := "PLAYER_CONNECT " + fmt.Sprintf("%d", i)
		logrus.WithField("msg", msg).Info("Sending Player Connect")
		reply, err := framework.SendGameServerUDP(t, gs, msg)
		if err != nil {
			t.Fatalf("Could not message GameServer: %v", err)
		}
		assert.Equal(t, fmt.Sprintf("ACK: %s\n", msg), reply)
	}

	// deliberately do this before polling, to test the SDK returning the correct
	// results before it is committed to the GameServer resource.
	reply, err := framework.SendGameServerUDP(t, gs, "PLAYER_CONNECTED 1")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "true\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "GET_PLAYERS")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.ElementsMatch(t, []string{"1", "2", "3"}, strings.Split(strings.TrimSpace(reply), ","))

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_COUNT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "3\n", reply)

	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return gs.Status.Players.Count == playerCount, nil
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2", "3"}, gs.Status.Players.IDs)

	// let's disconnect player 2
	logrus.Info("Disconnect Player 2")
	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_DISCONNECT 2")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "ACK: PLAYER_DISCONNECT 2\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_CONNECTED 2")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "false\n", reply)

	reply, err = framework.SendGameServerUDP(t, gs, "GET_PLAYERS")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.ElementsMatch(t, []string{"1", "3"}, strings.Split(strings.TrimSpace(reply), ","))

	reply, err = framework.SendGameServerUDP(t, gs, "PLAYER_COUNT")
	if err != nil {
		t.Fatalf("Could not message GameServer: %v", err)
	}
	assert.Equal(t, "2\n", reply)

	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return gs.Status.Players.Count == 2, nil
	})
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "3"}, gs.Status.Players.IDs)
}

func TestCounters(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)

	gs.Spec.Counters = make(map[string]agonesv1.CounterStatus)
	gs.Spec.Counters["games"] = agonesv1.CounterStatus{
		Count:    1,
		Capacity: 50,
	}
	gs.Spec.Counters["foo"] = agonesv1.CounterStatus{
		Count:    10,
		Capacity: 100,
	}
	gs.Spec.Counters["bar"] = agonesv1.CounterStatus{
		Count:    10,
		Capacity: 10,
	}
	gs.Spec.Counters["baz"] = agonesv1.CounterStatus{
		Count:    1000,
		Capacity: 1000,
	}
	gs.Spec.Counters["qux"] = agonesv1.CounterStatus{
		Count:    42,
		Capacity: 50,
	}

	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

	testCases := map[string]struct {
		msg          string
		want         string
		counterName  string
		wantCount    string
		wantCapacity string
	}{
		"GetCounterCount": {
			msg:  "GET_COUNTER_COUNT games",
			want: "COUNTER: 1\n",
		},
		"GetCounterCount Counter Does Not Exist": {
			msg:  "GET_COUNTER_COUNT fame",
			want: "ERROR: -1\n",
		},
		"IncrementCounter": {
			msg:         "INCREMENT_COUNTER foo 10",
			want:        "SUCCESS\n",
			counterName: "foo",
			wantCount:   "COUNTER: 20\n",
		},
		"IncrementCounter Past Capacity": {
			msg:         "INCREMENT_COUNTER games 50",
			want:        "ERROR: could not increment Counter games by amount 50: rpc error: code = Unknown desc = out of range. Count must be within range [0,Capacity]. Found Count: 51, Capacity: 50\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"IncrementCounter Negative": {
			msg:         "INCREMENT_COUNTER games -1",
			want:        "ERROR: amount must be a positive int64, found -1\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"IncrementCounter Counter Does Not Exist": {
			msg:  "INCREMENT_COUNTER same 1",
			want: "ERROR: could not increment Counter same by amount 1: rpc error: code = Unknown desc = counter not found: same\n",
		},
		"DecrementCounter": {
			msg:         "DECREMENT_COUNTER bar 10",
			want:        "SUCCESS\n",
			counterName: "bar",
			wantCount:   "COUNTER: 0\n",
		},
		"DecrementCounter Past Capacity": {
			msg:         "DECREMENT_COUNTER games 2",
			want:        "ERROR: could not decrement Counter games by amount 2: rpc error: code = Unknown desc = out of range. Count must be within range [0,Capacity]. Found Count: -1, Capacity: 50\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"DecrementCounter Negative": {
			msg:         "DECREMENT_COUNTER games -1",
			want:        "ERROR: amount must be a positive int64, found -1\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"DecrementCounter Counter Does Not Exist": {
			msg:  "DECREMENT_COUNTER lame 1",
			want: "ERROR: could not decrement Counter lame by amount 1: rpc error: code = Unknown desc = counter not found: lame\n",
		},
		"SetCounterCount": {
			msg:         "SET_COUNTER_COUNT baz 0",
			want:        "SUCCESS\n",
			counterName: "baz",
			wantCount:   "COUNTER: 0\n",
		},
		"SetCounterCount Past Capacity": {
			msg:         "SET_COUNTER_COUNT games 51",
			want:        "ERROR: could not set Counter games count to amount 51: rpc error: code = Unknown desc = out of range. Count must be within range [0,Capacity]. Found Count: 51, Capacity: 50\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"SetCounterCount Past Zero": {
			msg:         "SET_COUNTER_COUNT games -1",
			want:        "ERROR: could not set Counter games count to amount -1: rpc error: code = Unknown desc = out of range. Count must be within range [0,Capacity]. Found Count: -1, Capacity: 50\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
		"GetCounterCapacity": {
			msg:  "GET_COUNTER_CAPACITY games",
			want: "CAPACITY: 50\n",
		},
		"GetCounterCapacity Counter Does Not Exist": {
			msg:  "GET_COUNTER_CAPACITY dame",
			want: "ERROR: -1\n",
		},
		"SetCounterCapacity": {
			msg:          "SET_COUNTER_CAPACITY qux 0",
			want:         "SUCCESS\n",
			counterName:  "qux",
			wantCapacity: "CAPACITY: 0\n",
		},
		"SetCounterCapacity Past Zero": {
			msg:         "SET_COUNTER_CAPACITY games -42",
			want:        "ERROR: could not set Counter games capacity to amount -42: rpc error: code = Unknown desc = out of range. Capacity must be greater than or equal to 0. Found Capacity: -42\n",
			counterName: "games",
			wantCount:   "COUNTER: 1\n",
		},
	}
	// nolint:dupl  // Linter errors on lines are duplicate of TestLists
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			logrus.WithField("msg", testCase.msg).Info(name)
			reply, err := framework.SendGameServerUDP(t, gs, testCase.msg)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, reply)

			if testCase.wantCount != "" {
				msg := "GET_COUNTER_COUNT " + testCase.counterName
				logrus.WithField("msg", msg).Info("Sending GetCounterCount")
				reply, err = framework.SendGameServerUDP(t, gs, msg)
				require.NoError(t, err)
				assert.Equal(t, testCase.wantCount, reply)
			}

			if testCase.wantCapacity != "" {
				msg := "GET_COUNTER_CAPACITY " + testCase.counterName
				logrus.WithField("msg", msg).Info("Sending GetCounterCapacity")
				reply, err = framework.SendGameServerUDP(t, gs, msg)
				require.NoError(t, err)
				assert.Equal(t, testCase.wantCapacity, reply)
			}
		})
	}
}

func TestLists(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Labels = map[string]string{agonesv1.FleetNameLabel: "fleet-example"}
	gs.Spec.Lists = make(map[string]agonesv1.ListStatus)
	gs.Spec.Lists["players"] = agonesv1.ListStatus{
		Capacity: 1000,
	}
	gs.Spec.Lists["games"] = agonesv1.ListStatus{
		Values:   []string{"game1", "game2"},
		Capacity: 50,
	}
	gs.Spec.Lists["foo"] = agonesv1.ListStatus{
		Values:   []string{},
		Capacity: 1,
	}
	gs.Spec.Lists["bar"] = agonesv1.ListStatus{
		Values:   []string{"bar1", "bar2"},
		Capacity: 10,
	}
	gs.Spec.Lists["baz"] = agonesv1.ListStatus{
		Values:   []string{"baz1"},
		Capacity: 1,
	}
	gs.Spec.Lists["qux"] = agonesv1.ListStatus{
		Values:   []string{"qux1", "qux2", "qux3", "qux4"},
		Capacity: 5,
	}

	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

	testCases := map[string]struct {
		msg          string
		want         string
		listName     string
		wantLength   string
		wantCapacity string
	}{
		"GetListCapacity": {
			msg:  "GET_LIST_CAPACITY games",
			want: "CAPACITY: 50\n",
		},
		"SetListCapacity": {
			msg:          "SET_LIST_CAPACITY foo 1000",
			want:         "SUCCESS\n",
			listName:     "foo",
			wantCapacity: "CAPACITY: 1000\n",
		},
		"SetListCapacity past 1000": {
			msg:          "SET_LIST_CAPACITY games 1001",
			want:         "ERROR: could not set List games capacity to amount 1001: rpc error: code = Unknown desc = out of range. Capacity must be within range [0,1000]. Found Capacity: 1001\n",
			listName:     "games",
			wantCapacity: "CAPACITY: 50\n",
		},
		"SetListCapacity negative": {
			msg:          "SET_LIST_CAPACITY games -1",
			want:         "ERROR: could not set List games capacity to amount -1: rpc error: code = Unknown desc = out of range. Capacity must be within range [0,1000]. Found Capacity: -1\n",
			listName:     "games",
			wantCapacity: "CAPACITY: 50\n",
		},
		"ListContains": {
			msg:  "LIST_CONTAINS games game2",
			want: "FOUND: true\n",
		},
		"ListContains false": {
			msg:  "LIST_CONTAINS games game0",
			want: "FOUND: false\n",
		},
		"GetListLength": {
			msg:  "GET_LIST_LENGTH games",
			want: "LENGTH: 2\n",
		},
		"GetListValues": {
			msg:  "GET_LIST_VALUES games",
			want: "VALUES: game1,game2\n",
		},
		"GetListValues empty": {
			msg:  "GET_LIST_VALUES foo",
			want: "VALUES: <none>\n",
		},
		"AppendListValue": {
			msg:        "APPEND_LIST_VALUE bar bar3",
			want:       "SUCCESS\n",
			listName:   "bar",
			wantLength: "LENGTH: 3\n",
		},
		"AppendListValue past capacity": {
			msg:        "APPEND_LIST_VALUE baz baz2",
			want:       "ERROR: could not get List baz: rpc error: code = Unknown desc = out of range. No available capacity. Current Capacity: 1, List Size: 1\n",
			listName:   "baz",
			wantLength: "LENGTH: 1\n",
		},
		"DeleteListValue": {
			msg:        "DELETE_LIST_VALUE qux qux3",
			want:       "SUCCESS\n",
			listName:   "qux",
			wantLength: "LENGTH: 3\n",
		},
		"DeleteListValue value does not exist": {
			msg:        "DELETE_LIST_VALUE games game4",
			want:       "ERROR: could not get List games: rpc error: code = Unknown desc = not found: value game4 not in list games\n",
			listName:   "games",
			wantLength: "LENGTH: 2\n",
		},
	}
	// nolint:dupl  // Linter errors on lines are duplicate of TestCounters
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			logrus.WithField("msg", testCase.msg).Info(name)
			reply, err := framework.SendGameServerUDP(t, gs, testCase.msg)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, reply)

			if testCase.wantLength != "" {
				msg := "GET_LIST_LENGTH " + testCase.listName
				logrus.WithField("msg", msg).Info("Sending GetListLength")
				reply, err = framework.SendGameServerUDP(t, gs, msg)
				require.NoError(t, err)
				assert.Equal(t, testCase.wantLength, reply)
			}

			if testCase.wantCapacity != "" {
				msg := "GET_LIST_CAPACITY " + testCase.listName
				logrus.WithField("msg", msg).Info("Sending GetListCapacity")
				reply, err = framework.SendGameServerUDP(t, gs, msg)
				require.NoError(t, err)
				assert.Equal(t, testCase.wantCapacity, reply)
			}
		})
	}
}

func TestSideCarCommunicatesWhileTerminating(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)

	minute := int64(60)
	gs.Spec.Template.Spec.Containers[0].Args = append(gs.Spec.Template.Spec.Containers[0].Args, "--gracefulTerminationDelaySec", "60")
	gs.Spec.Template.Spec.TerminationGracePeriodSeconds = &minute
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	require.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)

	// delete the GameServer
	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	err = gameServers.Delete(context.Background(), readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err, "Could not delete GameServer")

	// wait for the deletion timestamp to be set
	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
		gs, err := gameServers.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return gs.DeletionTimestamp != nil, nil
	})
	require.NoError(t, err, "Could not get a GameServer with deletion timestamp")

	// send a "GAMESERVER" message, and confirm it works
	reply, err := framework.SendGameServerUDP(t, readyGs, "GAMESERVER")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("NAME: %s\n", readyGs.ObjectMeta.Name), reply)
}

func TestGracefulShutdown(t *testing.T) {
	// with the new sidecar pattern, there's no need for waiting on the Shutdown state.
	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		t.SkipNow()
	}

	t.Parallel()

	log := e2eframework.TestLogger(t)
	ctx := context.Background()
	gs := framework.DefaultGameServer(framework.Namespace)
	var minute int64 = 60
	gs.Spec.Template.Spec.TerminationGracePeriodSeconds = &minute
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	if err != nil {
		t.Fatalf("Could not get a GameServer ready: %v", err)
	}
	assert.Equal(t, readyGs.Status.State, agonesv1.GameServerStateReady)
	gameservers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	err = gameservers.Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
	require.NoError(t, err)
	log.Info("Deleted GameServer, waiting 20 seconds...")
	time.Sleep(20 * time.Second)
	log.WithField("gs", gs).Info("Checking GameServer")
	gs, err = gameservers.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, readyGs.ObjectMeta.Name, gs.ObjectMeta.Name)

	// move it to shutdown
	gsCopy := gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateShutdown
	_, err = gameservers.Update(ctx, gsCopy, metav1.UpdateOptions{})
	require.NoError(t, err)

	start := time.Now()
	require.Eventually(t, func() bool {
		_, err := gameservers.Get(ctx, readyGs.ObjectMeta.Name, metav1.GetOptions{})
		log.WithError(err).Info("checking GameServer")
		if err == nil {
			return false
		}
		return k8serrors.IsNotFound(err)
	}, 40*time.Second, time.Second)

	diff := int(time.Since(start).Seconds())
	log.WithField("diff", diff).Info("Time difference")
	require.Less(t, diff, 40)
}

func TestGameServerSlowStart(t *testing.T) {
	t.Parallel()

	// Inject an additional game server sidecar that forces a delayed start
	// to the main game server container following the pattern at
	// https://medium.com/@marko.luksa/delaying-application-start-until-sidecar-is-ready-2ec2d21a7b74
	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Template.Spec.Containers = append(
		[]corev1.Container{{
			Name:            "delay-game-server-start",
			Image:           "alpine:latest",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"sleep", "3600"},
			Lifecycle: &corev1.Lifecycle{
				PostStart: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{"sleep", "60"},
					},
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("30m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("30m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		}},
		gs.Spec.Template.Spec.Containers...)

	// Validate that a game server whose primary container starts slowly (a full minute
	// after the SDK starts) is capable of reaching Ready. Here we force the condition
	// with a lifecycle hook, but it imitates a slow image pull, or other container
	// start delays.
	_, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	assert.NoError(t, err)
}

func TestGameServerPatch(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()

	gs := framework.DefaultGameServer(framework.Namespace)
	gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, gs.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint: errcheck
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

	// Create a gameserver to patch against
	gsCopy := gs.DeepCopy()
	gsCopy.ObjectMeta.Labels = map[string]string{"foo": "foo-value"}

	patch, err := gs.Patch(gsCopy)
	require.NoError(t, err)
	patchString := string(patch)
	require.Contains(t, patchString, fmt.Sprintf("{\"op\":\"test\",\"path\":\"/metadata/resourceVersion\",\"value\":%q}", gs.ObjectMeta.ResourceVersion))
	require.Contains(t, patchString, "{\"op\":\"add\",\"path\":\"/metadata/labels\",\"value\":{\"foo\":\"foo-value\"}}")

	// Confirm patch is applied correctly
	patchedGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Patch(ctx, gs.GetObjectMeta().GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
	require.NoError(t, err)
	require.Equal(t, patchedGs.ObjectMeta.Labels, map[string]string{"foo": "foo-value"})
	require.NotEqual(t, patchedGs.ObjectMeta.ResourceVersion, gs.ObjectMeta.ResourceVersion)

	// Confirm a patch applied to an old version of a game server is not applied
	gsCopy.ObjectMeta.Labels = map[string]string{"bar": "bar-value"}
	patch, err = gs.Patch(gsCopy)
	require.NoError(t, err)

	_, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Patch(ctx, gs.GetObjectMeta().GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
	require.Error(t, err)

	getGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, getGs.ObjectMeta.Labels, map[string]string{"foo": "foo-value"})
	require.Equal(t, getGs.ObjectMeta.ResourceVersion, patchedGs.ObjectMeta.ResourceVersion)

	// Confirm patch goes through with the most up-to-date game server
	gsCopy = patchedGs.DeepCopy()
	gsCopy.ObjectMeta.Labels = map[string]string{"bar": "bar-value"}
	patch, err = patchedGs.Patch(gsCopy)
	require.NoError(t, err)

	rePatchedGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Patch(ctx, gs.GetObjectMeta().GetName(), types.JSONPatchType, patch, metav1.PatchOptions{})
	require.NoError(t, err)
	require.Equal(t, rePatchedGs.ObjectMeta.Labels, map[string]string{"bar": "bar-value"})

	getGs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, getGs.ObjectMeta.Labels, map[string]string{"bar": "bar-value"})
	require.Equal(t, getGs.ObjectMeta.ResourceVersion, rePatchedGs.ObjectMeta.ResourceVersion)
}
