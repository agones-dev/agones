// Copyright 2018 Google LLC All Rights Reserved.
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

// nolint:goconst
package fleets

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	v1 "agones.dev/agones/pkg/apis/agones/v1"
	applyconfigurations "agones.dev/agones/pkg/client/applyconfiguration/agones/v1"
	agonesv1clientset "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	agonesv1client "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/cloudproduct/generic"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestControllerSyncFleet(t *testing.T) {
	t.Parallel()

	t.Run("no gameserverset, create it", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		created := false
		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)

			created = true
			assert.True(t, metav1.IsControlledBy(gsSet, f))
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, created, "gameserverset should have been created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("gameserverset with the same number of replicas", func(t *testing.T) {
		t.Parallel()
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()

		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "4321"
		gsSet.Spec.Replicas = f.Spec.Replicas

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not be created")
			return true, nil, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not have been updated")
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("gameserverset with different number of replicas", func(t *testing.T) {
		if utilruntime.FeatureEnabled(utilruntime.FeatureRollingUpdateFix) {
			t.SkipNow()
		}

		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = f.Spec.Replicas + 10
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("gameserverset with different scheduling", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = f.Spec.Replicas
		gsSet.Spec.Scheduling = apis.Distributed
		updated := false

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, f.Spec.Replicas, gsSet.Spec.Replicas)
			assert.Equal(t, f.Spec.Scheduling, gsSet.Spec.Scheduling)
			return true, gsSet, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("gameserverset with different image details", func(t *testing.T) {
		if utilruntime.FeatureEnabled(utilruntime.FeatureRollingUpdateFix) {
			t.SkipNow()
		}

		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
		f.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 5555}}
		f.Status.ReadyReplicas = 5
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "4321"
		gsSet.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 7777}}
		gsSet.Spec.Replicas = f.Spec.Replicas
		gsSet.Spec.Scheduling = f.Spec.Scheduling
		gsSet.Status.Replicas = 5
		updated := false
		created := false

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, int32(2), gsSet.Spec.Replicas)
			assert.Equal(t, f.Spec.Template.Spec.Ports[0].HostPort, gsSet.Spec.Template.Spec.Ports[0].HostPort)

			return true, gsSet, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			// separate update of status subresource
			if ua.GetSubresource() != "" {
				assert.Equal(t, ua.GetSubresource(), "status")
				return true, nil, nil
			}
			// update main resource
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, int32(4), gsSet.Spec.Replicas)
			assert.Equal(t, "gsSet1", gsSet.ObjectMeta.Name)

			return true, gsSet, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		assert.True(t, updated, "gameserverset should have been updated")
		assert.True(t, created, "gameserverset should have been created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("fleet marked for deletion shouldn't take any action on gameserver sets", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
		f.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 5555}}
		f.Status.ReadyReplicas = 5
		f.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}

		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.Name = "gsSet1"
		gsSet.ObjectMeta.UID = "4321"
		gsSet.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 7777}}
		gsSet.Spec.Replicas = f.Spec.Replicas
		gsSet.Spec.Scheduling = f.Spec.Scheduling
		gsSet.Status.Replicas = 5

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not have been created")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserverset should not have been updated")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("error on getting fleet", func(t *testing.T) {
		c, _ := newFakeController()
		c.fleetLister = &fakeFleetListerWithErr{}

		err := c.syncFleet(context.Background(), "default/fleet-1")
		assert.EqualError(t, err, "error retrieving fleet fleet-1 from namespace default: err-from-namespace-lister")
	})

	t.Run("error on getting list of GS", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		c.gameServerSetLister = &fakeGSSListerWithErr{}

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.EqualError(t, err, "error listing gameserversets for fleet fleet-1: random-err")
	})

	t.Run("fleet not found", func(t *testing.T) {
		c, _ := newFakeController()

		err := c.syncFleet(context.Background(), "default/fleet-1")
		assert.Nil(t, err)
	})

	t.Run("fleet invalid strategy type", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()
		f.Spec.Strategy.Type = "invalid-strategy-type"

		gsSet := f.GameServerSet()
		// make gsSet.Spec.Template and f.Spec.Template different in order to make 'rest' list not empty
		gsSet.Spec.Template.Name = "qqqqqqqqqqqqqqqqqqq"
		// make sure there is at least one replica, or the logic will escape before the check.
		gsSet.Spec.Replicas = 1

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.EqualError(t, err, "unexpected deployment strategy type: invalid-strategy-type")
	})

	t.Run("error on deleteEmptyGameServerSets", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		gsSet := f.GameServerSet()
		// make gsSet.Spec.Template and f.Spec.Template different in order to make 'rest' list not empty
		gsSet.Spec.Template.Name = "qqqqqqqqqqqqqqqqqqq"

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("delete", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.EqualError(t, err, "error updating gameserverset : random-err")
	})

	t.Run("error on upsertGameServerSet", func(t *testing.T) {
		f := defaultFixture()
		c, m := newFakeController()

		gsSet := f.GameServerSet()

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})

		m.AgonesClient.AddReactor("create", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.syncFleet(ctx, "default/fleet-1")
		assert.EqualError(t, err, "error creating gameserverset for fleet fleet-1: random-err")
	})
}

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	ext := newFakeExtensions()

	t.Run("invalid JSON", func(t *testing.T) {
		raw, err := json.Marshal([]byte(`1`))
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		_, err = ext.creationValidationHandler(review)
		assert.EqualError(t, err, "error unmarshalling Fleet json after schema validation: \"MQ==\": json: cannot unmarshal string into Go value of type v1.Fleet")
	})

	t.Run("invalid fleet", func(t *testing.T) {
		fixture := agonesv1.Fleet{}

		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, "Failure", result.Response.Result.Status)
	})

	t.Run("valid fleet", func(t *testing.T) {
		gsSpec := *defaultGSSpec()
		f := defaultFixture()
		f.Spec.Template = gsSpec

		raw, err := json.Marshal(f)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		assert.True(t, result.Response.Allowed)
	})
}

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	t.Run("ok scenario", func(t *testing.T) {
		ext := newFakeExtensions()
		fixture := agonesv1.Fleet{}

		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := ext.creationMutationHandler(review)
		require.NoError(t, err)
		assert.True(t, result.Response.Allowed)
		assert.Equal(t, admissionv1.PatchTypeJSONPatch, *result.Response.PatchType)

		patch := &jsonpatch.ByPath{}
		err = json.Unmarshal(result.Response.Patch, patch)
		require.NoError(t, err)

		assertContains := func(patch *jsonpatch.ByPath, op jsonpatch.JsonPatchOperation) {
			found := false
			for _, p := range *patch {
				if assert.ObjectsAreEqualValues(p, op) {
					found = true
				}
			}

			assert.True(t, found, "Could not find operation %#v in patch %v", op, *patch)
		}

		assertContains(patch, jsonpatch.JsonPatchOperation{Operation: "add", Path: "/spec/strategy/type", Value: "RollingUpdate"})
	})

	t.Run("invalid JSON", func(t *testing.T) {
		ext := newFakeExtensions()
		raw, err := json.Marshal([]byte(`1`))
		require.NoError(t, err)
		review := getAdmissionReview(raw)

		result, err := ext.creationMutationHandler(review)
		assert.NoError(t, err)
		require.Nil(t, result.Response.PatchType)
	})
}

