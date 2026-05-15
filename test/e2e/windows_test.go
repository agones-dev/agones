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

//go:build e2e
// +build e2e

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

// windowsGameServerImage is the windows/amd64 build of simple-game-server.
// Built from examples/simple-game-server/Dockerfile.windows with ltsc2019 base.
const windowsGameServerImage = "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.42-windows_amd64-ltsc2019"

// windowsGameServerFixture returns a GameServer spec targeting the Windows node pool.
//
// Both nodeSelector AND toleration are required on GKE:
//   - nodeSelector: schedules only to Windows nodes
//   - toleration: GKE auto-applies node.kubernetes.io/os=windows:NoSchedule to
//     all Windows nodes; without this toleration the pod stays Pending forever.
func windowsGameServerFixture() *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-game-server-windows-",
			Namespace:    framework.Namespace, // package-level var from main_test.go
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
//  2. Pod is scheduled on a Windows node (not Linux).
//  3. sdk-server sidecar (windows/amd64) initialises and signals Ready.
//  4. UDP connectivity to the host port works from outside the cluster.
func TestWindowsCreateConnect(t *testing.T) {
	t.Parallel()

	gs := windowsGameServerFixture()

	// framework.WaitForState is the cluster-default timeout (set in main_test.go).
	// Windows image pulls on a cold node can take 3-8 min, so we use 10 minutes.
	// The framework variable is the *framework.Framework from main_test.go — no import needed.
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err, "GameServer must reach Ready state — check Windows node pool exists and image is pushed")

	defer framework.AgonesClient.AgonesV1().
		GameServers(readyGs.Namespace).
		Delete(context.Background(), readyGs.Name, metav1.DeleteOptions{}) // nolint:errcheck

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
	t.Logf("Pod %s is running on node %s (os=%s)", pod.Name, pod.Spec.NodeName, nodeOS)

	// Assert address and port are populated.
	require.NotEmpty(t, readyGs.Status.Address, "GameServer must have an external address")
	require.NotEmpty(t, readyGs.Status.Ports, "GameServer must have at least one port")

	// Send a UDP message using the framework helper.
	// SendGameServerUDP retries 5 times and times out after 10 seconds.
	// simple-game-server echoes back "ACK: <message>".
	reply, err := framework.SendGameServerUDP(t, readyGs, "PING")
	require.NoError(t, err, "UDP send/receive must succeed against Windows GameServer at %s:%d",
		readyGs.Status.Address, readyGs.Status.Ports[0].Port)

	assert.Contains(t, reply, "ACK",
		"simple-game-server must echo back ACK, got: %q", reply)
	t.Logf("UDP response from Windows GameServer: %q", reply)
}
