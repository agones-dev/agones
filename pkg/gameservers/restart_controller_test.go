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

package gameservers

import (
	"context"
	"strings"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned/fake"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type noopAPIHooks struct{}

func (noopAPIHooks) ValidateGameServerSpec(_ *agonesv1.GameServerSpec, _ *field.Path) field.ErrorList {
	return field.ErrorList{}
}
func (noopAPIHooks) ValidateScheduling(_ apis.SchedulingStrategy, _ *field.Path) field.ErrorList {
	return field.ErrorList{}
}
func (noopAPIHooks) MutateGameServerPod(_ *agonesv1.GameServerSpec, _ *corev1.Pod) error { return nil }
func (noopAPIHooks) SetEviction(_ *agonesv1.Eviction, _ *corev1.Pod) error               { return nil }

const (
	testNamespace   = "default"
	testGSName      = "test-game-server"
	everyMinuteCron = "* * * * *" // window always open when anchor is in past
	neverCron       = "0 0 1 1 *" // Jan 1st only — window never opens in tests
)

func enableRestartFeatureGate(t *testing.T) {
	t.Helper()
	err := runtime.ParseFeatures(string(runtime.FeatureGameServerScheduledRestart) + "=true")
	require.NoError(t, err,
		"Failed to enable %s — did you add it to featureDefaults in features.go?",
		runtime.FeatureGameServerScheduledRestart)
	t.Cleanup(func() {
		_ = runtime.ParseFeatures(string(runtime.FeatureGameServerScheduledRestart) + "=false")
	})
}

func newTestRestartController(t *testing.T) (
	*RestartController,
	*fake.Clientset,
	externalversions.SharedInformerFactory,
) {
	t.Helper()
	enableRestartFeatureGate(t)

	fakeAgonesClient := fake.NewSimpleClientset()
	fakeKubeClient := k8sfake.NewSimpleClientset()
	agonesInformerFactory := externalversions.NewSharedInformerFactory(fakeAgonesClient, 0)

	c := NewRestartController(
		healthcheck.NewHandler(),
		fakeKubeClient,
		fakeAgonesClient,
		agonesInformerFactory,
	)
	return c, fakeAgonesClient, agonesInformerFactory
}

func newReadyGS(rp *agonesv1.RestartPolicy) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testGSName,
			Namespace:         testNamespace,
			CreationTimestamp: metav1.Now(),
			Annotations:       map[string]string{},
		},
		Spec: agonesv1.GameServerSpec{
			RestartPolicy: rp,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "game-server", Image: "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.35"},
					},
				},
			},
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}
}

func seedLister(t *testing.T, factory externalversions.SharedInformerFactory, gs *agonesv1.GameServer) {
	t.Helper()
	err := factory.Agones().V1().GameServers().Informer().GetStore().Add(gs)
	require.NoError(t, err, "failed to seed GS into informer store")
}

func runReconcile(
	t *testing.T,
	c *RestartController,
	fakeClient *fake.Clientset,
	factory externalversions.SharedInformerFactory,
	gs *agonesv1.GameServer,
) *agonesv1.GameServer {
	t.Helper()

	seedLister(t, factory, gs)
	_, err := fakeClient.AgonesV1().GameServers(testNamespace).Create(
		context.Background(), gs, metav1.CreateOptions{},
	)
	require.NoError(t, err)
	fakeClient.ClearActions() // only count actions from reconcileRestart

	require.NoError(t, c.reconcileRestart(context.Background(), gs))

	return lastUpdateGS(t, fakeClient)
}

func lastUpdateGS(t *testing.T, fakeClient *fake.Clientset) *agonesv1.GameServer {
	t.Helper()
	actions := fakeClient.Actions()
	for i := len(actions) - 1; i >= 0; i-- {
		a := actions[i]
		if a.GetVerb() != "update" || a.GetResource().Resource != "gameservers" {
			continue
		}
		if ua, ok := a.(k8stesting.UpdateAction); ok {
			if gs, ok2 := ua.GetObject().(*agonesv1.GameServer); ok2 {
				return gs
			}
		}
	}
	return nil
}