func TestControllerRun(t *testing.T) {
	t.Parallel()

	fleet := defaultFixture()
	c, m := newFakeController()
	received := make(chan string)
	defer close(received)

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})

	fleetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(fleetWatch, nil))

	gsSetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(gsSetWatch, nil))

	c.workerqueue.SyncHandler = func(_ context.Context, name string) error {
		received <- name
		return nil
	}

	ctx, cancel := agtesting.StartInformers(m, c.fleetSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx, 1)
		assert.Nil(t, err)
	}()

	f := func() string {
		select {
		case result := <-received:
			return result
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "timeout occurred")
		}
		return ""
	}

	expected, err := cache.MetaNamespaceKeyFunc(fleet)
	require.NoError(t, err)

	// test adding fleet
	fleetWatch.Add(fleet.DeepCopy())
	assert.Equal(t, expected, f())

	// test updating fleet
	fCopy := fleet.DeepCopy()
	fCopy.Spec.Replicas += 10
	fleetWatch.Modify(fCopy)
	assert.Equal(t, expected, f())

	// test add/update of gameserver set
	gsSet := fleet.GameServerSet()
	gsSet.ObjectMeta.Name = "gs1"
	gsSet.ObjectMeta.GenerateName = ""
	gsSetWatch.Add(gsSet)
	assert.Equal(t, expected, f())

	gsSet.Spec.Replicas += 10
	gsSetWatch.Modify(gsSet)
	assert.Equal(t, expected, f())
}

func TestControllerUpdateFleetStatus(t *testing.T) {
	t.Parallel()

	t.Run("update, no errors", func(t *testing.T) {
		fleet := defaultFixture()
		c, m := newFakeController()

		gsSet1 := fleet.GameServerSet()
		gsSet1.ObjectMeta.Name = "gsSet1"
		gsSet1.Status.Replicas = 3
		gsSet1.Status.ReadyReplicas = 2
		gsSet1.Status.ReservedReplicas = 4
		gsSet1.Status.AllocatedReplicas = 1

		gsSet2 := fleet.GameServerSet()
		// nolint:goconst
		gsSet2.ObjectMeta.Name = "gsSet2"
		gsSet2.Status.Replicas = 5
		gsSet2.Status.ReadyReplicas = 5
		gsSet2.Status.ReservedReplicas = 3
		gsSet2.Status.AllocatedReplicas = 2

		m.AgonesClient.AddReactor("list", "gameserversets",
			func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
			})

		updated := false
		m.AgonesClient.AddReactor("update", "fleets",
			func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				fleet := ua.GetObject().(*agonesv1.Fleet)

				assert.Equal(t, gsSet1.Status.Replicas+gsSet2.Status.Replicas, fleet.Status.Replicas)
				assert.Equal(t, gsSet1.Status.ReadyReplicas+gsSet2.Status.ReadyReplicas, fleet.Status.ReadyReplicas)
				assert.Equal(t, gsSet1.Status.ReservedReplicas+gsSet2.Status.ReservedReplicas, fleet.Status.ReservedReplicas)
				assert.Equal(t, gsSet1.Status.AllocatedReplicas+gsSet2.Status.AllocatedReplicas, fleet.Status.AllocatedReplicas)
				return true, fleet, nil
			})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
		defer cancel()

		err := c.updateFleetStatus(ctx, fleet)
		assert.Nil(t, err)
		assert.True(t, updated)
	})

	t.Run("list gameservers returns an error", func(t *testing.T) {
		fleet := defaultFixture()
		c, _ := newFakeController()
		c.gameServerSetLister = &fakeGSSListerWithErr{}

		err := c.updateFleetStatus(context.Background(), fleet)
		assert.EqualError(t, err, "error listing gameserversets for fleet fleet-1: random-err")
	})

	t.Run("fleets getter returns an error", func(t *testing.T) {
		fleet := defaultFixture()
		c, _ := newFakeController()

		c.fleetGetter = &fakeFleetsGetterWithErr{}

		err := c.updateFleetStatus(context.Background(), fleet)

		assert.EqualError(t, err, "err-from-fleet-getter")
	})

}

