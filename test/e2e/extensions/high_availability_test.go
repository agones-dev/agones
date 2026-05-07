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

package extensions

import (
	"context"
	"testing"
	"time"

	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Test creating a gameserver when one of the extensions pods is down/deleted
func TestGameServerCreationAfterDeletingOneExtensionsPod(t *testing.T) {
	logger := e2eframework.TestLogger(t)
	ctx := context.Background()

	assert.NoError(t, waitForAgonesExtensionsRunning(ctx))

	list, err := getAgonesExtensionsPods(ctx)
	logger.Infof("Length of pod list is %v", len(list.Items))
	if len(list.Items) > 2 {
		logger.WithField("podCount", len(list.Items)).Info("Logging events for the Deployment due to pod count > 2 before deleting extensions pod")
		for i := range list.Items {
			logger.Infof("Name of extensions pod %v: %v", i, list.Items[i].ObjectMeta.Name)
			logger.Infof("Status of extensions pod %v", list.Items[i].Status)
			framework.LogEvents(t, logger, "agones-system", &list.Items[i])
		}
	} else {
		for i := range list.Items {
			logger.Infof("Name of extensions pod %v: %v", i, list.Items[i].ObjectMeta.Name)
			logger.Infof("Host IP %v", list.Items[i].Status.HostIP)
			logger.Infof("Pod IPs %v", list.Items[i].Status.PodIPs)
		}
	}
	require.NoError(t, err, "Could not get list of Extension pods")
	require.Greater(t, len(list.Items), 1, "Cluster has no Extensions pod or has only 1 extensions pod")

	logger.Infof("Removing one of the Extensions Pods: %v", list.Items[1].ObjectMeta.Name)
	deleteAgonesExtensionsPod(ctx, t, true)

	require.NoError(t, waitForAgonesExtensionsRunning(ctx))

	gs := framework.DefaultGameServer(defaultNs)
	logger.Infof("Creating game-server %s...", gs.Name)
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, defaultNs, gs)
	require.NoError(t, err, "Could not get a GameServer ready")
	logger.WithField("gsKey", readyGs.ObjectMeta.Name).Info("GameServer Ready")

	require.NoError(t, framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(ctx, readyGs.ObjectMeta.Name, metav1.DeleteOptions{}))
}

func TestGameServerCreationRightAfterDeletingOneExtensionsPod(t *testing.T) {
	logger := e2eframework.TestLogger(t)
	ctx := context.Background()

	assert.NoError(t, waitForAgonesExtensionsRunning(ctx))

	list, err := getAgonesExtensionsPods(ctx)
	logger.Infof("Length of pod list is %v", len(list.Items))
	for i := range list.Items {
		logger.Infof("Name of extensions pod %v: %v", i, list.Items[i].ObjectMeta.Name)
	}
	require.NoError(t, err, "Could not get list of Extension pods")

	logger.Infof("Removing one of the Extensions Pods: %v", list.Items[1].ObjectMeta.Name)
	deleteAgonesExtensionsPod(ctx, t, false)

	endTime := time.Now().Add(30 * time.Second)
	for time.Now().Before(endTime) {
		gs := framework.DefaultGameServer(defaultNs)
		logger.Infof("Creating game-server %s...", gs.Name)
		newGs, err := framework.AgonesClient.AgonesV1().GameServers(defaultNs).Create(context.Background(), gs, metav1.CreateOptions{})
		assert.NoError(t, err)
		assert.NoError(t, framework.AgonesClient.AgonesV1().GameServers(defaultNs).Delete(ctx, newGs.ObjectMeta.Name, metav1.DeleteOptions{})) // nolint: errcheck
	}
}

// deleteAgonesExtensionsPod deletes one of the extensions pod for the Agones extensions,
// faking a extensions pod crash.
func deleteAgonesExtensionsPod(ctx context.Context, t *testing.T, waitForExtensions bool) {
	list, err := getAgonesExtensionsPods(ctx)
	require.NoError(t, err, "Could not get list of Extension pods")

	policy := metav1.DeletePropagationBackground
	podToDelete := list.Items[1]
	err = framework.KubeClient.CoreV1().Pods("agones-system").Delete(ctx, podToDelete.ObjectMeta.Name,
		metav1.DeleteOptions{PropagationPolicy: &policy})
	require.NoError(t, err, "Could not delete the Extension pods")
	if waitForExtensions {
		require.Eventually(t, func() bool {
			_, err := framework.KubeClient.CoreV1().Pods("agones-system").Get(ctx, podToDelete.ObjectMeta.Name, metav1.GetOptions{})
			return err != nil
		}, 5*time.Minute, time.Second)
	}
}

func waitForAgonesExtensionsRunning(ctx context.Context) error {
	return wait.PollUntilContextTimeout(ctx, time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		list, err := getAgonesExtensionsPods(ctx)
		if err != nil {
			return true, err
		}

		// count the number of Running instances
		count := 0
		for i := range list.Items {
			if list.Items[i].Status.Phase == corev1.PodRunning {
				count++
			}
		}

		return count == 2, nil
	})
}

// getAgonesExtensionsPods returns all the Agones extensions pods
func getAgonesExtensionsPods(ctx context.Context) (*corev1.PodList, error) {
	opts := metav1.ListOptions{LabelSelector: labels.Set{"agones.dev/role": "extensions"}.String()}
	pods, err := framework.KubeClient.CoreV1().Pods("agones-system").List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Filter Running pods
	var runningPods []corev1.Pod
	for i := range pods.Items {
		if pods.Items[i].ObjectMeta.DeletionTimestamp.IsZero() {
			runningPods = append(runningPods, pods.Items[i])
		}
	}

	return &corev1.PodList{Items: runningPods}, nil
}
