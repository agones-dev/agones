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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestWindowsCreateConnect is a smoke test that verifies a GameServer can be
// scheduled onto a Windows node, reach Ready, and successfully respond to a
// UDP connection. Windows GameServer support is Beta - see
// https://agones.dev/site/docs/guides/windows-gameservers/ for details.
func TestWindowsCreateConnect(t *testing.T) {
	t.Parallel()

	framework.SkipOnCloudProduct(t, "gke-autopilot", "Windows nodes are not supported on GKE Autopilot")
	skipIfNoWindowsNodes(t)

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

	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err, "Could not get a GameServer ready")

	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)

	reply, err := framework.SendGameServerUDP(t, readyGs, "Hello Windows World !")
	require.NoError(t, err, "Could not message GameServer")

	assert.Equal(t, "ACK: Hello Windows World !\n", reply)
}

// skipIfNoWindowsNodes skips the test when the cluster has no node labelled
// kubernetes.io/os=windows.
//
// The win-ltsc2022 node pool (build/terraform/e2e/gke-standard/module.tf) is
// only defined in Terraform - it is not applied automatically for every e2e
// run, so a shared test cluster can lag behind this PR's infra changes until
// a maintainer runs `terraform apply`. Without this check, the GameServer
// created above simply sits Pending (no node satisfies the Windows
// nodeSelector/toleration) until CreateGameServerAndWaitUntilReady times out,
// which is what was failing e2e-feature-gates: the test was hard-failing on
// missing infra instead of reporting that clearly.
func skipIfNoWindowsNodes(t *testing.T) {
	nodes, err := framework.KubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubernetes.io/os=windows",
	})
	require.NoError(t, err, "could not list nodes")

	if len(nodes.Items) == 0 {
		t.Skip("skipping TestWindowsCreateConnect: no nodes labelled kubernetes.io/os=windows found in cluster " +
			"(the win-ltsc2022 Terraform node pool may not be applied to this test cluster yet)")
	}
}
