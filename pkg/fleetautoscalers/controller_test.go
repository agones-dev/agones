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

package fleetautoscalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	admregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var (
	gvk = metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("FleetAutoscaler"))
)

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	type expected struct {
		responseAllowed bool
		patches         []jsonpatch.JsonPatchOperation
		nilPatch        bool
	}

	var testCases = []struct {
		description string
		fixture     interface{}
		expected    expected
	}{
		{
			description: "OK",
			fixture: &autoscalingv1.FleetAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "fas-1",
					Namespace:  "default",
					Generation: 2,
				},
				Spec: autoscalingv1.FleetAutoscalerSpec{
					FleetName: "fleet-1",
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type: autoscalingv1.BufferPolicyType,
						Buffer: &autoscalingv1.BufferPolicy{
							BufferSize:  intstr.FromInt(5),
							MaxReplicas: 100,
						},
					},
					Sync: &autoscalingv1.FleetAutoscalerSync{
						Type: autoscalingv1.FixedIntervalSyncType,
						FixedInterval: autoscalingv1.FixedIntervalSync{
							Seconds: 30,
						},
					},
				},
			},
			expected: expected{
				responseAllowed: true,
				patches:         []jsonpatch.JsonPatchOperation{},
			},
		},
		{
			description: "OK",
			// Spec.Sync is not defined
			fixture: &autoscalingv1.FleetAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "fas-1",
					Namespace:  "default",
					Generation: 2,
				},
				Spec: autoscalingv1.FleetAutoscalerSpec{
					FleetName: "fleet-1",
					Policy: autoscalingv1.FleetAutoscalerPolicy{
						Type: autoscalingv1.BufferPolicyType,
						Buffer: &autoscalingv1.BufferPolicy{
							BufferSize:  intstr.FromInt(5),
							MaxReplicas: 100,
						},
					},
				},
			},
			expected: expected{
				responseAllowed: true,
				patches: []jsonpatch.JsonPatchOperation{
					{
						Operation: "add",
						Path:      "/spec/sync",
						Value: map[string]interface{}{
							"fixedInterval": map[string]interface{}{
								"seconds": float64(30),
							},
							"type": "FixedInterval",
						},
					},
				},
			},
		},
		{
			description: "Wrong request object, err expected",
			fixture:     "WRONG DATA",
			expected:    expected{nilPatch: true},
		},
	}

	ext := newFakeExtensions()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			raw, err := json.Marshal(tc.fixture)
			require.NoError(t, err)

			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Kind:      gvk,
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: raw,
					},
				},
				Response: &admissionv1.AdmissionResponse{Allowed: true},
			}

			result, err := ext.mutationHandler(review)

			assert.NoError(t, err)
			if tc.expected.nilPatch {
				require.Nil(t, result.Response.PatchType)
			} else {
				assert.True(t, result.Response.Allowed)
				assert.Equal(t, admissionv1.PatchTypeJSONPatch, *result.Response.PatchType)

				patch := &jsonpatch.ByPath{}
				err = json.Unmarshal(result.Response.Patch, patch)
				require.NoError(t, err)

				found := false

				for _, expected := range tc.expected.patches {
					for _, p := range *patch {
						if assert.ObjectsAreEqual(p, expected) {
							found = true
						}
					}
					assert.True(t, found, "Could not find operation %#v in patch %v", expected, *patch)
				}
			}
		})
	}
}

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	t.Run("valid fleet autoscaler", func(t *testing.T) {
		_, m := newFakeController()
		ext := newFakeExtensions()
		fas, _ := defaultFixtures()
		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := ext.validationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	})

	t.Run("invalid fleet autoscaler", func(t *testing.T) {
		_, m := newFakeController()
		ext := newFakeExtensions()
		fas, _ := defaultFixtures()
		// this make it invalid
		fas.Spec.Policy.Buffer = nil

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := ext.validationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotEmpty(t, result.Response.Result.Details)
	})

	t.Run("unable to unmarshal AdmissionRequest", func(t *testing.T) {
		ext := newFakeExtensions()

		review, err := newInvalidAdmissionReview()
		assert.Nil(t, err)

		_, err = ext.validationHandler(review)

		if assert.NotNil(t, err) {
			assert.Equal(t, "error unmarshalling FleetAutoscaler json after schema validation: \"MQ==\": json: cannot unmarshal string into Go value of type v1.FleetAutoscaler", err.Error())
		}
	})
}

func TestWebhookControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	t.Run("valid fleet autoscaler", func(t *testing.T) {
		_, m := newFakeController()
		ext := newFakeExtensions()
		fas, _ := defaultWebhookFixtures()
		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := ext.validationHandler(review)
		assert.Nil(t, err)
		assert.True(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
	})

	t.Run("invalid fleet autoscaler", func(t *testing.T) {
		_, m := newFakeController()
		ext := newFakeExtensions()
		fas, _ := defaultWebhookFixtures()
		// this make it invalid
		fas.Spec.Policy.Webhook = nil

		_, cancel := agtesting.StartInformers(m)
		defer cancel()

		review, err := newAdmissionReview(*fas)
		assert.Nil(t, err)

		result, err := ext.validationHandler(review)
		assert.Nil(t, err)
		assert.False(t, result.Response.Allowed, fmt.Sprintf("%#v", result.Response))
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.NotEmpty(t, result.Response.Result.Details)
	})
}

// nolint:dupl
func TestControllerSyncFleetAutoscaler(t *testing.T) {

	t.Run("no scaling up because fleet is marked for deletion, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(7)

		f.Spec.Replicas = 5
		f.Status.Replicas = 5
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 0
		f.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("scaling up, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(7)

		f.Spec.Replicas = 5
		f.Status.Replicas = 5
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 0

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(5))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(12))
			assert.Equal(t, fas.Status.LastAppliedPolicy, autoscalingv1.BufferPolicyType)
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(12))
			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("scaling up, webhook policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultWebhookFixtures()
		f.Spec.Replicas = 50
		f.Status.Replicas = f.Spec.Replicas
		f.Status.AllocatedReplicas = 45
		f.Status.ReadyReplicas = 0

		ts := testServer{}
		server := httptest.NewServer(ts)
		defer server.Close()

		fas.Spec.Policy.Webhook.URL = &(server.URL)
		fas.Spec.Policy.Webhook.Service = nil

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(50))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(100))
			assert.Equal(t, fas.Status.LastAppliedPolicy, autoscalingv1.WebhookPolicyType)
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(100))
			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("no scaling up because fleet is marked for deletion, webhook policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultWebhookFixtures()
		f.Spec.Replicas = 50
		f.Status.Replicas = f.Spec.Replicas
		f.Status.AllocatedReplicas = 45
		f.Status.ReadyReplicas = 0
		f.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}

		ts := testServer{}
		server := httptest.NewServer(ts)
		defer server.Close()

		fas.Spec.Policy.Webhook.URL = &(server.URL)
		fas.Spec.Policy.Webhook.Service = nil

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("scaling down, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(8)

		f.Spec.Replicas = 20
		f.Status.Replicas = 20
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 15

		fUpdated := false
		fasUpdated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(20))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(13))
			assert.Equal(t, fas.Status.LastAppliedPolicy, autoscalingv1.BufferPolicyType)
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fUpdated = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, f.Spec.Replicas, int32(13))

			return true, f, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, fUpdated, "fleet should have been updated")
		assert.True(t, fasUpdated, "fleetautoscaler should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleet")
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("no scaling down because fleet is marked for deletion, buffer policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(8)

		f.Spec.Replicas = 20
		f.Status.Replicas = 20
		f.Status.AllocatedReplicas = 5
		f.Status.ReadyReplicas = 15
		f.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return true, nil, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")

			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("no scaling no update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		f.Spec.Replicas = 10
		f.Status.Replicas = 10
		f.Status.ReadyReplicas = 5
		fas.Spec.Policy.Buffer.BufferSize = intstr.FromInt(5)
		fas.Status.CurrentReplicas = 10
		fas.Status.DesiredReplicas = 10

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, fas.ObjectMeta.Name)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("fleet not available", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		fas.Status.CurrentReplicas = 5
		updated := false

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(0))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(0))
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		assert.Nil(t, err)
		assert.True(t, updated)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "FailedGetFleet")
	})

	t.Run("fleet not available, error on status update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		fas.Status.CurrentReplicas = 5

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "FailedGetFleet")
	})

	t.Run("wrong policy", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		// wrong policy, should fail
		fas.Spec.Policy = autoscalingv1.FleetAutoscalerPolicy{
			Type: "WRONG TYPE",
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error calculating autoscaling fleet: fleet-1: wrong policy type, should be one of: Buffer, Webhook, Counter, List, Schedule, Chain", err.Error())
		}
	})

	t.Run("wrong policy, error on status update", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()
		fas.Status.DesiredReplicas = 10
		// wrong policy, should fail
		fas.Spec.Policy = autoscalingv1.FleetAutoscalerPolicy{
			Type: "WRONG TYPE",
		}

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
	})

	t.Run("error on scale fleet step", func(t *testing.T) {
		t.Parallel()
		c, m := newFakeController()
		fas, f := defaultFixtures()

		m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
		})

		m.AgonesClient.AddReactor("list", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.FleetList{Items: []agonesv1.Fleet{*f}}, nil
		})

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			return true, ca.GetObject().(*agonesv1.Fleet), errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()
		fleetAutoscalerThreadEventually(t, c, fas)

		err := c.syncFleetAutoscaler(ctx, "default/fas-1")
		if assert.NotNil(t, err) {
			assert.Equal(t, "error autoscaling fleet fleet-1 to 7 replicas: error updating replicas for fleet fleet-1: random-err", err.Error())
		}

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleetError")
	})

	t.Run("Missing fleet autoscaler, doesn't fail/panic", func(t *testing.T) {
		t.Parallel()

		c, m := newFakeController()
		m.AgonesClient.AddReactor("get", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ga := action.(k8stesting.GetAction)
			return true, nil, k8serrors.NewNotFound(corev1.Resource("gameserver"), ga.GetName())
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetSynced, c.fleetAutoscalerSynced)
		defer cancel()

		require.NoError(t, c.syncFleetAutoscaler(ctx, "default/fas-1"))
	})
}

