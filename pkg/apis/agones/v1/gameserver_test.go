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

package v1

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	ipFixture = "127.1.1.1"
)

func TestStatus(t *testing.T) {
	testCases := map[string]struct {
		hostPort      int32
		containerPort int32
		portPolicy    PortPolicy
		expected      GameServerStatusPort
	}{
		"PortPolicy Dynamic, should use hostPort": {
			hostPort:      7788,
			containerPort: 7777,
			portPolicy:    Dynamic,
			expected:      GameServerStatusPort{Name: "test-name", Port: 7788},
		},
		"PortPolicy Static - should use hostPort": {
			hostPort:      7788,
			containerPort: 7777,
			portPolicy:    Static,
			expected:      GameServerStatusPort{Name: "test-name", Port: 7788},
		},
		"PortPolicy Passthrough - should use hostPort": {
			hostPort:      7788,
			containerPort: 7777,
			portPolicy:    Passthrough,
			expected:      GameServerStatusPort{Name: "test-name", Port: 7788},
		},
		"PortPolicy None - should use containerPort and ignore hostPort": {
			hostPort:      7788,
			containerPort: 7777,
			portPolicy:    None,
			expected:      GameServerStatusPort{Name: "test-name", Port: 7777},
		},
	}
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePortPolicyNone)+"=true"))

	for _, tc := range testCases {
		name := "test-name"
		p := GameServerPort{Name: name, HostPort: tc.hostPort, ContainerPort: tc.containerPort, PortPolicy: tc.portPolicy}

		res := p.Status()
		assert.Equal(t, tc.expected, res)

	}
}

