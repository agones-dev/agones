// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package gke

import (
	"fmt"
	"testing"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestSyncPodPortsToGameServer(t *testing.T) {
	assignmentAnnotation := map[string]string{hostPortAssignmentAnnotation: `{"min":7000,"max":8000,"portsAssigned":{"7001":7737,"7002":7738}}`}
	badAnnotation := map[string]string{hostPortAssignmentAnnotation: `good luck parsing this as JSON`}
	for name, tc := range map[string]struct {
		gs      *agonesv1.GameServer
		pod     *corev1.Pod
		wantGS  *agonesv1.GameServer
		wantErr bool
	}{
		"no ports => no change": {
			gs:     &agonesv1.GameServer{},
			pod:    testPod(nil),
			wantGS: &agonesv1.GameServer{},
		},
		"no annotation => no change": {
			gs:     testGameServer([]int32{7777}, nil),
			pod:    testPod(nil),
			wantGS: testGameServer([]int32{7777}, nil),
		},
		"annotation => ports mapped": {
			gs:     testGameServer([]int32{7002, 7001, 7002}, nil),
			pod:    testPod(assignmentAnnotation),
			wantGS: testGameServer([]int32{7738, 7737, 7738}, nil),
		},
		"annotation, but ports already assigned => ports mapped": {
			gs:     testGameServer([]int32{7001, 7002}, []int32{7001, 7002}),
			pod:    testPod(assignmentAnnotation),
			wantGS: testGameServer([]int32{7001, 7002}, []int32{7001, 7002}),
		},
		"bad annotation": {
			gs:      testGameServer([]int32{7002, 7001, 7002}, nil),
			pod:     testPod(badAnnotation),
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			oldPod := tc.pod.DeepCopy()
			err := (&gkeAutopilot{}).SyncPodPortsToGameServer(tc.gs, tc.pod)
			if tc.wantErr {
				assert.NotNil(t, err)
				return
			}
			if assert.NoError(t, err) {
				require.Equal(t, tc.wantGS, tc.gs)
				require.Equal(t, oldPod, tc.pod)
			}
		})
	}
}