// fleetAutoscalerThreadEventually waits to see if a thread is started for the given fleet autoscaler
// this is a sign that the informer has been started and the fleet autoscaler has been processed
// by the informer.
func fleetAutoscalerThreadEventually(t *testing.T, c *Controller, fas *autoscalingv1.FleetAutoscaler) {
	require.Eventually(t, func() bool {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		_, ok := c.fasThreads[fas.ObjectMeta.UID]
		return ok
	}, 5*time.Second, 100*time.Millisecond, "fleet autoscaler controller didn't start the sync thread")
}

func TestControllerScaleFleet(t *testing.T) {
	t.Parallel()

	t.Run("fleet that must be scaled", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas + 5

		update := false

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			update = true
			ca := action.(k8stesting.UpdateAction)
			f := ca.GetObject().(*agonesv1.Fleet)
			assert.Equal(t, replicas, f.Spec.Replicas)

			return true, f, nil
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		assert.Nil(t, err)
		assert.True(t, update, "Fleet should be updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingFleet")
	})

	t.Run("error on updating fleet", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas + 5

		m.AgonesClient.AddReactor("update", "fleets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			return true, ca.GetObject().(*agonesv1.Fleet), errors.New("random-err")
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating replicas for fleet fleet-1: random-err", err.Error())
		}
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "AutoScalingFleetError")
	})

	t.Run("equal replicas, no update", func(t *testing.T) {
		c, m := newFakeController()
		fas, f := defaultFixtures()
		replicas := f.Spec.Replicas

		m.AgonesClient.AddReactor("update", "fleets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleet should not update")
			return false, nil, nil
		})

		err := c.scaleFleet(context.Background(), fas, f, replicas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestControllerUpdateStatus(t *testing.T) {
	t.Parallel()

	t.Run("must update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		fasUpdated := false

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, true)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(10))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(20))
			assert.Equal(t, fas.Status.LastAppliedPolicy, autoscalingv1.BufferPolicyType)
			assert.NotNil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, 10, 20, true, false, fas.Spec.Policy.Type)
		assert.Nil(t, err)
		assert.True(t, fasUpdated)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("must not update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		fas.Status.AbleToScale = true
		fas.Status.ScalingLimited = false
		fas.Status.CurrentReplicas = 10
		fas.Status.DesiredReplicas = 20
		fas.Status.LastScaleTime = nil
		fas.Status.LastAppliedPolicy = fas.Spec.Policy.Type

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, fas.Status.CurrentReplicas, fas.Status.DesiredReplicas, false, fas.Status.ScalingLimited, fas.Spec.Policy.Type)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with error", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatus(ctx, fas, fas.Status.CurrentReplicas, fas.Status.DesiredReplicas, false, fas.Status.ScalingLimited, fas.Spec.Policy.Type)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with a scaling limit", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		err := c.updateStatus(context.Background(), fas, 10, 20, true, true, fas.Spec.Policy.Type)
		assert.Nil(t, err)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "ScalingLimited")
	})

	t.Run("update with a scaling limit with minimum", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		err := c.updateStatus(context.Background(), fas, 1, 3, true, true, fas.Spec.Policy.Type)
		assert.Nil(t, err)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "limited to minimum size of 3")
	})

	t.Run("update with a scaling limit with maximum", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()

		err := c.updateStatus(context.Background(), fas, 12, 10, true, true, fas.Spec.Policy.Type)
		assert.Nil(t, err)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "limited to maximum size of 10")
	})
}