func TestIsBeingDeleted(t *testing.T) {
	deletionTimestamp := metav1.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	testCases := []struct {
		description string
		gs          *GameServer
		expected    bool
	}{
		{
			description: "ready gs, is not being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
				},
				Status: GameServerStatus{State: GameServerStateReady},
			},
			expected: false,
		},
		{
			description: "DeletionTimestamp is set, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &deletionTimestamp,
				},
				Status: GameServerStatus{State: GameServerStateReady},
			},
			expected: true,
		},
		{
			description: "gs status is GameServerStateShutdown, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
				},
				Status: GameServerStatus{State: GameServerStateShutdown},
			},
			expected: true,
		},
		{
			description: "gs status is GameServerStateShutdown and DeletionTimestamp is set, gs is being deleted",
			gs: &GameServer{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &deletionTimestamp,
				},
				Status: GameServerStatus{State: GameServerStateShutdown},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := tc.gs.IsBeingDeleted()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGameServerApplyDefaults(t *testing.T) {
	t.Parallel()

	ten := int64(10)

	defaultGameServerAnd := func(f func(gss *GameServerSpec)) GameServer {
		gs := GameServer{
			Spec: GameServerSpec{
				Ports: []GameServerPort{{ContainerPort: 999}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{Containers: []corev1.Container{
						{Name: "testing", Image: "testing/image"},
					}},
				},
			},
		}
		f(&gs.Spec)
		return gs
	}
	type expected struct {
		container           string
		protocol            corev1.Protocol
		state               GameServerState
		policy              PortPolicy
		portRange           string
		health              Health
		scheduling          apis.SchedulingStrategy
		sdkServer           SdkServer
		alphaPlayerCapacity *int64
		counterSpec         map[string]CounterStatus
		listSpec            map[string]ListStatus
		evictionSafeSpec    EvictionSafe
		evictionSafeStatus  EvictionSafe
	}
	wantDefaultAnd := func(f func(e *expected)) expected {
		e := expected{
			container:  "testing",
			protocol:   "UDP",
			portRange:  DefaultPortRange,
			state:      GameServerStatePortAllocation,
			policy:     Dynamic,
			scheduling: apis.Packed,
			health: Health{
				Disabled:            false,
				FailureThreshold:    3,
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
			},
			sdkServer: SdkServer{
				LogLevel: SdkServerLogLevelInfo,
				GRPCPort: 9357,
				HTTPPort: 9358,
			},
			evictionSafeSpec:   EvictionSafeNever,
			evictionSafeStatus: EvictionSafeNever,
		}
		f(&e)
		return e
	}
	data := map[string]struct {
		gameServer   GameServer
		container    string
		featureFlags string
		expected     expected
	}{
		"set basic defaults on a very simple gameserver": {
			gameServer: defaultGameServerAnd(func(_ *GameServerSpec) {}),
			expected:   wantDefaultAnd(func(_ *expected) {}),
		},
		"PlayerTracking=true": {
			featureFlags: string(runtime.FeaturePlayerTracking) + "=true",
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Players = &PlayersSpec{InitialCapacity: 10}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.alphaPlayerCapacity = &ten
			}),
		},
		"CountsAndLists=true, Counters": {
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Counters = make(map[string]CounterStatus)
				gss.Counters["games"] = CounterStatus{Count: 1, Capacity: 100}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.counterSpec = make(map[string]CounterStatus)
				e.counterSpec["games"] = CounterStatus{Count: 1, Capacity: 100}
			}),
		},
		"CountsAndLists=true, Lists": {
			featureFlags: string(runtime.FeatureCountsAndLists) + "=true",
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Lists = make(map[string]ListStatus)
				gss.Lists["players"] = ListStatus{Capacity: 100, Values: []string{"foo", "bar"}}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.listSpec = make(map[string]ListStatus)
				e.listSpec["players"] = ListStatus{Capacity: 100, Values: []string{"foo", "bar"}}
			}),
		},
		"defaults on passthrough": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Ports[0].PortPolicy = Passthrough
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.policy = Passthrough
			}),
		},
		"defaults are already set": {
			gameServer: GameServer{
				Spec: GameServerSpec{
					Container: "testing2",
					Ports: []GameServerPort{{
						Protocol:   "TCP",
						Range:      DefaultPortRange,
						PortPolicy: Static,
					}},
					Health: Health{
						Disabled:            false,
						PeriodSeconds:       12,
						InitialDelaySeconds: 11,
						FailureThreshold:    10,
					},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "testing", Image: "testing/image"},
								{Name: "testing2", Image: "testing/image2"},
							},
						},
					},
					SdkServer: SdkServer{
						LogLevel: SdkServerLogLevelInfo,
						GRPCPort: 9357,
						HTTPPort: 9358,
					},
				},
				Status: GameServerStatus{State: "TestState"},
			},
			expected: wantDefaultAnd(func(e *expected) {
				e.container = "testing2"
				e.protocol = "TCP"
				e.state = "TestState"
				e.health = Health{
					Disabled:            false,
					FailureThreshold:    10,
					InitialDelaySeconds: 11,
					PeriodSeconds:       12,
				}
			}),
		},
		"set basic defaults on static gameserver": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Ports[0].PortPolicy = Static
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.state = GameServerStateCreating
				e.policy = Static
			}),
		},
		"health is disabled": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Health = Health{Disabled: true}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.health = Health{Disabled: true}
			}),
		},
		"convert from legacy single port to multiple": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Ports[0] = GameServerPort{
					ContainerPort: 777,
					HostPort:      777,
					PortPolicy:    Static,
					Protocol:      corev1.ProtocolTCP,
				}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.protocol = "TCP"
				e.state = GameServerStateCreating
			}),
		},
		"set Debug logging level": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.SdkServer = SdkServer{LogLevel: SdkServerLogLevelDebug}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.sdkServer.LogLevel = SdkServerLogLevelDebug
			}),
		},
		"set gRPC and HTTP ports on SDK Server": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.SdkServer = SdkServer{
					LogLevel: SdkServerLogLevelError,
					GRPCPort: 19357,
					HTTPPort: 19358,
				}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.sdkServer = SdkServer{
					LogLevel: SdkServerLogLevelError,
					GRPCPort: 19357,
					HTTPPort: 19358,
				}
			}),
		},
		"defaults are eviction.safe: Never": {
			gameServer: defaultGameServerAnd(func(_ *GameServerSpec) {}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeNever
				e.evictionSafeStatus = EvictionSafeNever
			}),
		},
		"eviction.safe: Always": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Eviction = &Eviction{Safe: EvictionSafeAlways}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeAlways
				e.evictionSafeStatus = EvictionSafeAlways
			}),
		},
		"eviction.safe: OnUpgrade": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Eviction = &Eviction{Safe: EvictionSafeOnUpgrade}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeOnUpgrade
				e.evictionSafeStatus = EvictionSafeOnUpgrade
			}),
		},
		"eviction.safe: Never": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Eviction = &Eviction{Safe: EvictionSafeNever}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeNever
				e.evictionSafeStatus = EvictionSafeNever
			}),
		},
		"eviction.safe: Always inferred from safe-to-evict=true": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Template.ObjectMeta.Annotations = map[string]string{PodSafeToEvictAnnotation: "true"}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeNever
				e.evictionSafeStatus = EvictionSafeAlways
			}),
		},
		"Nothing inferred from safe-to-evict=false": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Template.ObjectMeta.Annotations = map[string]string{PodSafeToEvictAnnotation: "false"}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeNever
				e.evictionSafeStatus = EvictionSafeNever
			}),
		},
		"safe-to-evict=false AND eviction.safe: Always => eviction.safe: Always": {
			gameServer: defaultGameServerAnd(func(gss *GameServerSpec) {
				gss.Eviction = &Eviction{Safe: EvictionSafeAlways}
				gss.Template.ObjectMeta.Annotations = map[string]string{PodSafeToEvictAnnotation: "false"}
			}),
			expected: wantDefaultAnd(func(e *expected) {
				e.evictionSafeSpec = EvictionSafeAlways
				e.evictionSafeStatus = EvictionSafeAlways
			}),
		},
	}

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, test := range data {
		t.Run(name, func(t *testing.T) {
			err := runtime.ParseFeatures(test.featureFlags)
			assert.NoError(t, err)

			test.gameServer.ApplyDefaults()

			assert.Equal(t, pkg.Version, test.gameServer.Annotations[VersionAnnotation])

			spec := test.gameServer.Spec
			assert.Contains(t, test.gameServer.ObjectMeta.Finalizers, FinalizerName)
			assert.Equal(t, test.expected.container, spec.Container)
			assert.Equal(t, test.expected.protocol, spec.Ports[0].Protocol)
			assert.Equal(t, test.expected.portRange, spec.Ports[0].Range)
			assert.Equal(t, test.expected.state, test.gameServer.Status.State)
			assert.Equal(t, test.expected.scheduling, test.gameServer.Spec.Scheduling)
			assert.Equal(t, test.expected.health, test.gameServer.Spec.Health)
			assert.Equal(t, test.expected.sdkServer, test.gameServer.Spec.SdkServer)
			if test.expected.alphaPlayerCapacity != nil {
				assert.Equal(t, *test.expected.alphaPlayerCapacity, test.gameServer.Status.Players.Capacity)
			} else {
				assert.Nil(t, test.gameServer.Spec.Players)
				assert.Nil(t, test.gameServer.Status.Players)
			}
			if len(test.expected.evictionSafeSpec) > 0 {
				assert.Equal(t, test.expected.evictionSafeSpec, spec.Eviction.Safe)
			} else {
				assert.Nil(t, spec.Eviction)
			}
			if len(test.expected.evictionSafeStatus) > 0 {
				assert.Equal(t, test.expected.evictionSafeStatus, test.gameServer.Status.Eviction.Safe)
			} else {
				assert.Nil(t, test.gameServer.Status.Eviction)
			}
			if test.expected.counterSpec != nil {
				assert.Equal(t, test.expected.counterSpec, test.gameServer.Status.Counters)
			} else {
				assert.Nil(t, test.gameServer.Spec.Counters)
				assert.Nil(t, test.gameServer.Status.Counters)
			}
			if test.expected.listSpec != nil {
				assert.Equal(t, test.expected.listSpec, test.gameServer.Status.Lists)
			} else {
				assert.Nil(t, test.gameServer.Spec.Lists)
				assert.Nil(t, test.gameServer.Status.Lists)
			}
		})
	}
}