func TestValidateGameServer(t *testing.T) {
	for name, tc := range map[string]struct {
		edPods             bool
		ports              []agonesv1.GameServerPort
		scheduling         apis.SchedulingStrategy
		safeToEvict        agonesv1.EvictionSafe
		want               field.ErrorList
		portPolicyNoneFlag string
	}{
		"no ports => validated": {scheduling: apis.Packed},
		"good ports => validated": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "none-udp",
					PortPolicy:    agonesv1.None,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-udp",
					PortPolicy:    agonesv1.Passthrough,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-tcp",
					PortPolicy:    agonesv1.Passthrough,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			safeToEvict: agonesv1.EvictionSafeAlways,
			scheduling:  apis.Packed,
		},
		"bad port range => fails validation": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "best-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "bad-range",
					PortPolicy:    agonesv1.Dynamic,
					Range:         "game",
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "another-bad-range",
					PortPolicy:    agonesv1.Dynamic,
					Range:         "game",
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-udp-bad-range",
					PortPolicy:    agonesv1.Passthrough,
					Range:         "passthrough",
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-tcp-bad-range",
					PortPolicy:    agonesv1.Passthrough,
					Range:         "games",
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			safeToEvict: agonesv1.EvictionSafeAlways,
			scheduling:  apis.Packed,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "ports").Index(1).Child("range"), "game", "range must not be used on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(2).Child("range"), "game", "range must not be used on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(3).Child("range"), "passthrough", "range must not be used on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(4).Child("range"), "games", "range must not be used on GKE Autopilot"),
			},
		},
		"bad policy (no feature gates) => fails validation": {
			ports: []agonesv1.GameServerPort{
				{
					Name:          "best-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "bad-udp",
					PortPolicy:    agonesv1.Static,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "another-bad-udp",
					PortPolicy:    agonesv1.Static,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			safeToEvict: agonesv1.EvictionSafeOnUpgrade,
			scheduling:  apis.Distributed,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "scheduling"), "Distributed", "scheduling strategy must be Packed on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(1).Child("portPolicy"), agonesv1.Static, "portPolicy must be Dynamic, Passthrough, or None on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(2).Child("portPolicy"), agonesv1.Static, "portPolicy must be Dynamic, Passthrough, or None on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "eviction", "safe"), "OnUpgrade", "eviction.safe OnUpgrade not supported on GKE Autopilot"),
			},
		},
		"bad policy (GKEAutopilotExtendedDurationPods enabled) => fails validation but OnUpgrade works": {
			edPods: true,
			ports: []agonesv1.GameServerPort{
				{
					Name:          "best-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "bad-udp",
					PortPolicy:    agonesv1.Static,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "another-bad-udp",
					PortPolicy:    agonesv1.Static,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-udp",
					PortPolicy:    agonesv1.Passthrough,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			safeToEvict: agonesv1.EvictionSafeOnUpgrade,
			scheduling:  apis.Distributed,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "scheduling"), "Distributed", "scheduling strategy must be Packed on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(1).Child("portPolicy"), agonesv1.Static, "portPolicy must be Dynamic, Passthrough, or None on GKE Autopilot"),
				field.Invalid(field.NewPath("spec", "ports").Index(2).Child("portPolicy"), agonesv1.Static, "portPolicy must be Dynamic, Passthrough, or None on GKE Autopilot"),
			},
		},
		"port policy none with feature disabled => fails validation": {
			portPolicyNoneFlag: "false",
			ports: []agonesv1.GameServerPort{
				{
					Name:          "none-gate-turned-off",
					PortPolicy:    agonesv1.None,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			scheduling: apis.Packed,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "ports").Index(0).Child("portPolicy"), agonesv1.None, "PortPolicy 'None' is not enabled"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			// PortPolicy None is behind a feature flag
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			if tc.portPolicyNoneFlag == "" {
				tc.portPolicyNoneFlag = "true"
			}
			require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=%s", runtime.FeaturePortPolicyNone, tc.portPolicyNoneFlag)))
			causes := (&gkeAutopilot{useExtendedDurationPods: tc.edPods}).ValidateGameServerSpec(&agonesv1.GameServerSpec{
				Ports:      tc.ports,
				Scheduling: tc.scheduling,
				Eviction:   &agonesv1.Eviction{Safe: tc.safeToEvict},
			}, field.NewPath("spec"))
			require.Equal(t, tc.want, causes)
		})
	}
}

func TestPodSeccompUnconfined(t *testing.T) {
	for name, tc := range map[string]struct {
		podSpec     *corev1.PodSpec
		wantPodSpec *corev1.PodSpec
	}{
		"no context defined": {
			podSpec: &corev1.PodSpec{},
			wantPodSpec: &corev1.PodSpec{
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
				},
			},
		},
		"security context set, no seccomp set": {
			podSpec: &corev1.PodSpec{SecurityContext: &corev1.PodSecurityContext{}},
			wantPodSpec: &corev1.PodSpec{
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
				},
			},
		},
		"seccomp already set": {
			podSpec: &corev1.PodSpec{
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				},
			},
			wantPodSpec: &corev1.PodSpec{
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			podSpec := tc.podSpec.DeepCopy()
			podSpecSeccompUnconfined(podSpec)
			assert.Equal(t, tc.wantPodSpec, podSpec)
		})
	}
}

func TestSetPassthroughLabel(t *testing.T) {
	for name, tc := range map[string]struct {
		pod      *corev1.Pod
		wantPod  *corev1.Pod
		ports    []agonesv1.GameServerPort
		features string
	}{
		"gameserver with with Passthrough port policy adds label to pod": {
			features: "",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			ports: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
					Labels: map[string]string{
						agonesv1.GameServerPortPolicyPodLabel: "autopilot-passthrough",
					},
				},
			},
		},
		"gameserver with  Static port policy does not add label to pod": {
			features: "",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			ports: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(tc.features))
			gs := (&autopilotPortAllocator{minPort: 7000, maxPort: 8000}).Allocate(&agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.ports}})
			pod := tc.pod.DeepCopy()
			setPassthroughLabel(&gs.Spec, pod)
			assert.Equal(t, tc.wantPod, pod)
		})
	}
}