func TestControllerUpdateFleetPlayerStatus(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeaturePlayerTracking)+"=true"))

	fleet := defaultFixture()
	c, m := newFakeController()

	gsSet1 := fleet.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Status.Players = &agonesv1.AggregatedPlayerStatus{
		Count:    5,
		Capacity: 10,
	}

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Players = &agonesv1.AggregatedPlayerStatus{
		Count:    10,
		Capacity: 20,
	}

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
		})

	updated := false
	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*agonesv1.Fleet)

			assert.Equal(t, gsSet1.Status.Players.Count+gsSet2.Status.Players.Count, fleet.Status.Players.Count)
			assert.Equal(t, gsSet1.Status.Players.Capacity+gsSet2.Status.Players.Capacity, fleet.Status.Players.Capacity)

			return true, fleet, nil
		})

	ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
	defer cancel()

	err := c.updateFleetStatus(ctx, fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

// nolint:dupl // Linter errors on lines are duplicate of TestControllerUpdateFleetListStatus
func TestControllerUpdateFleetCounterStatus(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

	fleet := defaultFixture()
	c, m := newFakeController()

	gsSet1 := fleet.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Status.Counters = map[string]agonesv1.AggregatedCounterStatus{
		"fullCounter": {
			AllocatedCount:    9223372036854775807,
			AllocatedCapacity: 9223372036854775807,
			Capacity:          9223372036854775807,
			Count:             9223372036854775807,
		},
		"anotherCounter": {
			AllocatedCount:    11,
			AllocatedCapacity: 20,
			Capacity:          100,
			Count:             42,
		},
	}

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Counters = map[string]agonesv1.AggregatedCounterStatus{
		"fullCounter": {
			AllocatedCount:    100,
			AllocatedCapacity: 100,
			Capacity:          100,
			Count:             100,
		},
		"anotherCounter": {
			AllocatedCount:    0,
			AllocatedCapacity: 0,
			Capacity:          100,
			Count:             0,
		},
		"thirdCounter": {
			AllocatedCount:    21,
			AllocatedCapacity: 30,
			Capacity:          400,
			Count:             21,
		},
	}

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
		})

	updated := false
	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*agonesv1.Fleet)

			assert.Equal(t, int64(9223372036854775807), fleet.Status.Counters["fullCounter"].AllocatedCount)
			assert.Equal(t, int64(9223372036854775807), fleet.Status.Counters["fullCounter"].AllocatedCapacity)
			assert.Equal(t, int64(9223372036854775807), fleet.Status.Counters["fullCounter"].Capacity)
			assert.Equal(t, int64(9223372036854775807), fleet.Status.Counters["fullCounter"].Count)

			assert.Equal(t, int64(11), fleet.Status.Counters["anotherCounter"].AllocatedCount)
			assert.Equal(t, int64(20), fleet.Status.Counters["anotherCounter"].AllocatedCapacity)
			assert.Equal(t, int64(200), fleet.Status.Counters["anotherCounter"].Capacity)
			assert.Equal(t, int64(42), fleet.Status.Counters["anotherCounter"].Count)

			assert.Equal(t, int64(21), fleet.Status.Counters["thirdCounter"].AllocatedCount)
			assert.Equal(t, int64(30), fleet.Status.Counters["thirdCounter"].AllocatedCapacity)
			assert.Equal(t, int64(400), fleet.Status.Counters["thirdCounter"].Capacity)
			assert.Equal(t, int64(21), fleet.Status.Counters["thirdCounter"].Count)

			return true, fleet, nil
		})

	ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
	defer cancel()

	err := c.updateFleetStatus(ctx, fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

// nolint:dupl // Linter errors on lines are duplicate of TestControllerUpdateFleetCounterStatus
func TestControllerUpdateFleetListStatus(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

	fleet := defaultFixture()
	c, m := newFakeController()

	gsSet1 := fleet.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Status.Lists = map[string]agonesv1.AggregatedListStatus{
		"fullList": {
			AllocatedCount:    1000,
			AllocatedCapacity: 1000,
			Capacity:          1000,
			Count:             1000,
		},
		"anotherList": {
			AllocatedCount:    11,
			AllocatedCapacity: 100,
			Capacity:          100,
			Count:             11,
		},
		"thirdList": {
			AllocatedCount:    1,
			AllocatedCapacity: 20,
			Capacity:          30,
			Count:             4,
		},
	}

	gsSet2 := fleet.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Status.Lists = map[string]agonesv1.AggregatedListStatus{
		"fullList": {
			AllocatedCount:    200,
			AllocatedCapacity: 200,
			Capacity:          200,
			Count:             200,
		},
		"anotherList": {
			AllocatedCount:    1,
			AllocatedCapacity: 10,
			Capacity:          100,
			Count:             11,
		},
	}

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2}}, nil
		})

	updated := false
	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*agonesv1.Fleet)

			assert.Equal(t, int64(1200), fleet.Status.Lists["fullList"].AllocatedCount)
			assert.Equal(t, int64(1200), fleet.Status.Lists["fullList"].AllocatedCapacity)
			assert.Equal(t, int64(1200), fleet.Status.Lists["fullList"].Capacity)
			assert.Equal(t, int64(1200), fleet.Status.Lists["fullList"].Count)

			assert.Equal(t, int64(12), fleet.Status.Lists["anotherList"].AllocatedCount)
			assert.Equal(t, int64(110), fleet.Status.Lists["anotherList"].AllocatedCapacity)
			assert.Equal(t, int64(200), fleet.Status.Lists["anotherList"].Capacity)
			assert.Equal(t, int64(22), fleet.Status.Lists["anotherList"].Count)

			assert.Equal(t, int64(1), fleet.Status.Lists["thirdList"].AllocatedCount)
			assert.Equal(t, int64(20), fleet.Status.Lists["thirdList"].AllocatedCapacity)
			assert.Equal(t, int64(30), fleet.Status.Lists["thirdList"].Capacity)
			assert.Equal(t, int64(4), fleet.Status.Lists["thirdList"].Count)

			return true, fleet, nil
		})

	ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
	defer cancel()

	err := c.updateFleetStatus(ctx, fleet)
	assert.Nil(t, err)
	assert.True(t, updated)
}