// nolint:dupl
func TestGameServerValidate(t *testing.T) {
	t.Parallel()

	longNameLen64 := strings.Repeat("f", validation.LabelValueMaxLength+1)

	testCases := []struct {
		description   string
		gs            GameServer
		applyDefaults bool
		want          field.ErrorList
	}{
		{
			description: "Valid game server",
			gs: GameServer{
				Spec: GameServerSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
			applyDefaults: true,
		},
		{
			description: "Invalid gs: container, containerPort, hostPort",
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "",
					Ports: []GameServerPort{{
						Name:       "main",
						HostPort:   5001,
						PortPolicy: Dynamic,
					}, {
						Name:          "sidecar",
						HostPort:      5002,
						PortPolicy:    Static,
						ContainerPort: 5002,
					}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
							{Name: "anothertest", Image: "testing/image"},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Required(field.NewPath("spec", "container"), "Container is required when using multiple containers in the pod template"),
				field.Invalid(field.NewPath("spec", "container"), "", "Could not find a container named "),
				field.Required(field.NewPath("spec", "ports").Index(0).Child("containerPort"), "ContainerPort must be defined for Dynamic and Static PortPolicies"),
				field.Forbidden(field.NewPath("spec", "ports").Index(0).Child("hostPort"), "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy"),
			},
		},
		{
			description: "DevAddressAnnotation: Invalid IP, no host port",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dev-game",
					Namespace:   "default",
					Annotations: map[string]string{DevAddressAnnotation: "invalid-ip"},
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{Name: "main", ContainerPort: 7777, PortPolicy: Static}},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(field.NewPath("metadata").Child("annotations", "agones.dev/dev-address"), "invalid-ip", "must be a valid IP address"),
				field.Required(field.NewPath("spec").Child("ports").Index(0).Child("hostPort"), "agones.dev/dev-address"),
			},
		},
		{
			description: "Long gs name",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.TooLongMaxLength(field.NewPath("metadata", "name"), longNameLen64, 63),
			},
		},
		{
			description: "Long gs GenerateName is not validated on agones side",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: false,
		},
		{
			description: "Long label key is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: longNameLen64,
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{longNameLen64: ""},
						},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				&field.Error{
					Type:     field.ErrorTypeInvalid,
					Field:    field.NewPath("spec", "template", "metadata", "labels").String(),
					BadValue: longNameLen64,
					Detail:   "name part must be no more than 63 characters",
					Origin:   "labelKey",
				},
			},
		},
		{
			description: "Long label value is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"agones.dev/longValueKey": longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "template", "metadata", "labels"), longNameLen64, "must be no more than 63 characters"),
			},
		},
		{
			description: "Long annotation key is invalid",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{longNameLen64: longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "template", "metadata", "annotations"), longNameLen64, "name part must be no more than 63 characters"),
			},
		},
		{
			description: "Invalid character in annotation key",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"agones.dev/short±Name": longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(field.NewPath("spec", "template", "metadata", "annotations"), "agones.dev/short±Name", "name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')"),
			},
		},
		{
			description: "Valid annotation key",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"agones.dev/shortName": longNameLen64},
						},
					},
				},
			},
			applyDefaults: false,
		},
		{
			description: "Check ContainerPort and HostPort with different policies",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "ok-name",
				},
				TypeMeta: metav1.TypeMeta{
					Kind: "test-kind",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{Name: "one", PortPolicy: Passthrough, ContainerPort: 1294},
						{PortPolicy: Passthrough, Name: "two", HostPort: 7890},
						{PortPolicy: Dynamic, Name: "three", HostPort: 7890, ContainerPort: 1294},
					},
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: true,
			want: field.ErrorList{
				field.Required(field.NewPath("spec", "ports").Index(0).Child("containerPort"), "ContainerPort cannot be specified with Passthrough PortPolicy"),
				field.Forbidden(field.NewPath("spec", "ports").Index(1).Child("hostPort"), "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy"),
				field.Forbidden(field.NewPath("spec", "ports").Index(2).Child("hostPort"), "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy"),
			},
		},
		{
			description: "PortPolicy must be Static with HostPort specified",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dev-game",
					Namespace:   "default",
					Annotations: map[string]string{DevAddressAnnotation: ipFixture},
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{PortPolicy: Passthrough, Name: "main", HostPort: 7890, ContainerPort: 7777},
					},
					Container: "my_image",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "my_image", Image: "foo/my_image"},
							},
						},
					},
				},
			},
			applyDefaults: true,
			want: field.ErrorList{
				field.Required(
					field.NewPath("spec", "ports").Index(0).Child("portPolicy"),
					"PortPolicy must be Static"),
			},
		},
		{
			description: "ContainerPort is less than zero",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: -4,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Required(
					field.NewPath("spec", "ports").Index(0).Child("containerPort"),
					"ContainerPort must be defined for Dynamic and Static PortPolicies",
				),
			},
		},
		{
			description: "CPU Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("50m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests"),
					"50m",
					"must be less than or equal to cpu limit of 30m",
				),
			},
		},
		{
			description: "CPU negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("-30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests").Key("cpu"),
					"-30m",
					"must be greater than or equal to 0",
				),
			},
		},
		{
			description: "CPU negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("-30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "limits").Key("cpu"),
					"-30m",
					"must be greater than or equal to 0",
				),
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests"),
					"30m",
					"must be less than or equal to cpu limit of -30m",
				),
			},
		},
		{
			description: "Memory Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("55Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests"),
					"55Mi",
					"must be less than or equal to memory limit of 32Mi",
				),
			},
		},
		{
			description: "Memory negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("-32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests").Key("memory"),
					"-32Mi",
					"must be greater than or equal to 0",
				),
			},
		},
		{
			description: "Memory negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{
								Name:  "testing",
								Image: "testing/image",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("32Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("30m"),
										corev1.ResourceMemory: resource.MustParse("-32Mi"),
									},
								},
							},
						}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "limits").Key("memory"),
					"-32Mi",
					"must be greater than or equal to 0",
				),
				field.Invalid(
					field.NewPath("spec", "template", "spec", "containers").Index(0).Child("resources", "requests"),
					"32Mi",
					"must be less than or equal to memory limit of -32Mi",
				),
			},
		},
		{
			description: "Init CPU Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "sidecar",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("50m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests"),
					"50m",
					"must be less than or equal to cpu limit of 30m",
				),
			},
		},
		{
			description: "Init CPU negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "sidecar",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("-30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests").Key("cpu"),
					"-30m",
					"must be greater than or equal to 0",
				),
			},
		},
		{
			description: "Init CPU negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "testing",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("-30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "limits").Key("cpu"),
					"-30m",
					"must be greater than or equal to 0",
				),
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests"),
					"30m",
					"must be less than or equal to cpu limit of -30m",
				),
			},
		},
		{
			description: "Init Memory Request > Limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "testing",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("55Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests"),
					"55Mi",
					"must be less than or equal to memory limit of 32Mi",
				),
			},
		},
		{
			description: "Init Memory negative request",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "testing",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("-32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests").Key("memory"),
					"-32Mi",
					"must be greater than or equal to 0",
				),
			},
		},
		{
			description: "Init Memory negative limit",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{{
						Name:          "main",
						ContainerPort: 7777,
						PortPolicy:    Dynamic,
					}},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}},
							InitContainers: []corev1.Container{
								{
									Name:  "testing",
									Image: "testing/image",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("32Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("30m"),
											corev1.ResourceMemory: resource.MustParse("-32Mi"),
										},
									},
								},
							}},
					},
				},
			},
			applyDefaults: false,
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "limits").Key("memory"),
					"-32Mi",
					"must be greater than or equal to 0",
				),
				field.Invalid(
					field.NewPath("spec", "template", "spec", "initContainers").Index(0).Child("resources", "requests"),
					"32Mi",
					"must be less than or equal to memory limit of -32Mi",
				),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.applyDefaults {
				tc.gs.ApplyDefaults()
			}

			errs := tc.gs.Validate(fakeAPIHooks{})
			assert.ElementsMatch(t, tc.want, errs, "ErrorList check")
		})
	}
}

