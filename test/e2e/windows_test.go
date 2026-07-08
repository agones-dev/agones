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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

// TestWindowsCreateConnect is a smoke test that verifies a GameServer can be
// scheduled onto a Windows node, reach Ready, and successfully respond to a
// UDP connection. Windows GameServer support is Beta - see
// https://agones.dev/site/docs/guides/windows-gameservers/ for details.
func TestWindowsCreateConnect(t *testing.T) {
	t.Parallel()

	if framework.CloudProduct == "gke-autopilot" {
		t.Skip("Windows nodes are not supported on GKE Autopilot")
	}

	gs := framework.DefaultGameServer(framework.Namespace)
	gs.Spec.Template.Spec.NodeSelector = map[string]string{
		"kubernetes.io/os": "windows",
	}
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