// Test that the aggregated Counters and Lists are removed from the Fleet status if the
// FeatureCountsAndLists flag is set to false.
func TestFleetDropCountsAndListsStatus(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	f := defaultFixture()
	defaultFleetName := "default/fleet-1"
	c, m := newFakeController()

	gss := f.GameServerSet()
	gss.ObjectMeta.Name = "gsSet1"
	gss.ObjectMeta.UID = "4321"
	gss.Spec.Replicas = f.Spec.Replicas
	gss.Status.Counters = map[string]agonesv1.AggregatedCounterStatus{
		"aCounter": {
			AllocatedCount:    1,
			AllocatedCapacity: 10,
			Capacity:          1000,
			Count:             100,
		},
	}
	gss.Status.Lists = map[string]agonesv1.AggregatedListStatus{
		"aList": {
			AllocatedCount:    10,
			AllocatedCapacity: 100,
			Capacity:          10000,
			Count:             1000,
		},
	}

	flag := ""
	updated := false

	m.AgonesClient.AddReactor("list", "gameserversets",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gss}}, nil
		})

	m.AgonesClient.AddReactor("list", "fleets",
		func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

	m.AgonesClient.AddReactor("update", "fleets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			fleet := ua.GetObject().(*agonesv1.Fleet)
			updated = true

			switch flag {
			case string(utilruntime.FeatureCountsAndLists) + "=true":
				assert.Equal(t, gss.Status.Counters, fleet.Status.Counters)
				assert.Equal(t, gss.Status.Lists, fleet.Status.Lists)
			case string(utilruntime.FeatureCountsAndLists) + "=false":
				assert.Nil(t, fleet.Status.Counters)
				assert.Nil(t, fleet.Status.Lists)
			default:
				return false, fleet, errors.Errorf("Flag string(utilruntime.FeatureCountsAndLists) should be set")
			}
			return true, fleet, nil
		})

	ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.gameServerSetSynced)
	defer cancel()

	// Expect starting fleet to have Aggregated Counter and List Statuses
	flag = string(utilruntime.FeatureCountsAndLists) + "=true"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err := c.syncFleet(ctx, defaultFleetName)
	assert.NoError(t, err)
	assert.True(t, updated)

	updated = false
	flag = string(utilruntime.FeatureCountsAndLists) + "=false"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err = c.syncFleet(ctx, defaultFleetName)
	assert.NoError(t, err)
	assert.True(t, updated)

	updated = false
	flag = string(utilruntime.FeatureCountsAndLists) + "=true"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err = c.syncFleet(ctx, defaultFleetName)
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestControllerFilterGameServerSetByActive(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	c, _ := newFakeController()
	// the same GameServer Template
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"

	// different GameServer Template
	gsSet2 := f.GameServerSet()
	gsSet2.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 9999}}

	// one active
	active, rest := c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Equal(t, gsSet1, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet2}, rest)

	// none active
	gsSet1.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 9999}}
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet1, gsSet2}, rest)
}

func TestControllerFilterGameServerSetByActiveDuplicates(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	c, _ := newFakeController()

	now := time.Now()

	// Three GameServerSets with the same template (simulates duplicate creation
	// caused by informer cache lag).
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.ObjectMeta.UID = "uid-1"
	gsSet1.ObjectMeta.CreationTimestamp = metav1.NewTime(now.Add(-2 * time.Minute))

	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.ObjectMeta.UID = "uid-2"
	gsSet2.ObjectMeta.CreationTimestamp = metav1.NewTime(now.Add(-1 * time.Minute))

	gsSet3 := f.GameServerSet()
	gsSet3.ObjectMeta.Name = "gsSet3"
	gsSet3.ObjectMeta.UID = "uid-3"
	gsSet3.ObjectMeta.CreationTimestamp = metav1.NewTime(now)

	// Different template GameServerSet (from a previous Fleet spec).
	gsSet4 := f.GameServerSet()
	gsSet4.ObjectMeta.Name = "gsSet4"
	gsSet4.ObjectMeta.UID = "uid-4"
	gsSet4.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{HostPort: 9999}}

	// With 3 duplicate-template GSS, oldest should be active, others in rest.
	active, rest := c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet2, gsSet3})
	assert.Equal(t, gsSet1, active, "oldest GameServerSet should be active")
	assert.Len(t, rest, 2)
	assert.Contains(t, rest, gsSet2)
	assert.Contains(t, rest, gsSet3)

	// Order should not matter — even if newest comes first in the list,
	// oldest should still be active.
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet3, gsSet1, gsSet2})
	assert.Equal(t, gsSet1, active, "oldest GameServerSet should be active regardless of list order")
	assert.Len(t, rest, 2)
	assert.Contains(t, rest, gsSet2)
	assert.Contains(t, rest, gsSet3)

	// Mix of matching and non-matching templates.
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet3, gsSet4, gsSet1, gsSet2})
	assert.Equal(t, gsSet1, active, "oldest matching GameServerSet should be active")
	assert.Len(t, rest, 3)
	assert.Contains(t, rest, gsSet2)
	assert.Contains(t, rest, gsSet3)
	assert.Contains(t, rest, gsSet4)
}

func TestControllerRecreateDeployment(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	f.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	f.Spec.Replicas = 10
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Spec.Replicas = 10
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Spec.Replicas = 0
	gsSet2.Status.AllocatedReplicas = 1

	t.Run("ok scenario", func(t *testing.T) {
		c, m := newFakeController()
		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
			assert.Equal(t, int32(0), gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		replicas, err := c.recreateDeployment(context.Background(), f, []*agonesv1.GameServerSet{gsSet1, gsSet2})

		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, f.Spec.Replicas-1, replicas)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("error on update", func(t *testing.T) {
		c, m := newFakeController()
		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		_, err := c.recreateDeployment(context.Background(), f, []*agonesv1.GameServerSet{gsSet1, gsSet2})

		assert.EqualError(t, err, "error updating gameserverset gsSet1: random-err")
	})
}

func TestControllerApplyDeploymentStrategy(t *testing.T) {
	t.Parallel()

	type expected struct {
		inactiveReplicas int32
		replicas         int32
	}

	fixtures := map[string]struct {
		strategyType         appsv1.DeploymentStrategyType
		gsSet1StatusReplicas int32
		gsSet2StatusReplicas int32
		expected             expected
	}{
		string(appsv1.RecreateDeploymentStrategyType): {
			strategyType:         appsv1.RecreateDeploymentStrategyType,
			gsSet1StatusReplicas: 0,
			gsSet2StatusReplicas: 0,
			expected: expected{
				inactiveReplicas: 0,
				replicas:         10,
			},
		},
		string(appsv1.RollingUpdateDeploymentStrategyType): {
			strategyType:         appsv1.RollingUpdateDeploymentStrategyType,
			gsSet1StatusReplicas: 10,
			gsSet2StatusReplicas: 1,
			expected: expected{
				inactiveReplicas: 8,
				replicas:         2,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			if utilruntime.FeatureEnabled(utilruntime.FeatureRollingUpdateFix) {
				t.SkipNow()
			}

			f := defaultFixture()
			f.Spec.Strategy.Type = v.strategyType
			f.Spec.Replicas = 10

			gsSet1 := f.GameServerSet()
			gsSet1.ObjectMeta.Name = "gsSet1"
			gsSet1.Spec.Replicas = 10
			gsSet1.Status.Replicas = v.gsSet1StatusReplicas

			gsSet2 := f.GameServerSet()
			gsSet2.ObjectMeta.Name = "gsSet2"
			gsSet2.Spec.Replicas = 0
			gsSet2.Status.Replicas = v.gsSet2StatusReplicas

			c, m := newFakeController()

			updated := false
			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsSet := ua.GetObject().(*agonesv1.GameServerSet)
				assert.Equal(t, gsSet1.ObjectMeta.Name, gsSet.ObjectMeta.Name)
				assert.Equal(t, v.expected.inactiveReplicas, gsSet.Spec.Replicas)

				return true, gsSet, nil
			})

			replicas, err := c.applyDeploymentStrategy(context.Background(), f, f.GameServerSet(), []*agonesv1.GameServerSet{gsSet1, gsSet2})
			require.NoError(t, err)
			assert.True(t, updated, "update should happen")
			assert.Equal(t, v.expected.replicas, replicas)
		})
	}

	t.Run("a single gameserverset", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Replicas = 10

		gsSet1 := f.GameServerSet()
		gsSet1.ObjectMeta.Name = "gsSet1"

		c, _ := newFakeController()

		replicas, err := c.applyDeploymentStrategy(context.Background(), f, f.GameServerSet(), []*agonesv1.GameServerSet{})
		require.NoError(t, err)
		assert.Equal(t, f.Spec.Replicas, replicas)
	})

	t.Run("rest gameservers that are already scaled down", func(t *testing.T) {
		f := defaultFixture()
		f.Spec.Replicas = 10

		gsSet1 := f.GameServerSet()
		gsSet1.ObjectMeta.Name = "gsSet1"
		gsSet1.Spec.Replicas = 0
		gsSet1.Status.AllocatedReplicas = 1

		c, _ := newFakeController()

		replicas, err := c.applyDeploymentStrategy(context.Background(), f, f.GameServerSet(), []*agonesv1.GameServerSet{gsSet1})
		require.NoError(t, err)
		assert.Equal(t, int32(9), replicas)
	})
}