func TestGameServerValidateFeatures(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	portContainerName := "another-container"

	testCases := []struct {
		description string
		feature     string
		gs          GameServer
		want        field.ErrorList
	}{
		{
			description: "Invalid container name, container was not found",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    Dynamic,
							Container:     &portContainerName,
						},
					},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}},
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("spec", "ports").Index(0).Child("container"),
					"another-container",
					"Container must be empty or the name of a container in the pod template",
				),
			},
		},
		{
			description: "Multiple container ports, OK scenario",
			gs: GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dev-game",
					Namespace: "default",
				},
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    Dynamic,
						},
					},
					Container: "testing",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{
							{Name: "testing", Image: "testing/image"},
						}},
					},
				},
			},
		},
		{
			description: "PlayerTracking is disabled, Players field specified",
			feature:     fmt.Sprintf("%s=false", runtime.FeaturePlayerTracking),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Players:   &PlayersSpec{InitialCapacity: 10},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
			want: field.ErrorList{
				field.Forbidden(
					field.NewPath("spec", "players"),
					"Value cannot be set unless feature flag PlayerTracking is enabled",
				),
			},
		},
		{
			description: "PlayerTracking is enabled, Players field specified",
			feature:     fmt.Sprintf("%s=true", runtime.FeaturePlayerTracking),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Players:   &PlayersSpec{InitialCapacity: 10},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
		},
		{
			description: "CountsAndLists is disabled, Counters field specified",
			feature:     fmt.Sprintf("%s=false", runtime.FeatureCountsAndLists),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Counters:  map[string]CounterStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
			want: field.ErrorList{
				field.Forbidden(
					field.NewPath("spec", "counters"),
					"Value cannot be set unless feature flag CountsAndLists is enabled",
				),
			},
		},
		{
			description: "CountsAndLists is disabled, Lists field specified",
			feature:     fmt.Sprintf("%s=false", runtime.FeatureCountsAndLists),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Lists:     map[string]ListStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
			want: field.ErrorList{
				field.Forbidden(
					field.NewPath("spec", "lists"),
					"Value cannot be set unless feature flag CountsAndLists is enabled",
				),
			},
		},
		{
			description: "CountsAndLists is enabled, Counters field specified",
			feature:     fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Counters:  map[string]CounterStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
		},
		{
			description: "CountsAndLists is enabled, Lists field specified",
			feature:     fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gs: GameServer{
				Spec: GameServerSpec{
					Container: "testing",
					Lists:     map[string]ListStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
		},
		{
			description: "PortPolicyNone is disabled, PortPolicy field set to None",
			feature:     fmt.Sprintf("%s=false", runtime.FeaturePortPolicyNone),
			gs: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    None,
						},
					},
					Container: "testing",
					Lists:     map[string]ListStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
			want: field.ErrorList{
				field.Forbidden(
					field.NewPath("spec.ports[0].portPolicy"),
					"Value cannot be set to None unless feature flag PortPolicyNone is enabled",
				),
			},
		},
		{
			description: "PortPolicyNone is enabled, PortPolicy field set to None",
			feature:     fmt.Sprintf("%s=true", runtime.FeaturePortPolicyNone),
			gs: GameServer{
				Spec: GameServerSpec{
					Ports: []GameServerPort{
						{
							Name:          "main",
							ContainerPort: 7777,
							PortPolicy:    None,
						},
					},
					Container: "testing",
					Lists:     map[string]ListStatus{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := runtime.ParseFeatures(tc.feature)
			assert.NoError(t, err)

			errs := tc.gs.Validate(fakeAPIHooks{})
			assert.ElementsMatch(t, tc.want, errs, "ErrorList check")
		})
	}
}

func TestGameServerPodNoErrors(t *testing.T) {
	t.Parallel()
	fixture := defaultGameServer()
	fixture.ApplyDefaults()

	pod, err := fixture.Pod(fakeAPIHooks{})
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Name, pod.Spec.Hostname)
	assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
	assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
	assert.Equal(t, fixture.Spec.Container, pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, fixture.Spec.Ports[0].HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[0].ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
}

func TestGameServerPodHostName(t *testing.T) {
	t.Parallel()

	fixture := defaultGameServer()
	fixture.ObjectMeta.Name = "test-1.0"
	fixture.ApplyDefaults()
	pod, err := fixture.Pod(fakeAPIHooks{})
	require.NoError(t, err)
	assert.Equal(t, "test-1-0", pod.Spec.Hostname)

	fixture = defaultGameServer()
	fixture.ApplyDefaults()
	expected := "ORANGE"
	fixture.Spec.Template.Spec.Hostname = expected
	pod, err = fixture.Pod(fakeAPIHooks{})
	require.NoError(t, err)
	assert.Equal(t, expected, pod.Spec.Hostname)
}

func TestGameServerPodContainerNotFoundErrReturned(t *testing.T) {
	t.Parallel()

	containerName1 := "Container1"
	fixture := &GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: GameServerSpec{
			Container: "can-not-find-this-name",
			Ports: []GameServerPort{
				{
					Container:     &containerName1,
					ContainerPort: 7777,
					HostPort:      9999,
					PortPolicy:    Static,
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "Container2", Image: "container/image"}},
				},
			},
		}, Status: GameServerStatus{State: GameServerStateCreating},
	}

	_, err := fixture.Pod(fakeAPIHooks{})
	if assert.NotNil(t, err, "Pod should return an error") {
		assert.Equal(t, "failed to find container named Container1 in pod spec", err.Error())
	}
}