func TestControllerUpdateStatusUnableToScale(t *testing.T) {
	t.Parallel()

	t.Run("must update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10

		fasUpdated := false

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			fasUpdated = true
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			assert.Equal(t, fas.Status.AbleToScale, false)
			assert.Equal(t, fas.Status.ScalingLimited, false)
			assert.Equal(t, fas.Status.CurrentReplicas, int32(0))
			assert.Equal(t, fas.Status.DesiredReplicas, int32(0))
			assert.Equal(t, fas.Status.LastAppliedPolicy, autoscalingv1.FleetAutoscalerPolicyType(""))
			assert.Nil(t, fas.Status.LastScaleTime)
			return true, fas, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		assert.Nil(t, err)
		assert.True(t, fasUpdated)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("update with error", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.DesiredReplicas = 10

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.UpdateAction)
			fas := ca.GetObject().(*autoscalingv1.FleetAutoscaler)
			return true, fas, errors.New("random-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		if assert.NotNil(t, err) {
			assert.Equal(t, "error updating status for fleetautoscaler fas-1: random-err", err.Error())
		}
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})

	t.Run("must not update", func(t *testing.T) {
		c, m := newFakeController()
		fas, _ := defaultFixtures()
		fas.Status.AbleToScale = false
		fas.Status.ScalingLimited = false
		fas.Status.CurrentReplicas = 0
		fas.Status.DesiredReplicas = 0
		fas.Status.LastAppliedPolicy = autoscalingv1.FleetAutoscalerPolicyType("")

		m.AgonesClient.AddReactor("update", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "fleetautoscaler should not update")
			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
		defer cancel()

		err := c.updateStatusUnableToScale(ctx, fas)
		assert.Nil(t, err)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)
	})
}