func TestControllerUpsertGameServerSet(t *testing.T) {
	t.Parallel()
	f := defaultFixture()
	replicas := int32(10)

	t.Run("insert", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		created := false
		m.AgonesClient.AddReactor("create", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(context.Background(), f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, created, "Should be created")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "CreatingGameServerSet")
	})

	t.Run("update", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas + 10
		update := false

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			update = true
			ca := action.(k8stesting.UpdateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, replicas, gsSet.Spec.Replicas)

			return true, gsSet, nil
		})

		err := c.upsertGameServerSet(context.Background(), f, gsSet, replicas)
		assert.Nil(t, err)

		assert.True(t, update, "Should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
	})

	t.Run("error updating gss replicas", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas + 10

		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		err := c.upsertGameServerSet(context.Background(), f, gsSet, replicas)

		assert.EqualError(t, err, "error updating replicas for gameserverset for fleet fleet-1: random-err")
	})

	t.Run("error on gs status update", func(t *testing.T) {
		c, m := newFakeController()
		gsSet := f.GameServerSet()

		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		err := c.upsertGameServerSet(context.Background(), f, gsSet, replicas)

		assert.EqualError(t, err, "error updating status of gameserverset for fleet fleet-1: random-err")
	})

	t.Run("nothing happens, nil is returned", func(t *testing.T) {
		t.Parallel()

		c, m := newFakeController()
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		gsSet.Spec.Replicas = replicas

		m.AgonesClient.AddReactor("create", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not create")
			return false, nil, nil
		})
		m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not update")
			return false, nil, nil
		})

		err := c.upsertGameServerSet(context.Background(), f, gsSet, replicas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update Priorities", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

		c, m := newFakeController()
		// Default GameServerSet has no Priorities
		gsSet := f.GameServerSet()
		gsSet.ObjectMeta.UID = "1234"
		// Add Priorities to the Fleet
		f.Spec.Priorities = []agonesv1.Priority{
			{
				Type:  "List",
				Key:   "Baz",
				Order: "Ascending",
			}}
		update := false

		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			update = true
			ca := action.(k8stesting.UpdateAction)
			gsSet := ca.GetObject().(*agonesv1.GameServerSet)
			assert.Equal(t, agonesv1.Priority{Type: "List", Key: "Baz", Order: "Ascending"}, gsSet.Spec.Priorities[0])
			return true, gsSet, nil
		})

		// Update Priorities on the GameServerSet to match the Fleet
		err := c.upsertGameServerSet(context.Background(), f, gsSet, gsSet.Spec.Replicas)
		assert.Nil(t, err)

		assert.True(t, update, "Should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "UpdatingGameServerSet")
	})
}

func TestResourcesRequestsAndLimits(t *testing.T) {
	t.Parallel()

	gsSpec := *defaultGSSpec()
	c, _ := newFakeController()
	f := defaultFixture()
	f.Spec.Template = gsSpec
	f.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1000m")

	// Semantically equal definition, 1 == 1000m CPU
	gsSet1 := f.GameServerSet()
	gsSet1.Spec.Template = gsSpec
	gsSet1.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1")

	// Absolutely different GameServer Spec, 1.1 CPU
	gsSet3 := f.GameServerSet()
	gsSet3.Spec.Template = *gsSpec.DeepCopy()
	gsSet3.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1.1")
	active, rest := c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet1, gsSet3})
	assert.Equal(t, gsSet1, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet3}, rest)

	gsSet2 := f.GameServerSet()
	gsSet2.Spec.Template = *gsSpec.DeepCopy()
	gsSet2.Spec.Template.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1000m")
	active, rest = c.filterGameServerSetByActive(f, []*agonesv1.GameServerSet{gsSet2, gsSet3})
	assert.Equal(t, gsSet2, active)
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet3}, rest)
}

func TestControllerDeleteEmptyGameServerSets(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet1.Spec.Replicas = 10
	gsSet1.Status.Replicas = 10
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet2.Spec.Replicas = 0
	gsSet2.Status.Replicas = 0

	c, m := newFakeController()
	deleted := false

	m.AgonesClient.AddReactor("delete", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleted = true
		da := action.(k8stesting.DeleteAction)
		assert.Equal(t, gsSet2.ObjectMeta.Name, da.GetName())
		return true, nil, nil
	})

	err := c.deleteEmptyGameServerSets(context.Background(), f, []*agonesv1.GameServerSet{gsSet1, gsSet2})
	assert.Nil(t, err)
	assert.True(t, deleted, "delete should happen")
}