func TestGameServerPodWithSidecarNoErrors(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureSidecarContainers)+"=false"))

	fixture := defaultGameServer()
	fixture.ApplyDefaults()

	sidecar := corev1.Container{Name: "sidecar", Image: "container/sidecar"}
	fixture.Spec.Template.Spec.ServiceAccountName = "other-agones-sdk"
	pod, err := fixture.Pod(fakeAPIHooks{}, sidecar)
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Len(t, pod.Spec.Containers, 2, "Should have two containers")
	assert.Equal(t, "other-agones-sdk", pod.Spec.ServiceAccountName)
	assert.Equal(t, "sidecar", pod.Spec.Containers[0].Name)
	assert.Equal(t, "container", pod.Spec.Containers[1].Name)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
}

func TestGameServerPodWithInitSidecarNoErrors(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureSidecarContainers)+"=true"))

	fixture := defaultGameServer()
	fixture.ApplyDefaults()

	sidecar := corev1.Container{Name: "sidecar", Image: "container/sidecar"}
	fixture.Spec.Template.Spec.ServiceAccountName = "other-agones-sdk"
	pod, err := fixture.Pod(fakeAPIHooks{}, sidecar)
	assert.Nil(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Len(t, pod.Spec.Containers, 1, "Should have one containers")
	assert.Equal(t, "other-agones-sdk", pod.Spec.ServiceAccountName)
	assert.Equal(t, "sidecar", pod.Spec.InitContainers[0].Name)
	assert.Equal(t, corev1.ContainerRestartPolicyAlways, *pod.Spec.InitContainers[0].RestartPolicy)
	assert.Equal(t, "container", pod.Spec.Containers[0].Name)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, pod.Spec.RestartPolicy, corev1.RestartPolicyNever)
}

func TestGameServerPodWithInitSidecarPrependsToExistingInitContainers(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureSidecarContainers)+"=true"))

	fixture := defaultGameServer()
	// Add existing init containers to the template
	existingInitContainer1 := corev1.Container{Name: "existing-init-1", Image: "existing/init1"}
	existingInitContainer2 := corev1.Container{Name: "existing-init-2", Image: "existing/init2"}
	fixture.Spec.Template.Spec.InitContainers = []corev1.Container{existingInitContainer1, existingInitContainer2}
	fixture.ApplyDefaults()

	sidecar1 := corev1.Container{Name: "sidecar-1", Image: "container/sidecar1"}
	sidecar2 := corev1.Container{Name: "sidecar-2", Image: "container/sidecar2"}
	pod, err := fixture.Pod(fakeAPIHooks{}, sidecar1, sidecar2)
	require.NoError(t, err)

	// Verify that sidecars are placed at the beginning of InitContainers
	assert.Len(t, pod.Spec.InitContainers, 4, "Should have four init containers")
	assert.Equal(t, "sidecar-1", pod.Spec.InitContainers[0].Name, "First sidecar should be at index 0")
	assert.Equal(t, "sidecar-2", pod.Spec.InitContainers[1].Name, "Second sidecar should be at index 1")
	assert.Equal(t, "existing-init-1", pod.Spec.InitContainers[2].Name, "First existing init container should be at index 2")
	assert.Equal(t, "existing-init-2", pod.Spec.InitContainers[3].Name, "Second existing init container should be at index 3")

	// Verify restart policies
	assert.Equal(t, corev1.ContainerRestartPolicyAlways, *pod.Spec.InitContainers[0].RestartPolicy)
	assert.Equal(t, corev1.ContainerRestartPolicyAlways, *pod.Spec.InitContainers[1].RestartPolicy)
}

func TestGameServerPodWithMultiplePortAllocations(t *testing.T) {
	fixture := defaultGameServer()
	containerName := "authContainer"
	fixture.Spec.Template.Spec.Containers = append(fixture.Spec.Template.Spec.Containers, corev1.Container{
		Name: containerName,
	})
	fixture.Spec.Ports = append(fixture.Spec.Ports, GameServerPort{
		Name:          "containerPort",
		PortPolicy:    Dynamic,
		Container:     &containerName,
		ContainerPort: 5000,
	})
	fixture.Spec.Container = fixture.Spec.Template.Spec.Containers[0].Name
	fixture.ApplyDefaults()

	pod, err := fixture.Pod(fakeAPIHooks{})
	assert.NoError(t, err, "Pod should not return an error")
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
	assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
	assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
	assert.Equal(t, fixture.Spec.Container, pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
	assert.Equal(t, fixture.Spec.Ports[0].HostPort, pod.Spec.Containers[0].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[0].ContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
	assert.Equal(t, *fixture.Spec.Ports[0].Container, pod.Spec.Containers[0].Name)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[0].Ports[0].Protocol)
	assert.True(t, metav1.IsControlledBy(pod, fixture))
	assert.Equal(t, fixture.Spec.Ports[1].HostPort, pod.Spec.Containers[1].Ports[0].HostPort)
	assert.Equal(t, fixture.Spec.Ports[1].ContainerPort, pod.Spec.Containers[1].Ports[0].ContainerPort)
	assert.Equal(t, *fixture.Spec.Ports[1].Container, pod.Spec.Containers[1].Name)
	assert.Equal(t, corev1.Protocol("UDP"), pod.Spec.Containers[1].Ports[0].Protocol)
}

func TestGameServerPodObjectMeta(t *testing.T) {
	fixture := &GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "lucy"},
		Spec:       GameServerSpec{Container: "goat"},
	}

	for desc, tc := range map[string]struct {
		scheduling apis.SchedulingStrategy
		wantSafe   string
	}{
		"packed": {
			scheduling: apis.Packed,
		},
		"distributed": {
			scheduling: apis.Distributed,
		},
	} {
		t.Run(desc, func(t *testing.T) {
			gs := fixture.DeepCopy()
			gs.Spec.Scheduling = tc.scheduling
			pod := &corev1.Pod{}

			gs.podObjectMeta(pod)

			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Name)
			assert.Equal(t, gs.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			assert.Equal(t, GameServerLabelRole, pod.ObjectMeta.Labels[RoleLabel])
			assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
			assert.Equal(t, gs.ObjectMeta.Name, pod.ObjectMeta.Labels[GameServerPodLabel])
			assert.Equal(t, "goat", pod.ObjectMeta.Annotations[GameServerContainerAnnotation])
			assert.True(t, metav1.IsControlledBy(pod, gs))
			assert.Equal(t, tc.wantSafe, pod.ObjectMeta.Annotations[PodSafeToEvictAnnotation])
		})
	}
}