func TestControllerEvents(t *testing.T) {
	t.Parallel()

	c, mocks := newFakeController()
	fakeWatch := watch.NewFake()
	mocks.AgonesClient.AddWatchReactor("fleetautoscalers", k8stesting.DefaultWatchReactor(fakeWatch, nil))
	_, cancel := agtesting.StartInformers(mocks, c.fleetAutoscalerSynced)
	defer cancel()

	// add fleet autoscaler
	fas, _ := defaultFixtures()
	fakeWatch.Add(fas.DeepCopy())

	require.Eventually(t, func() bool {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		return len(c.fasThreads) == 1
	}, 30*time.Second, time.Second, "should be added")

	c.fasThreadMutex.Lock()
	require.Equal(t, fas.ObjectMeta.Generation, c.fasThreads[fas.ObjectMeta.UID].generation)
	c.fasThreadMutex.Unlock()

	// modify the fleet autoscaler
	fas.ObjectMeta.Generation++
	fakeWatch.Modify(fas.DeepCopy())

	require.Eventually(t, func() bool {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		return fas.ObjectMeta.Generation == c.fasThreads[fas.ObjectMeta.UID].generation
	}, 30*time.Second, time.Second, "should be updated")

	// delete the fleet auto scaler
	fakeWatch.Delete(fas.DeepCopy())
	require.Eventually(t, func() bool {
		c.fasThreadMutex.Lock()
		defer c.fasThreadMutex.Unlock()
		return len(c.fasThreads) == 0
	}, 30*time.Second, time.Second, "should be deleted")
}

func TestControllerAddUpdateDeleteFasThread(t *testing.T) {
	t.Parallel()

	var counter int64
	c, m := newFakeController()
	c.workerqueue.SyncHandler = func(_ context.Context, _ string) error {
		atomic.AddInt64(&counter, 1)
		return nil
	}

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})

	ctx, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
	defer cancel()
	go func() {
		require.NoError(t, c.Run(ctx, 1))
	}()

	fas, _ := defaultFixtures()
	fas.Spec.Sync.FixedInterval.Seconds = 1

	c.addFasThread(fas, true)

	// unfortunately we can't mock the timer, so we'll confirm that two enqueue processes fire. One on method execution,
	// and then one based on the ticker.
	require.Eventuallyf(t, func() bool {
		return atomic.LoadInt64(&counter) >= 2
	}, 10*time.Second, time.Second, "Should have at least two counters")

	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 1)
	c.fasThreadMutex.Unlock()

	// update with the same values
	c.updateFasThread(ctx, fas)
	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 1)
	require.Equal(t, fas.ObjectMeta.Generation, c.fasThreads[fas.ObjectMeta.UID].generation)
	c.fasThreadMutex.Unlock()

	// update duration
	fas.Spec.Sync.FixedInterval.Seconds = 3
	fas.ObjectMeta.Generation++

	c.updateFasThread(ctx, fas)
	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 1)
	require.Equal(t, fas.ObjectMeta.Generation, c.fasThreads[fas.ObjectMeta.UID].generation)
	c.fasThreadMutex.Unlock()

	// update, but with a second fas, that doesn't exist in the system yet
	fas2 := fas.DeepCopy()
	fas2.Spec.Sync.FixedInterval.Seconds = 1
	fas2.ObjectMeta.Generation = 5
	fas2.ObjectMeta.UID = "4321"

	c.updateFasThread(ctx, fas2)
	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 2)
	require.Equal(t, fas.ObjectMeta.Generation, c.fasThreads[fas.ObjectMeta.UID].generation)
	require.Equal(t, fas2.ObjectMeta.Generation, c.fasThreads[fas2.ObjectMeta.UID].generation)
	c.fasThreadMutex.Unlock()

	// delete the current fas.
	c.deleteFasThread(ctx, fas, true)
	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 1)
	require.Equal(t, fas2.ObjectMeta.Generation, c.fasThreads[fas2.ObjectMeta.UID].generation)
	c.fasThreadMutex.Unlock()

	c.deleteFasThread(ctx, fas2, true)
	c.fasThreadMutex.Lock()
	require.Len(t, c.fasThreads, 0)
	c.fasThreadMutex.Unlock()

	// we shouldn't get any more updates, so wait for 3 checks in a row that have the
	// same counter amount to prove that there aren't any changes for a while.
	var check []int64
	require.Eventually(t, func() bool {
		check = append(check, atomic.LoadInt64(&counter))
		l := len(check)
		if l < 3 {
			return false
		}
		l--
		return check[l] == check[l-1] && check[l-1] == check[l-2]
	}, 30*time.Second, 2*time.Second, "changes keep happening", check)
}