func TestNoRestartBeforeWindow(t *testing.T) {
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{Schedule: neverCron})
	gs.CreationTimestamp = metav1.Now() // anchor=now → nextRestart=future Jan 1st

	updated := runReconcile(t, c, fakeClient, factory, gs)
	require.NotNil(t, updated, "expected one Update call to write next-restart annotation")

	_, pendingSet := updated.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]
	assert.False(t, pendingSet, "restart-pending-since must NOT be set before the window opens")

	_, nextSet := updated.Annotations[agonesv1.GameServerNextRestartAnnotation]
	assert.True(t, nextSet, "next-restart annotation must be written")
}

func TestRestartWhenIdle(t *testing.T) {
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{Schedule: everyMinuteCron})
	pastAnchor := time.Now().UTC().Add(-2 * time.Minute)
	gs.Annotations[agonesv1.GameServerNextRestartAnnotation] = pastAnchor.Format(time.RFC3339)
	gs.Status.State = agonesv1.GameServerStateReady
	gs.Status.Players = nil

	updated := runReconcile(t, c, fakeClient, factory, gs)
	require.NotNil(t, updated, "expected an Update call (advanceAnchor after idle restart)")

	nextAnnotation, ok := updated.Annotations[agonesv1.GameServerNextRestartAnnotation]
	assert.True(t, ok, "next-restart annotation must still be present after restart")

	advancedTime, err := time.Parse(time.RFC3339, nextAnnotation)
	require.NoError(t, err)
	assert.True(t, advancedTime.After(pastAnchor), "next-restart must be advanced past old window")

	_, stillPending := updated.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]
	assert.False(t, stillPending, "restart-pending-since must be cleared after successful restart")
}

func TestDeferWhenAllocated(t *testing.T) {
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{Schedule: everyMinuteCron})
	pastAnchor := time.Now().UTC().Add(-2 * time.Minute)
	gs.Annotations[agonesv1.GameServerNextRestartAnnotation] = pastAnchor.Format(time.RFC3339)
	gs.Status.State = agonesv1.GameServerStateAllocated

	updated := runReconcile(t, c, fakeClient, factory, gs)
	require.NotNil(t, updated, "expected an Update call to annotate restart-pending-since")

	pendingSince, ok := updated.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]
	assert.True(t, ok, "restart-pending-since must be set when restart is deferred")

	pendingTime, err := time.Parse(time.RFC3339, pendingSince)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().UTC(), pendingTime, 5*time.Second,
		"restart-pending-since must record approximately the current time")
}

func assertAnchorAdvanced(t *testing.T, updated *agonesv1.GameServer, windowOpenedAt time.Time, msg string) {
	t.Helper()
	require.NotNil(t, updated, msg)

	nextAnnotation, ok := updated.Annotations[agonesv1.GameServerNextRestartAnnotation]
	assert.True(t, ok, "next-restart annotation must be present")
	advancedTime, err := time.Parse(time.RFC3339, nextAnnotation)
	require.NoError(t, err)
	assert.True(t, advancedTime.After(windowOpenedAt), "anchor must advance past old window")

	_, stillPending := updated.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]
	assert.False(t, stillPending, "restart-pending-since must be cleared")
}

func TestSoftDeadlineSkip(t *testing.T) {
	softDeadline := metav1.Duration{Duration: 1 * time.Hour}
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{
		Schedule:             everyMinuteCron,
		SoftDeadlineDuration: &softDeadline,
	})
	windowOpenedAt := time.Now().UTC().Add(-2 * time.Minute)
	gs.Annotations[agonesv1.GameServerNextRestartAnnotation] = windowOpenedAt.Format(time.RFC3339)
	gs.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation] =
		time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339) // 2h > 1h soft deadline
	gs.Status.State = agonesv1.GameServerStateAllocated

	updated := runReconcile(t, c, fakeClient, factory, gs)
	assertAnchorAdvanced(t, updated, windowOpenedAt, "expected Update call (advanceAnchor after soft deadline)")
}