func TestGameServerPodScheduling(t *testing.T) {
	fixture := &corev1.Pod{Spec: corev1.PodSpec{}}

	t.Run("packed", func(t *testing.T) {
		gs := &GameServer{Spec: GameServerSpec{Scheduling: apis.Packed}}
		pod := fixture.DeepCopy()
		gs.podScheduling(pod)

		assert.Len(t, pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, 1)
		wpat := pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0]
		assert.Equal(t, int32(100), wpat.Weight)
		assert.Contains(t, wpat.PodAffinityTerm.LabelSelector.String(), GameServerLabelRole)
		assert.Contains(t, wpat.PodAffinityTerm.LabelSelector.String(), RoleLabel)
	})

	t.Run("distributed", func(t *testing.T) {
		gs := &GameServer{Spec: GameServerSpec{Scheduling: apis.Distributed}}
		pod := fixture.DeepCopy()
		gs.podScheduling(pod)
		assert.Empty(t, pod.Spec.Affinity)
	})
}

func TestGameServerDisableServiceAccount(t *testing.T) {
	t.Parallel()

	gs := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"}, Spec: GameServerSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
		},
	}}

	gs.ApplyDefaults()
	pod, err := gs.Pod(fakeAPIHooks{})
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 1)
	assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)

	err = gs.DisableServiceAccount(pod)
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 1)
	assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
}

func TestGameServerPassthroughPortAnnotation(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	containerOne := "containerOne"
	containerTwo := "containerTwo"
	containerThree := "containerThree"
	containerFour := "containerFour"
	gs := &GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"}, Spec: GameServerSpec{
		Container: "containerOne",
		Ports: []GameServerPort{
			{Name: "defaultDynamicOne", PortPolicy: Dynamic, ContainerPort: 7659, Container: &containerOne},
			{Name: "defaultPassthroughOne", PortPolicy: Passthrough, Container: &containerOne},
			{Name: "defaultPassthroughTwo", PortPolicy: Passthrough, Container: &containerTwo},
			{Name: "defaultDynamicTwo", PortPolicy: Dynamic, ContainerPort: 7654, Container: &containerTwo},
			{Name: "defaultDynamicThree", PortPolicy: Dynamic, ContainerPort: 7660, Container: &containerThree},
			{Name: "defaultDynamicThree", PortPolicy: Dynamic, ContainerPort: 7661, Container: &containerThree},
			{Name: "defaultDynamicThree", PortPolicy: Dynamic, ContainerPort: 7662, Container: &containerThree},
			{Name: "defaulPassthroughThree", PortPolicy: Passthrough, Container: &containerThree},
			{Name: "defaultPassthroughFour", PortPolicy: Passthrough, Container: &containerFour},
		},
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "containerOne", Image: "container/image"},
					{Name: "containerTwo", Image: "container/image"},
					{Name: "containerThree", Image: "container/image"},
					{Name: "containerFour", Image: "container/image"},
				},
			},
		},
	}}

	passthroughContainerPortMap := "{\"containerFour\":[0],\"containerOne\":[1],\"containerThree\":[3],\"containerTwo\":[0]}"

	gs.ApplyDefaults()
	pod, err := gs.Pod(fakeAPIHooks{})
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 4)
	assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)
	assert.Equal(t, pod.ObjectMeta.Annotations[PassthroughPortAssignmentAnnotation], passthroughContainerPortMap)

	err = gs.DisableServiceAccount(pod)
	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 4)
	assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
}

func TestGameServerCountPorts(t *testing.T) {
	fixture := &GameServer{Spec: GameServerSpec{Ports: []GameServerPort{
		{PortPolicy: Dynamic},
		{PortPolicy: Dynamic},
		{PortPolicy: Dynamic},
		{PortPolicy: Static},
	}}}

	assert.Equal(t, 3, fixture.CountPorts(func(policy PortPolicy) bool {
		return policy == Dynamic
	}))
	assert.Equal(t, 1, fixture.CountPorts(func(policy PortPolicy) bool {
		return policy == Static
	}))
}

func TestGameServerCountPortsForRange(t *testing.T) {
	fixture := &GameServer{Spec: GameServerSpec{Ports: []GameServerPort{
		{PortPolicy: Dynamic, Range: "test"},
		{PortPolicy: Dynamic},
		{PortPolicy: Dynamic, Range: "test"},
		{PortPolicy: Static, Range: "test"},
	}}}

	assert.Equal(t, 2, fixture.CountPortsForRange("test", func(policy PortPolicy) bool {
		return policy == Dynamic
	}))
	assert.Equal(t, 1, fixture.CountPortsForRange("test", func(policy PortPolicy) bool {
		return policy == Static
	}))
}

func TestGameServerPatch(t *testing.T) {
	fixture := &GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "lucy", ResourceVersion: "1234"},
		Spec:       GameServerSpec{Container: "goat"},
	}

	delta := fixture.DeepCopy()
	delta.Spec.Container = "bear"

	patch, err := fixture.Patch(delta)
	assert.Nil(t, err)

	assert.Contains(t, string(patch), `{"op":"replace","path":"/spec/container","value":"bear"}`)
	assert.Contains(t, string(patch), `{"op":"test","path":"/metadata/resourceVersion","value":"1234"}`)
}

func TestGameServerGetDevAddress(t *testing.T) {
	devGs := &GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "dev-game",
			Namespace:   "default",
			Annotations: map[string]string{DevAddressAnnotation: ipFixture},
		},
		Spec: GameServerSpec{
			Ports: []GameServerPort{{HostPort: 7777, PortPolicy: Static}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
			},
		},
	}

	devAddress, isDev := devGs.GetDevAddress()
	assert.True(t, isDev, "dev-game should had a dev-address")
	assert.Equal(t, ipFixture, devAddress, "dev-address IP address should be 127.1.1.1")

	regularGs := devGs.DeepCopy()
	regularGs.ObjectMeta.Annotations = map[string]string{}
	devAddress, isDev = regularGs.GetDevAddress()
	assert.False(t, isDev, "dev-game should NOT have a dev-address")
	assert.Equal(t, "", devAddress, "dev-address IP address should be 127.1.1.1")
}

func TestGameServerIsDeletable(t *testing.T) {
	gs := &GameServer{Status: GameServerStatus{State: GameServerStateStarting}}
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateAllocated
	assert.False(t, gs.IsDeletable())

	gs.Status.State = GameServerStateReserved
	assert.False(t, gs.IsDeletable())

	now := metav1.Now()
	gs.ObjectMeta.DeletionTimestamp = &now
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateAllocated
	assert.True(t, gs.IsDeletable())

	gs.Status.State = GameServerStateReady
	assert.True(t, gs.IsDeletable())
}