func TestControllerCleanFasThreads(t *testing.T) {
	c, m := newFakeController()
	fas, _ := defaultFixtures()

	m.AgonesClient.AddReactor("list", "fleetautoscalers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.FleetAutoscalerList{Items: []autoscalingv1.FleetAutoscaler{*fas}}, nil
	})

	_, cancel := agtesting.StartInformers(m, c.fleetAutoscalerSynced)
	defer cancel()

	c.fasThreadMutex.Lock()
	c.fasThreads = map[types.UID]fasThread{
		"1":                {func() {}, &fasState{}, 1},
		"2":                {func() {}, &fasState{}, 2},
		fas.ObjectMeta.UID: {func() {}, &fasState{}, 3},
	}
	c.fasThreadMutex.Unlock()

	key, err := cache.MetaNamespaceKeyFunc(fas)
	require.NoError(t, err)
	require.NoError(t, c.cleanFasThreads(context.Background(), key))

	require.Len(t, c.fasThreads, 1)
	require.Equal(t, int64(3), c.fasThreads[fas.ObjectMeta.UID].generation)
}

func defaultFixtures() (*autoscalingv1.FleetAutoscaler, *agonesv1.Fleet) {
	f := &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: agonesv1.FleetSpec{
			Replicas:   8,
			Scheduling: apis.Packed,
			Template:   agonesv1.GameServerTemplateSpec{},
		},
		Status: agonesv1.FleetStatus{
			Replicas:          5,
			ReadyReplicas:     3,
			ReservedReplicas:  3,
			AllocatedReplicas: 2,
		},
	}

	fas := &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "fas-1",
			Namespace:  "default",
			Generation: 2,
			UID:        "4567",
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: f.ObjectMeta.Name,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					BufferSize:  intstr.FromInt(5),
					MaxReplicas: 100,
				},
			},
			Sync: &autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: autoscalingv1.FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
	}

	return fas, f
}

func defaultWebhookFixtures() (*autoscalingv1.FleetAutoscaler, *agonesv1.Fleet) {
	fas, f := defaultFixtures()
	fas.Spec.Policy.Type = autoscalingv1.WebhookPolicyType
	fas.Spec.Policy.Buffer = nil
	url := "/autoscaler"
	fas.Spec.Policy.Webhook = &autoscalingv1.URLConfiguration{
		Service: &admregv1.ServiceReference{
			Name: "fleetautoscaler-service",
			Path: &url,
		},
	}

	return fas, f
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	counter := gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	c := NewController(healthcheck.NewHandler(), m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory, counter)
	c.recorder = m.FakeRecorder
	return c, m
}

// newFakeExtensions returns a fake extensions struct
func newFakeExtensions() *Extensions {
	return NewExtensions(webhooks.NewWebHook(http.NewServeMux()))
}

func newAdmissionReview(fas autoscalingv1.FleetAutoscaler) (admissionv1.AdmissionReview, error) {
	raw, err := json.Marshal(fas)
	if err != nil {
		return admissionv1.AdmissionReview{}, err
	}
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Kind:      gvk,
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admissionv1.AdmissionResponse{Allowed: true},
	}
	return review, err
}

func newInvalidAdmissionReview() (admissionv1.AdmissionReview, error) {
	raw, err := json.Marshal([]byte(`1`))
	if err != nil {
		return admissionv1.AdmissionReview{}, err
	}
	review := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Kind:      gvk,
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: "default",
		},
		Response: &admissionv1.AdmissionResponse{Allowed: true},
	}
	return review, nil
}