func TestHardDeadlineForce(t *testing.T) {
	hardDeadline := metav1.Duration{Duration: 24 * time.Hour}
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{
		Schedule:             everyMinuteCron,
		HardDeadlineDuration: &hardDeadline,
	})
	windowOpenedAt := time.Now().UTC().Add(-2 * time.Minute)
	gs.Annotations[agonesv1.GameServerNextRestartAnnotation] = windowOpenedAt.Format(time.RFC3339)
	gs.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation] =
		time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339) // 25h > 24h hard deadline
	gs.Status.State = agonesv1.GameServerStateAllocated

	updated := runReconcile(t, c, fakeClient, factory, gs)
	assertAnchorAdvanced(t, updated, windowOpenedAt, "expected Update call (hard deadline forced restart)")
}

func TestInvalidCronValidation(t *testing.T) {
	cases := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{name: "valid five-field cron", schedule: "0 4 * * *", wantErr: false},
		{name: "valid every-minute cron", schedule: "* * * * *", wantErr: false},
		{name: "invalid: only four fields", schedule: "0 4 * *", wantErr: true},
		{name: "invalid: natural language", schedule: "every day at midnight", wantErr: true},
		{name: "invalid: empty string", schedule: "", wantErr: true},
		{name: "invalid: minute 60 out of range", schedule: "60 4 * * *", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gss := &agonesv1.GameServerSpec{
				RestartPolicy: &agonesv1.RestartPolicy{Schedule: tc.schedule},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "game-server",
								Image: "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.35",
							},
						},
					},
				},
			}
			errs := gss.Validate(noopAPIHooks{}, "", field.NewPath("spec"))
			var rpErrs field.ErrorList
			for _, e := range errs {
				if strings.Contains(e.Field, "restartPolicy.schedule") {
					rpErrs = append(rpErrs, e)
				}
			}
			if tc.wantErr {
				assert.NotEmpty(t, rpErrs,
					"expected restartPolicy validation error for schedule %q, but got none", tc.schedule)
			} else {
				assert.Empty(t, rpErrs,
					"did NOT expect restartPolicy errors for schedule %q, but got: %v", tc.schedule, rpErrs)
			}
		})
	}
}

func TestNoRestartForTerminalState(t *testing.T) {
	for _, state := range []agonesv1.GameServerState{
		agonesv1.GameServerStateShutdown,
		agonesv1.GameServerStateError,
		agonesv1.GameServerStateUnhealthy,
	} {
		t.Run(string(state), func(t *testing.T) {
			c, fakeClient, factory := newTestRestartController(t)

			gs := newReadyGS(&agonesv1.RestartPolicy{Schedule: everyMinuteCron})
			gs.Status.State = state
			gs.Annotations[agonesv1.GameServerNextRestartAnnotation] =
				time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)

			// runReconcile clears the Create action for us.
			_ = runReconcile(t, c, fakeClient, factory, gs)

			for _, a := range fakeClient.Actions() {
				assert.NotEqual(t, "update", a.GetVerb(),
					"must not update a terminal-state GS (%s)", state)
			}
		})
	}
}

func TestNoRestartWithNoPolicy(t *testing.T) {
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(nil)
	seedLister(t, factory, gs)
	_, err := fakeClient.AgonesV1().GameServers(testNamespace).Create(
		context.Background(), gs, metav1.CreateOptions{},
	)
	require.NoError(t, err)
	fakeClient.ClearActions()

	require.NoError(t, c.syncGameServer(context.Background(), testNamespace+"/"+testGSName))

	for _, a := range fakeClient.Actions() {
		assert.NotEqual(t, "update", a.GetVerb(), "must not touch a GS with no RestartPolicy")
	}
}

func TestRestartDeferredWhenPlayersConnected(t *testing.T) {
	c, fakeClient, factory := newTestRestartController(t)

	gs := newReadyGS(&agonesv1.RestartPolicy{Schedule: everyMinuteCron})
	gs.Annotations[agonesv1.GameServerNextRestartAnnotation] =
		time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339)
	gs.Status.State = agonesv1.GameServerStateReady
	gs.Status.Players = &agonesv1.PlayerStatus{Count: 5, Capacity: 10}

	updated := runReconcile(t, c, fakeClient, factory, gs)
	require.NotNil(t, updated)
	_, ok := updated.Annotations[agonesv1.GameServerRestartPendingSinceAnnotation]
	assert.True(t, ok, "restart-pending-since must be set when active players block restart")
}