func TestGameServerIsBeforeReady(t *testing.T) {
	fixtures := []struct {
		state    GameServerState
		expected bool
	}{
		{GameServerStatePortAllocation, true},
		{GameServerStateCreating, true},
		{GameServerStateStarting, true},
		{GameServerStateScheduled, true},
		{GameServerStateRequestReady, true},
		{GameServerStateReady, false},
		{GameServerStateShutdown, false},
		{GameServerStateError, false},
		{GameServerStateUnhealthy, false},
		{GameServerStateReserved, false},
		{GameServerStateAllocated, false},
	}

	for _, test := range fixtures {
		t.Run(string(test.state), func(t *testing.T) {
			gs := &GameServer{Status: GameServerStatus{State: test.state}}
			assert.Equal(t, test.expected, gs.IsBeforeReady(), test.state)
		})
	}
}

func TestGameServerApplyToPodContainer(t *testing.T) {
	t.Parallel()
	type expected struct {
		err string
		tty bool
	}

	testCases := []struct {
		description string
		gs          *GameServer
		expected    expected
	}{
		{
			description: "OK, no error",
			gs: &GameServer{
				Spec: GameServerSpec{
					Container: "mycontainer",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "mycontainer", Image: "foo/mycontainer"},
								{Name: "notmycontainer", Image: "foo/notmycontainer"},
							},
						},
					},
				},
			},
			expected: expected{
				err: "",
				tty: true,
			},
		},
		{
			description: "container not found, error is returned",
			gs: &GameServer{
				Spec: GameServerSpec{
					Container: "mycontainer-WRONG-NAME",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "mycontainer", Image: "foo/mycontainer"},
								{Name: "notmycontainer", Image: "foo/notmycontainer"},
							},
						},
					},
				},
			},
			expected: expected{
				err: "failed to find container named mycontainer-WRONG-NAME in pod spec",
				tty: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			pod := &corev1.Pod{Spec: *tc.gs.Spec.Template.Spec.DeepCopy()}
			result := tc.gs.ApplyToPodContainer(pod, tc.gs.Spec.Container, func(c corev1.Container) corev1.Container {
				//  easy thing to change and test for
				c.TTY = true
				return c
			})

			if tc.expected.err != "" && assert.NotNil(t, result) {
				assert.Equal(t, tc.expected.err, result.Error())
			}
			assert.Equal(t, tc.expected.tty, pod.Spec.Containers[0].TTY)
			assert.False(t, pod.Spec.Containers[1].TTY)
		})
	}
}

func defaultGameServer() *GameServer {
	return &GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", UID: "1234"},
		Spec: GameServerSpec{
			Ports: []GameServerPort{
				{
					ContainerPort: 7777,
					HostPort:      9999,
					PortPolicy:    Static,
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
			},
		}, Status: GameServerStatus{State: GameServerStateCreating},
	}
}

func TestGameServerUpdateCount(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		gs      GameServer
		name    string
		action  string
		amount  int64
		want    CounterStatus
		wantErr bool
	}{
		"counter not in game server no-op and error": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"foos": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			name:    "foo",
			action:  "Increment",
			amount:  1,
			wantErr: true,
		},
		"negative amount no-op and error": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"foos": {
						Count:    1,
						Capacity: 100,
					},
				},
			}},
			name:   "foos",
			action: "Decrement",
			amount: -1,
			want: CounterStatus{
				Count:    1,
				Capacity: 100,
			},
			wantErr: true,
		},
		"increment by 1": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"players": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			name:   "players",
			action: "Increment",
			amount: 1,
			want: CounterStatus{
				Count:    1,
				Capacity: 100,
			},
			wantErr: false,
		},
		"decrement by 10": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"bars": {
						Count:    99,
						Capacity: 100,
					},
				},
			}},
			name:   "bars",
			action: "Decrement",
			amount: 10,
			want: CounterStatus{
				Count:    89,
				Capacity: 100,
			},
			wantErr: false,
		},
		"incorrect action no-op and error": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"bazes": {
						Count:    99,
						Capacity: 100,
					},
				},
			}},
			name:   "bazes",
			action: "decrement",
			amount: 10,
			want: CounterStatus{
				Count:    99,
				Capacity: 100,
			},
			wantErr: true,
		},
		"decrement beyond zero truncated": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"baz": {
						Count:    99,
						Capacity: 100,
					},
				},
			}},
			name:   "baz",
			action: "Decrement",
			amount: 100,
			want: CounterStatus{
				Count:    0,
				Capacity: 100,
			},
			wantErr: false,
		},
		"increment beyond capacity truncated": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"splayers": {
						Count:    99,
						Capacity: 100,
					},
				},
			}},
			name:   "splayers",
			action: "Increment",
			amount: 2,
			want: CounterStatus{
				Count:    100,
				Capacity: 100,
			},
			wantErr: false,
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			err := testCase.gs.UpdateCount(testCase.name, testCase.action, testCase.amount)
			if err != nil {
				assert.True(t, testCase.wantErr)
			} else {
				assert.False(t, testCase.wantErr)
			}
			if counter, ok := testCase.gs.Status.Counters[testCase.name]; ok {
				assert.Equal(t, testCase.want, counter)
			}
		})
	}
}

func TestGameServerUpdateCounterCapacity(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		gs       GameServer
		name     string
		capacity int64
		want     CounterStatus
		wantErr  bool
	}{
		"counter not in game server no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"foos": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			name:     "foo",
			capacity: 1000,
			wantErr:  true,
		},
		"capacity less than zero no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"foos": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			name:     "foos",
			capacity: -1000,
			want: CounterStatus{
				Count:    0,
				Capacity: 100,
			},
			wantErr: true,
		},
		"update capacity": {
			gs: GameServer{Status: GameServerStatus{
				Counters: map[string]CounterStatus{
					"sessions": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			name:     "sessions",
			capacity: 9223372036854775807,
			want: CounterStatus{
				Count:    0,
				Capacity: 9223372036854775807,
			},
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			err := testCase.gs.UpdateCounterCapacity(testCase.name, testCase.capacity)
			if err != nil {
				assert.True(t, testCase.wantErr)
			} else {
				assert.False(t, testCase.wantErr)
			}
			if counter, ok := testCase.gs.Status.Counters[testCase.name]; ok {
				assert.Equal(t, testCase.want, counter)
			}
		})
	}
}

