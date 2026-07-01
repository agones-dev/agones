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

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// windowsGameServerImage is the published windows/amd64 simple-game-server.
// Uses ltsc2019 — the currently available tag in agones-images registry.
const windowsGameServerImage = "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.42-windows_amd64-ltsc2019"

// windowsGameServerFixture returns a GameServer spec targeting Windows nodes.
//
// Both nodeSelector AND toleration are required on GKE:
//   - nodeSelector: restricts scheduling to Windows nodes only.
//   - toleration: GKE auto-applies node.kubernetes.io/os=windows:NoSchedule
//     to all Windows nodes. Without this the pod stays Pending forever.
func windowsGameServerFixture() *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-game-server-windows-",
			Namespace:    framework.Namespace,
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{
				{
					Name:          "default",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 7654,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "simple-game-server",
							Image: windowsGameServerImage,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("64Mi"),
									corev1.ResourceCPU:    resource.MustParse("20m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("64Mi"),
									corev1.ResourceCPU:    resource.MustParse("20m"),
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"kubernetes.io/os": "windows",
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "node.kubernetes.io/os",
							Operator: corev1.TolerationOpEqual,
							Value:    "windows",
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
	}
}

// TestWindowsCreateConnect is the Beta smoke test for Windows GameServer support.
// Validates:
//  1. GameServer with Windows image + nodeSelector + toleration reaches Ready.
//  2. Pod scheduled on a Windows node (not Linux).
//  3. sdk-server sidecar (windows/amd64) initialises and signals Ready.
//  4. UDP connectivity works from outside the cluster.
func TestWindowsCreateConnect(t *testing.T) {
	t.Parallel()
	framework.SkipOnCloudProduct(t, "gke-autopilot", "does not support Windows nodes")
	gs := windowsGameServerFixture()

	// framework is the package-level *framework.Framework from main_test.go.
	// CreateGameServerAndWaitUntilReady polls until Ready or default timeout.
	// Windows image pulls on a cold node can take 3-8 min.
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err,
		"GameServer must reach Ready — verify Windows node pool exists and image is pushed")

	defer func() {
		if err := framework.AgonesClient.AgonesV1().
			GameServers(readyGs.Namespace).
			Delete(context.Background(), readyGs.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("Failed to delete GameServer %s: %v", readyGs.Name, err)
		}
	}()
	t.Logf("GameServer %s is Ready: address=%s port=%d",
		readyGs.Name, readyGs.Status.Address, readyGs.Status.Ports[0].Port)

	// Assert the pod landed on a Windows node.
	pod, err := framework.KubeClient.CoreV1().
		Pods(readyGs.Namespace).
		Get(context.Background(), readyGs.Name, metav1.GetOptions{})
	require.NoError(t, err)

	node, err := framework.KubeClient.CoreV1().
		Nodes().
		Get(context.Background(), pod.Spec.NodeName, metav1.GetOptions{})
	require.NoError(t, err)

	nodeOS := node.Labels["kubernetes.io/os"]
	assert.Equal(t, "windows", nodeOS,
		"Pod must run on a Windows node, got os=%s on node %s", nodeOS, pod.Spec.NodeName)
	t.Logf("Pod %s running on node %s (os=%s)", pod.Name, pod.Spec.NodeName, nodeOS)

	// Assert address and port are populated.
	require.NotEmpty(t, readyGs.Status.Address)
	require.NotEmpty(t, readyGs.Status.Ports)

	// Send UDP and verify echo. simple-game-server replies "ACK: <message>".
	reply, err := framework.SendGameServerUDP(t, readyGs, "PING")
	require.NoError(t, err, "UDP send/receive must succeed at %s:%d",
		readyGs.Status.Address, readyGs.Status.Ports[0].Port)

	assert.Contains(t, reply, "ACK",
		"simple-game-server must echo ACK, got: %q", reply)
	t.Logf("UDP response: %q", reply)
}
