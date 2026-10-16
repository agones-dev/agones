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

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// windowsReadyTimeout is longer than the default WaitForState, as the first pull of a
// multi-GB Windows Server Core image onto a node can take well over 5 minutes.
const windowsReadyTimeout = 15 * time.Minute

func TestWindowsCreateConnect(t *testing.T) {
	t.Parallel()

	framework.SkipOnCloudProduct(t, "gke-autopilot", "Windows nodes are not supported on GKE Autopilot")
	requireWindowsNodes(t)

	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Template.Spec.NodeSelector = map[string]string{
		"kubernetes.io/os": "windows",
	}

	gs.Spec.Template.Spec.Tolerations = []corev1.Toleration{{
		Key:      "node.kubernetes.io/os",
		Operator: corev1.TolerationOpEqual,
		Value:    "windows",
		Effect:   corev1.TaintEffectNoSchedule,
	}}
	// framework.GameServerImage is multi-arch (linux/amd64 + windows/amd64),
	// so no separate Windows-tagged image is required - the correct
	// platform variant is selected automatically based on the node.
	gs.Spec.Template.Spec.Containers[0].Image = framework.GameServerImage
	// Windows counts reserved memory against the container limit, so the default 32Mi is not
	// enough for the Go runtime to start, and small CPU limits make startup very slow.
	gs.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("30m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
	}

	newGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Create(context.Background(), gs, metav1.CreateOptions{})
	require.NoError(t, err, "Could not create a GameServer")

	readyGs, err := framework.WaitForGameServerState(t, newGs, agonesv1.GameServerStateReady, windowsReadyTimeout)
	require.NoError(t, err, "Could not get a GameServer ready")

	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello Windows World !")
	require.NoError(t, err, "Could not message GameServer")

	assert.Equal(t, "ACK: Hello Windows World !\n", reply)
}

// requireWindowsNodes fails the test when the cluster has no node labelled kubernetes.io/os=windows.
// Windows nodes come from the windows_ltsc2022 node pool in build/terraform/e2e/gke-standard/module.tf.
func requireWindowsNodes(t *testing.T) {
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubernetes.io/os=windows",
	})
	require.NoError(t, err, "Could not list nodes")
	require.NotEmpty(t, nodes.Items, "No nodes labelled kubernetes.io/os=windows found in the cluster; "+
		"apply build/terraform/e2e to add the Windows node pool")
}