func TestControllerRollingUpdateDeploymentNoInactiveGSSNoErrors(t *testing.T) {
	t.Parallel()

	f := defaultFixture()

	f.Spec.Replicas = 100

	active := f.GameServerSet()
	active.ObjectMeta.Name = "active"

	c, _ := newFakeController()

	replicas, err := c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{})
	assert.Nil(t, err)
	assert.Equal(t, int32(25), replicas)
}

// Test when replicas is negative value(0 replicas - 1 allocated = -1)
func TestControllerRollingUpdateDeploymentNegativeReplica(t *testing.T) {
	t.Parallel()

	// Create Fleet with replicas: 5
	f := defaultFixture()
	f.Status.Replicas = 5
	// Allocate 1 gameserver
	f.Status.AllocatedReplicas = 1
	f.Status.ReadyReplicas = 4

	// Edit fleet spec.template.spec and create new gameserverset
	f.Spec.Template.Spec.Ports = []agonesv1.GameServerPort{{
		ContainerPort: 6000,
		Name:          "gameport",
		PortPolicy:    agonesv1.Dynamic,
		Protocol:      corev1.ProtocolUDP,
	}}

	// old gameserverset has only allocated gameserver
	inactive := f.GameServerSet()
	inactive.ObjectMeta.Name = "inactive"
	inactive.Spec.Replicas = 0
	inactive.Status.ReadyReplicas = 0
	inactive.Status.Replicas = 1
	inactive.Status.AllocatedReplicas = 1

	// new gameserverset has 4 gameserver(replicas:5 - sumAllocated:1)
	active := f.GameServerSet()
	active.ObjectMeta.Name = "active"
	active.Spec.Replicas = 4
	active.Status.ReadyReplicas = 4
	active.Status.Replicas = 4
	active.Status.AllocatedReplicas = 0

	c, m := newFakeController()

	// triggered inside rollingUpdateRest
	m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ca := action.(k8stesting.UpdateAction)
		gsSet := ca.GetObject().(*agonesv1.GameServerSet)
		assert.Equal(t, int32(4), gsSet.Spec.Replicas)
		assert.Equal(t, int32(5), f.Spec.Replicas)

		return true, nil, errors.Errorf("error updating replicas for gameserverset for fleet %s", f.Name)
	})

	// assert the active gameserverset's replicas when active and inactive gameserversets exist
	expected := f.Spec.Replicas - f.Status.AllocatedReplicas
	replicas, err := c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{inactive})
	f.Status.ReadyReplicas = replicas
	assert.NoError(t, err)
	assert.Equal(t, expected, replicas)

	// happened scale down to 0 by manual operation
	f.Spec.Replicas = 0
	// rolling update to scale 0
	replicas, err = c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{inactive})
	f.Status.ReadyReplicas = replicas
	// assert no error, when fleet replicas is negative value(0 replicas - 1 allocated = -1)
	assert.NoError(t, err)
	// assert replicas 0, after user scales replicas to 0
	assert.Equal(t, int32(0), replicas)
}

func TestControllerRollingUpdateDeploymentGSSUpdateFailedErrExpected(t *testing.T) {
	t.Parallel()

	f := defaultFixture()
	f.Spec.Replicas = 75
	f.Status.ReadyReplicas = 75

	active := f.GameServerSet()
	active.ObjectMeta.Name = "active"
	active.Spec.Replicas = 75
	active.Status.ReadyReplicas = 75
	active.Status.Replicas = 75

	inactive := f.GameServerSet()
	inactive.ObjectMeta.Name = "inactive"
	inactive.Spec.Replicas = 10
	inactive.Status.ReadyReplicas = 10
	inactive.Status.Replicas = 10
	inactive.Status.AllocatedReplicas = 5

	c, m := newFakeController()

	// triggered inside rollingUpdateRest
	m.AgonesClient.AddReactor("update", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("random-err")
	})

	_, err := c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{inactive})
	assert.EqualError(t, err, "error updating gameserverset inactive: random-err")
}

func TestRollingUpdateOnReady(t *testing.T) {
	type expected struct {
		inactiveSpecReplicas int32
		replicas             int32
		updated              bool
	}

	fixtures := map[string]struct {
		activeStatusReadyReplicas   int32
		inactiveStatusReadyReplicas int32
		allocatedReplicas           int32
		expected                    expected
	}{
		"not enough Ready GameServers - do not scale down rest GameServerSet": {
			activeStatusReadyReplicas:   10,
			inactiveStatusReadyReplicas: 10,
			expected: expected{
				updated:              false,
				inactiveSpecReplicas: 0,
				replicas:             75,
			},
		},
		"enough Ready GameServers - scale down rest GameServerSet to Allocated": {
			activeStatusReadyReplicas:   70,
			inactiveStatusReadyReplicas: 5,
			allocatedReplicas:           5,
			expected: expected{
				updated:              true,
				inactiveSpecReplicas: 5,
				replicas:             70,
			},
		},
		"enough Ready GameServers - scale down rest GameServerSet to 0": {
			activeStatusReadyReplicas:   70,
			inactiveStatusReadyReplicas: 10,
			allocatedReplicas:           0,
			expected: expected{
				updated:              true,
				inactiveSpecReplicas: 0,
				replicas:             75,
			},
		},
		"scale down rest GameServerSet to > 0": {
			// 75 - 19 = 56 is minimum number of gameservers
			// scaling 58 - 56 = -2 gameservers
			// initial 10 - 2 = 8
			activeStatusReadyReplicas:   50,
			inactiveStatusReadyReplicas: 8,
			allocatedReplicas:           0,
			expected: expected{
				updated:              true,
				inactiveSpecReplicas: 8,
				replicas:             75,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			c, m := newFakeController()

			f := defaultFixture()
			f.Spec.Replicas = 75
			f.Status.ReadyReplicas = v.activeStatusReadyReplicas + v.inactiveStatusReadyReplicas

			active := f.GameServerSet()
			active.ObjectMeta.Name = "active"
			active.Spec.Replicas = 75
			active.Status.Replicas = 75
			active.Status.ReadyReplicas = v.activeStatusReadyReplicas

			inactive := f.GameServerSet()
			inactive.ObjectMeta.Name = "inactive"
			inactive.Spec.Replicas = 10
			inactive.Status.Replicas = 10
			inactive.Status.ReadyReplicas = v.inactiveStatusReadyReplicas
			inactive.Status.AllocatedReplicas = v.allocatedReplicas
			updated := false
			// triggered inside rollingUpdateRest
			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsSet := ua.GetObject().(*agonesv1.GameServerSet)
				assert.Equal(t, inactive.ObjectMeta.Name, gsSet.ObjectMeta.Name)
				assert.Equal(t, v.expected.inactiveSpecReplicas, gsSet.Spec.Replicas)

				return true, gsSet, nil
			})

			replicas, err := c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{inactive})
			require.NoError(t, err, "no error")

			assert.Equal(t, v.expected.replicas, replicas)
			assert.Equal(t, v.expected.updated, updated)
			if updated {
				agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
			} else {
				agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
			}
		})
	}

}