func TestGameServerUpdateListCapacity(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		gs       GameServer
		name     string
		capacity int64
		want     ListStatus
		wantErr  bool
	}{
		"list not in game server no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"things": {
						Values:   []string{},
						Capacity: 100,
					},
				},
			}},
			name:     "thing",
			capacity: 1000,
			wantErr:  true,
		},
		"update list capacity": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"things": {
						Values:   []string{},
						Capacity: 100,
					},
				},
			}},
			name:     "things",
			capacity: 1000,
			want: ListStatus{
				Values:   []string{},
				Capacity: 1000,
			},
			wantErr: false,
		},
		"list capacity above max no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"slings": {
						Values:   []string{},
						Capacity: 100,
					},
				},
			}},
			name:     "slings",
			capacity: 10000,
			want: ListStatus{
				Values:   []string{},
				Capacity: 100,
			},
			wantErr: true,
		},
		"list capacity less than zero no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"flings": {
						Values:   []string{},
						Capacity: 999,
					},
				},
			}},
			name:     "flings",
			capacity: -100,
			want: ListStatus{
				Values:   []string{},
				Capacity: 999,
			},
			wantErr: true,
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			err := testCase.gs.UpdateListCapacity(testCase.name, testCase.capacity)
			if err != nil {
				assert.True(t, testCase.wantErr)
			} else {
				assert.False(t, testCase.wantErr)
			}
			if list, ok := testCase.gs.Status.Lists[testCase.name]; ok {
				assert.Equal(t, testCase.want, list)
			}
		})
	}
}

func TestGameServerAppendListValues(t *testing.T) {
	t.Parallel()

	var nilSlice []string

	testCases := map[string]struct {
		gs      GameServer
		name    string
		values  []string
		want    ListStatus
		wantErr bool
	}{
		"list not in game server no-op with error": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"things": {
						Values:   []string{},
						Capacity: 100,
					},
				},
			}},
			name:    "thing",
			values:  []string{"thing1", "thing2", "thing3"},
			wantErr: true,
		},
		"append values": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"things": {
						Values:   []string{"thing1"},
						Capacity: 100,
					},
				},
			}},
			name:   "things",
			values: []string{"thing2", "thing3"},
			want: ListStatus{
				Values:   []string{"thing1", "thing2", "thing3"},
				Capacity: 100,
			},
			wantErr: false,
		},
		"append values with silent drop of duplicates": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"games": {
						Values:   []string{"game0"},
						Capacity: 10,
					},
				},
			}},
			name:   "games",
			values: []string{"game1", "game2", "game2", "game1"},
			want: ListStatus{
				Values:   []string{"game0", "game1", "game2"},
				Capacity: 10,
			},
			wantErr: false,
		},
		"append values with silent drop of duplicates in original list": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"objects": {
						Values:   []string{"object1", "object2"},
						Capacity: 10,
					},
				},
			}},
			name:   "objects",
			values: []string{"object2", "object1", "object3", "object3"},
			want: ListStatus{
				Values:   []string{"object1", "object2", "object3"},
				Capacity: 10,
			},
			wantErr: false,
		},
		"append nil values": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"blings": {
						Values:   []string{"bling1"},
						Capacity: 10,
					},
				},
			}},
			name:   "blings",
			values: nilSlice,
			want: ListStatus{
				Values:   []string{"bling1"},
				Capacity: 10,
			},
			wantErr: true,
		},
		"append too many values truncates list": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"bananaslugs": {
						Values:   []string{"bananaslugs1", "bananaslug2", "bananaslug3"},
						Capacity: 5,
					},
				},
			}},
			name:   "bananaslugs",
			values: []string{"bananaslug4", "bananaslug5", "bananaslug6"},
			want: ListStatus{
				Values:   []string{"bananaslugs1", "bananaslug2", "bananaslug3", "bananaslug4", "bananaslug5"},
				Capacity: 5,
			},
			wantErr: false,
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			err := testCase.gs.AppendListValues(testCase.name, testCase.values)
			if err != nil {
				assert.True(t, testCase.wantErr)
			} else {
				assert.False(t, testCase.wantErr)
			}
			if list, ok := testCase.gs.Status.Lists[testCase.name]; ok {
				assert.Equal(t, testCase.want, list)
			}
		})
	}
}

func TestGameServerDeleteListValues(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		gs      GameServer
		name    string
		want    ListStatus
		values  []string
		wantErr bool
	}{
		"list not in game server no-op and error": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"foos": {
						Values:   []string{"foo", "bar", "bax"},
						Capacity: 100,
					},
				},
			}},
			name:    "foo",
			values:  []string{"bar", "baz"},
			wantErr: true,
		},
		"delete list value - one value not present": {
			gs: GameServer{Status: GameServerStatus{
				Lists: map[string]ListStatus{
					"foo": {
						Values:   []string{"foo", "bar", "bax"},
						Capacity: 100,
					},
				},
			}},
			name:    "foo",
			values:  []string{"bar", "baz"},
			wantErr: false,
			want: ListStatus{
				Values:   []string{"foo", "bax"},
				Capacity: 100,
			},
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			err := testCase.gs.DeleteListValues(testCase.name, testCase.values)
			if err != nil {
				assert.True(t, testCase.wantErr)
			} else {
				assert.False(t, testCase.wantErr)
			}
			if list, ok := testCase.gs.Status.Lists[testCase.name]; ok {
				assert.Equal(t, testCase.want, list)
			}
		})
	}
}

func TestMergeRemoveDuplicates(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		str1 []string
		str2 []string
		want []string
	}{
		"empty string arrays": {
			str1: []string{},
			str2: []string{},
			want: []string{},
		},
		"no duplicates": {
			str1: []string{"one"},
			str2: []string{"two", "three"},
			want: []string{"one", "two", "three"},
		},
		"remove one duplicate": {
			str1: []string{"one", "one", "one"},
			str2: []string{"one", "one", "one"},
			want: []string{"one"},
		},
		"remove multiple duplicates": {
			str1: []string{"one", "two"},
			str2: []string{"two", "one"},
			want: []string{"one", "two"},
		},
	}

	for test, testCase := range testCases {
		t.Run(test, func(t *testing.T) {
			got := MergeRemoveDuplicates(testCase.str1, testCase.str2)
			assert.Equal(t, testCase.want, got)
		})
	}
}