func TestSetEvictionNoExtended(t *testing.T) {
	emptyPodAnd := func(f func(*corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
		}
		f(pod)
		return pod
	}
	for desc, tc := range map[string]struct {
		eviction *agonesv1.Eviction
		pod      *corev1.Pod
		wantPod  *corev1.Pod
		wantErr  bool
	}{
		"eviction: safe: Always, no incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
			}),
		},
		"eviction: safe: Never, no incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
			}),
		},
		"eviction: safe: OnUpgrade => error": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeOnUpgrade},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantErr:  true,
		},
		"eviction: safe: Always, incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
		},
		"eviction: safe: Never, incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
		},
		"eviction: safe: Never, but safe-to-evict pod annotation set to false": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
			}),
		},
	} {
		t.Run(desc, func(t *testing.T) {
			err := setEvictionNoExtended(tc.eviction, tc.pod)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.wantPod, tc.pod)
		})
	}
}

func TestAutopilotPortAllocator(t *testing.T) {
	for name, tc := range map[string]struct {
		ports           []agonesv1.GameServerPort
		wantPorts       []agonesv1.GameServerPort
		passthroughFlag string
		wantAnnotation  bool
	}{
		"no ports => no change": {passthroughFlag: "false"},
		"ports => assigned and annotated": {
			passthroughFlag: "true",
			ports: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
				{
					Name:          "passthrough-tcp",
					PortPolicy:    agonesv1.Passthrough,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "passthrough-tcpudp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 5678,
					Protocol:      agonesv1.ProtocolTCPUDP,
				},
			},
			wantPorts: []agonesv1.GameServerPort{
				{
					Name:          "some-tcpudp-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					HostPort:      1,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "some-tcpudp-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 4321,
					HostPort:      1,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					HostPort:      2,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 1234,
					HostPort:      3,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp-tcp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					HostPort:      4,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "another-tcpudp-udp",
					PortPolicy:    agonesv1.Dynamic,
					ContainerPort: 5678,
					HostPort:      4,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-tcp",
					PortPolicy:    agonesv1.Passthrough,
					Range:         agonesv1.DefaultPortRange,
					ContainerPort: 1234,
					HostPort:      5,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "passthrough-tcpudp-tcp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 5678,
					HostPort:      6,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "passthrough-tcpudp-udp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 5678,
					HostPort:      6,
					Protocol:      corev1.ProtocolUDP,
				},
			},
			wantAnnotation: true,
		},
		"bad policy => no change (should be rejected by webhooks previously)": {
			passthroughFlag: "false",
			ports: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-none-udp",
					PortPolicy:    agonesv1.None,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-tcp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			wantPorts: []agonesv1.GameServerPort{
				{
					Name:          "awesome-udp",
					PortPolicy:    agonesv1.Static,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "awesome-none-udp",
					PortPolicy:    agonesv1.None,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolUDP,
				},
				{
					Name:          "passthrough-tcp",
					PortPolicy:    agonesv1.Passthrough,
					ContainerPort: 1234,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			// PortPolicy None is behind a feature flag
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			gs := (&autopilotPortAllocator{minPort: 8000, maxPort: 9000}).Allocate(&agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.ports}})
			wantGS := &agonesv1.GameServer{Spec: agonesv1.GameServerSpec{Ports: tc.wantPorts}}
			if tc.wantAnnotation {
				wantGS.Spec.Template = corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"autopilot.gke.io/host-port-assignment": `{"min":8000,"max":9000}`},
					},
				}
			}
			require.Equal(t, wantGS, gs)
		})
	}
}

func testPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "best-game-server",
			Namespace:   "best-game",
			Annotations: annotations,
		},
		TypeMeta: metav1.TypeMeta{Kind: "Pod"},
	}
}

func testGameServer(portSpecIn []int32, portStatusIn []int32) *agonesv1.GameServer {
	var portSpec []agonesv1.GameServerPort
	for _, port := range portSpecIn {
		portSpec = append(portSpec, agonesv1.GameServerPort{HostPort: port})
	}
	var portStatus []agonesv1.GameServerStatusPort
	for _, port := range portStatusIn {
		portStatus = append(portStatus, agonesv1.GameServerStatusPort{Port: port})
	}
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "best-game-server",
			Namespace: "best-game",
		},
		Spec: agonesv1.GameServerSpec{
			Ports: portSpec,
		},
		Status: agonesv1.GameServerStatus{
			Ports: portStatus,
		},
	}
}