func TestControllerRollingUpdateDeployment(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})

	type expected struct {
		inactiveSpecReplicas int32
		replicas             int32
		updated              bool
		err                  string
	}

	fixtures := map[string]struct {
		features                         string
		fleetSpecReplicas                int32
		activeSpecReplicas               int32
		activeStatusReplicas             int32
		readyReplicas                    int32
		inactiveSpecReplicas             int32
		inactiveStatusReplicas           int32
		inactiveStatusReadyReplicas      int32
		inactiveStatusAllocationReplicas int32
		nilMaxSurge                      bool
		nilMaxUnavailable                bool
		expected                         expected
	}{
		"nil MaxUnavailable, err expected": {
			fleetSpecReplicas:           100,
			activeSpecReplicas:          0,
			activeStatusReplicas:        0,
			inactiveSpecReplicas:        100,
			inactiveStatusReplicas:      100,
			inactiveStatusReadyReplicas: 100,
			nilMaxUnavailable:           true,
			expected: expected{
				err: "error parsing MaxUnavailable value: fleet-1: nil value for IntOrString",
			},
		},
		"nil MaxSurge, err expected": {
			fleetSpecReplicas:           100,
			activeSpecReplicas:          0,
			activeStatusReplicas:        0,
			inactiveSpecReplicas:        100,
			inactiveStatusReplicas:      100,
			inactiveStatusReadyReplicas: 100,
			nilMaxSurge:                 true,
			expected: expected{
				err: "error parsing MaxSurge value: fleet-1: nil value for IntOrString",
			},
		},
		"full inactive, empty inactive": {
			fleetSpecReplicas:           100,
			activeSpecReplicas:          0,
			activeStatusReplicas:        0,
			inactiveSpecReplicas:        100,
			inactiveStatusReplicas:      100,
			inactiveStatusReadyReplicas: 100,
			expected: expected{
				inactiveSpecReplicas: 70,
				replicas:             25,
				updated:              true,
			},
		},
		"almost empty inactive with allocated, almost full active": {
			fleetSpecReplicas:                100,
			activeSpecReplicas:               75,
			activeStatusReplicas:             75,
			inactiveSpecReplicas:             10,
			inactiveStatusReplicas:           10,
			inactiveStatusReadyReplicas:      10,
			inactiveStatusAllocationReplicas: 5,

			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             95,
				updated:              true,
			},
		},
		"attempt to drive replicas over the max surge": {
			features:                    "RollingUpdateFix=false",
			fleetSpecReplicas:           100,
			activeSpecReplicas:          25,
			activeStatusReplicas:        25,
			inactiveSpecReplicas:        95,
			inactiveStatusReplicas:      95,
			inactiveStatusReadyReplicas: 95,
			expected: expected{
				inactiveSpecReplicas: 45,
				replicas:             30,
				updated:              true,
			},
		},
		"test smalled numbers of active and allocated": {
			fleetSpecReplicas:                5,
			activeSpecReplicas:               0,
			activeStatusReplicas:             0,
			inactiveSpecReplicas:             5,
			inactiveStatusReplicas:           5,
			inactiveStatusReadyReplicas:      5,
			inactiveStatusAllocationReplicas: 2,
			expected: expected{
				inactiveSpecReplicas: 4,
				replicas:             2,
				updated:              true,
			},
		},
		"activeSpecReplicas >= (fleetSpecReplicas - inactiveStatusAllocationReplicas)": {
			fleetSpecReplicas:                75,
			activeSpecReplicas:               75,
			activeStatusReplicas:             75,
			inactiveSpecReplicas:             10,
			inactiveStatusReplicas:           10,
			inactiveStatusReadyReplicas:      10,
			inactiveStatusAllocationReplicas: 5,
			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             70,
				updated:              true,
			},
		},
		"rolling update does not remove all ready replicas": {
			features:                         "RollingUpdateFix=true",
			fleetSpecReplicas:                100,
			activeSpecReplicas:               0,
			activeStatusReplicas:             0,
			inactiveSpecReplicas:             100,
			inactiveStatusReplicas:           100,
			inactiveStatusReadyReplicas:      10,
			inactiveStatusAllocationReplicas: 90,
			expected: expected{
				inactiveSpecReplicas: 97,
				replicas:             10,
				updated:              true,
			},
		},
		"rolling update stops scaling fully allocated inactive": {
			features:                         "RollingUpdateFix=true",
			fleetSpecReplicas:                100,
			activeSpecReplicas:               50,
			activeStatusReplicas:             50,
			inactiveSpecReplicas:             50,
			inactiveStatusReplicas:           50,
			inactiveStatusReadyReplicas:      0,
			inactiveStatusAllocationReplicas: 50,
			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             50,
				updated:              true,
			},
		},
		"rolling update scales down with fleet spec replicas = 0": {
			features:                         "RollingUpdateFix=true",
			fleetSpecReplicas:                0,
			activeSpecReplicas:               0,
			activeStatusReplicas:             0,
			inactiveSpecReplicas:             3,
			inactiveStatusReplicas:           3,
			inactiveStatusReadyReplicas:      3,
			inactiveStatusAllocationReplicas: 0,
			expected: expected{
				inactiveSpecReplicas: 0,
				replicas:             0,
				updated:              true,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			utilruntime.FeatureTestMutex.Lock()
			defer utilruntime.FeatureTestMutex.Unlock()
			require.NoError(t, utilruntime.ParseFeatures(v.features))

			f := defaultFixture()

			mu := intstr.FromString("30%")
			f.Spec.Strategy.RollingUpdate.MaxUnavailable = &mu
			f.Spec.Replicas = v.fleetSpecReplicas

			// Inactive GameServerSet is downscaled second time only after
			// ReadyReplicas has raised.
			f.Status.ReadyReplicas = v.activeStatusReplicas + v.inactiveStatusReadyReplicas

			if v.nilMaxSurge {
				f.Spec.Strategy.RollingUpdate.MaxSurge = nil
			} else {
				assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
			}

			if v.nilMaxUnavailable {
				f.Spec.Strategy.RollingUpdate.MaxUnavailable = nil
			} else {
				assert.Equal(t, "30%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
			}

			active := f.GameServerSet()
			active.ObjectMeta.Name = "active"
			active.Spec.Replicas = v.activeSpecReplicas
			active.Status.Replicas = v.activeStatusReplicas
			active.Status.ReadyReplicas = v.activeStatusReplicas

			inactive := f.GameServerSet()
			inactive.ObjectMeta.Name = "inactive"
			inactive.Spec.Replicas = v.inactiveSpecReplicas
			inactive.Status.Replicas = v.inactiveStatusReplicas
			inactive.Status.ReadyReplicas = v.inactiveStatusReadyReplicas
			inactive.Status.AllocatedReplicas = v.inactiveStatusAllocationReplicas

			logrus.WithField("inactive", inactive).Info("Setting up the initial inactive")

			updated := false
			c, m := newFakeController()

			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsSet := ua.GetObject().(*agonesv1.GameServerSet)
				assert.Equal(t, inactive.ObjectMeta.Name, gsSet.ObjectMeta.Name)
				assert.Equal(t, v.expected.inactiveSpecReplicas, gsSet.Spec.Replicas)

				return true, gsSet, nil
			})

			replicas, err := c.rollingUpdateDeployment(context.Background(), f, active, []*agonesv1.GameServerSet{inactive})

			if v.expected.err != "" {
				assert.EqualError(t, err, v.expected.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, v.expected.replicas, replicas)
				assert.Equal(t, v.expected.updated, updated)
				if updated {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingGameServerSet")
				} else {
					agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
				}
			}
		})
	}
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewController(healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

// newFakeExtensions returns a fake extensions struct
func newFakeExtensions() *Extensions {
	return NewExtensions(generic.New(), webhooks.NewWebHook(http.NewServeMux()))
}

func defaultFixture() *agonesv1.Fleet {
	f := &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: agonesv1.FleetSpec{
			Replicas:   5,
			Scheduling: apis.Packed,
			Template:   agonesv1.GameServerTemplateSpec{},
		},
	}
	f.ApplyDefaults()
	return f
}

