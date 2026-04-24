// Copyright 2023 Google LLC All Rights Reserved.
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

	e2eframework "agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

func TestSuperTuxKartGameServerReady(t *testing.T) {
	t.Parallel()

	log := e2eframework.TestLogger(t)

	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "supertuxkart-",
		},
		Spec: agonesv1.GameServerSpec{
			Container: "supertuxkart",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 8080,
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Health: agonesv1.Health{
				InitialDelaySeconds: 120,
				PeriodSeconds:       10,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "supertuxkart",
							Image: "us-docker.pkg.dev/agones-images/examples/supertuxkart-example:0.23",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("1Gi"),
									corev1.ResourceCPU:    resource.MustParse("500m"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ENABLE_PLAYER_TRACKING",
									Value: "false",
								},
							},
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	defer func() {
		if readyGs == nil {
			return
		}
		err := framework.AgonesClient.AgonesV1().GameServers(readyGs.ObjectMeta.Namespace).Delete(context.Background(), readyGs.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Error(err, "Error deleting game server")
		}
	}()
	if err != nil && readyGs != nil {
		log.Info("Game Server Events:")
		framework.LogEvents(t, log, readyGs.ObjectMeta.Namespace, readyGs)

		// Get pod and log events
		log.Info("Game Server Pod Events:")
		pod, err := framework.KubeClient.CoreV1().Pods(framework.Namespace).Get(context.Background(), readyGs.ObjectMeta.Name, metav1.GetOptions{})
		require.NoError(t, err)
		framework.LogEvents(t, log, pod.ObjectMeta.Namespace, pod)

		// Get container logs
		log.Info("Game Server Container Logs:")
		framework.LogPodContainers(t, pod)
	}
	require.NoError(t, err, "Ready game server timed out")

	// Assert that the GameServer is in the expected state
	require.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)
}

func TestRustGameServerReady(t *testing.T) {
	t.Parallel()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "rust-simple-",
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				ContainerPort: 7654,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "rust-simple",
							Image: "us-docker.pkg.dev/agones-images/examples/rust-simple-server:0.13",
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	// Assert that the GameServer is in the expected state
	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)
}

func TestCppSimpleGameServerReady(t *testing.T) {
	t.Parallel()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cpp-simple-",
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				ContainerPort: 7654,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cpp-simple",
							Image: "us-docker.pkg.dev/agones-images/examples/cpp-simple-server:0.26",
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	// Assert that the GameServer is in the expected state
	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)
}

func TestNodeJSGameServerReady(t *testing.T) {
	t.Parallel()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "nodejs-simple-",
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				ContainerPort: 7654,
				Protocol:      corev1.ProtocolUDP,
			}},
			Health: agonesv1.Health{
				InitialDelaySeconds: 30,
				PeriodSeconds:       30,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nodejs-simple",
							Image: "us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("100Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	// Assert that the GameServer is in the expected state
	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)
}

func TestXonoticGameServerReady(t *testing.T) {
	t.Parallel()
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "xonotic-",
		},

		Spec: agonesv1.GameServerSpec{
			Container: "xonotic",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 26000,
				Name:          "default",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Health: agonesv1.Health{
				InitialDelaySeconds: 60,
				PeriodSeconds:       5,
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "xonotic",
							Image: "us-docker.pkg.dev/agones-images/examples/xonotic-example:2.8",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("700Mi"),
									corev1.ResourceCPU:    resource.MustParse("200m"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Use the e2e framework's function to create the GameServer and wait until it's ready
	readyGs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
	require.NoError(t, err)

	// Assert that the GameServer is in the expected state
	assert.Equal(t, agonesv1.GameServerStateReady, readyGs.Status.State)
}