func defaultGSSpec() *agonesv1.GameServerTemplateSpec {
	return &agonesv1.GameServerTemplateSpec{
		Spec: agonesv1.GameServerSpec{
			Container: "udp-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "udp-server",
						Image:           "gcr.io/images/new:0.2",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					}},
				},
			},
		},
	}
}

func getAdmissionReview(raw []byte) admissionv1.AdmissionReview {
	gvk := metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("Fleet"))

	return admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Kind:      gvk,
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
		},
		Response: &admissionv1.AdmissionResponse{Allowed: true},
	}
}

// MOCKS SECTION

type fakeGSSListerWithErr struct {
}

// GameServerSetLister interface implementation
func (fgsl *fakeGSSListerWithErr) List(_ labels.Selector) (ret []*v1.GameServerSet, err error) {
	return nil, errors.New("random-err")
}

// GameServerSetLister interface implementation
func (fgsl *fakeGSSListerWithErr) Get(_ string) (ret *v1.GameServerSet, err error) {
	return nil, errors.New("random-err")
}

func (fgsl *fakeGSSListerWithErr) GameServerSets(_ string) agonesv1client.GameServerSetNamespaceLister {
	return fgsl
}

type fakeFleetsGetterWithErr struct{}

// FleetsGetter interface implementation
func (ffg *fakeFleetsGetterWithErr) Fleets(_ string) agonesv1clientset.FleetInterface {
	return &fakeFleetsGetterWithErr{}
}

func (ffg *fakeFleetsGetterWithErr) Create(_ context.Context, _ *v1.Fleet, _ metav1.CreateOptions) (*v1.Fleet, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Update(_ context.Context, _ *v1.Fleet, _ metav1.UpdateOptions) (*v1.Fleet, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) UpdateStatus(_ context.Context, _ *v1.Fleet, _ metav1.UpdateOptions) (*v1.Fleet, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Delete(_ context.Context, _ string, _ metav1.DeleteOptions) error {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Get(_ context.Context, _ string, _ metav1.GetOptions) (*v1.Fleet, error) {
	return nil, errors.New("err-from-fleet-getter")
}

func (ffg *fakeFleetsGetterWithErr) List(_ context.Context, _ metav1.ListOptions) (*v1.FleetList, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (result *v1.Fleet, err error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) Apply(_ context.Context, _ *applyconfigurations.FleetApplyConfiguration, _ metav1.ApplyOptions) (*v1.Fleet, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) ApplyStatus(_ context.Context, _ *applyconfigurations.FleetApplyConfiguration, _ metav1.ApplyOptions) (*v1.Fleet, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) GetScale(_ context.Context, _ string, _ metav1.GetOptions) (*autoscalingv1.Scale, error) {
	panic("not implemented")
}

func (ffg *fakeFleetsGetterWithErr) UpdateScale(_ context.Context, _ string, _ *autoscalingv1.Scale, _ metav1.UpdateOptions) (*autoscalingv1.Scale, error) {
	panic("not implemented")
}

type fakeFleetListerWithErr struct{}

// FleetLister interface implementation
func (ffl *fakeFleetListerWithErr) List(_ labels.Selector) (ret []*v1.Fleet, err error) {
	return nil, errors.New("err-from-fleet-lister")
}

func (ffl *fakeFleetListerWithErr) Fleets(_ string) agonesv1client.FleetNamespaceLister {
	return &fakeFleetNamespaceListerWithErr{}
}

type fakeFleetNamespaceListerWithErr struct{}

// FleetNamespaceLister interface implementation
func (ffnl *fakeFleetNamespaceListerWithErr) List(_ labels.Selector) (ret []*v1.Fleet, err error) {
	return nil, errors.New("err-from-namespace-lister")
}

func (ffnl *fakeFleetNamespaceListerWithErr) Get(_ string) (*v1.Fleet, error) {
	return nil, errors.New("err-from-namespace-lister")
}
