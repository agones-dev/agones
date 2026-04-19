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

package webhooks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

func TestWebHookAddHandler(t *testing.T) {
	t.Parallel()

	type testHandler struct {
		gk schema.GroupKind
		op admissionv1.Operation
	}
	type expected struct {
		count int
	}
	fixtures := map[string]struct {
		handlers []testHandler
		expected expected
	}{
		"single, matching": {
			handlers: []testHandler{{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create}},
			expected: expected{count: 1},
		},
		"double, one matching op": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Update},
			},
		},
		"double, one matching group": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create},
				{gk: schema.GroupKind{Group: "nope", Kind: "kind"}, op: admissionv1.Create},
			},
		},
		"double, one matching kind": {
			expected: expected{count: 1},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "nope"}, op: admissionv1.Create},
			},
		},
		"double, both matching": {
			expected: expected{count: 2},
			handlers: []testHandler{
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create},
				{gk: schema.GroupKind{Group: "group", Kind: "kind"}, op: admissionv1.Create},
			},
		},
	}

	for k, handles := range fixtures {
		t.Run(k, func(t *testing.T) {

			stop := make(chan struct{})
			defer close(stop)

			fixture := admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{
				Kind:      metav1.GroupVersionKind{Kind: "kind", Group: "group", Version: "version"},
				Operation: admissionv1.Create,
				UID:       "1234"}}

			callCount := 0
			mux := http.NewServeMux()
			ts := httptest.NewUnstartedServer(mux)
			wh := NewWebHook(mux)

			for _, th := range handles.handlers {
				wh.AddHandler("/test", th.gk, th.op, func(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
					assert.Equal(t, fixture.Request, review.Request)
					assert.True(t, review.Response.Allowed)
					callCount++
					return review, nil
				})
			}

			ts.StartTLS()
			defer ts.Close()

			client := ts.Client()
			url := ts.URL + "/test"

			buf := &bytes.Buffer{}
			err := json.NewEncoder(buf).Encode(fixture)
			assert.Nil(t, err)

			r, err := http.NewRequest("GET", url, buf)
			assert.Nil(t, err)

			resp, err := client.Do(r)
			assert.Nil(t, err)
			defer resp.Body.Close() // nolint: errcheck
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, handles.expected.count, callCount, "[%v] /test should have been called for %#v", k, handles)
		})
	}
}

func TestWebHookFleetValidationHandler(t *testing.T) {
	t.Parallel()

	type testHandler struct {
		gk schema.GroupKind
		op admissionv1.Operation
	}
	type expected struct {
		count int
	}
	fixtures := map[string]struct {
		handlers []testHandler
		expected expected
	}{
		"single, matching": {
			handlers: []testHandler{{gk: schema.GroupKind{Group: "group", Kind: "fleet"}, op: admissionv1.Create}},
			expected: expected{count: 1},
		},
	}

	for k, handles := range fixtures {
		t.Run(k, func(t *testing.T) {

			stop := make(chan struct{})
			defer close(stop)

			raw := []byte(`{
				"apiVersion": "agones.dev/v1",
				"kind": "Fleet",
				"spec": {
					"replicas": 2,
					"template": {
						"spec": {
							"template": {
								"spec": {
									"containers": [{
										"image": "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.42",
										"name": false
									}]
								}
							}
						}
					}
				}
			}`)
			fixture := admissionv1.AdmissionReview{Request: &admissionv1.AdmissionRequest{
				Kind:      metav1.GroupVersionKind{Kind: "fleet", Group: "group", Version: "version"},
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: raw,
				},
				UID: "1234"}}

			callCount := 0
			mux := http.NewServeMux()
			ts := httptest.NewUnstartedServer(mux)
			wh := NewWebHook(mux)

			for _, th := range handles.handlers {
				wh.AddHandler("/test", th.gk, th.op, func(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
					fleet := &agonesv1.Fleet{}

					callCount++
					obj := review.Request.Object
					err := json.Unmarshal(obj.Raw, fleet)
					assert.NotNil(t, err)
					if err != nil {
						return review, errors.Wrapf(err, "error unmarshalling original Fleet json: %s", obj.Raw)
					}
					return review, nil
				})
			}

			ts.StartTLS()
			defer ts.Close()

			client := ts.Client()
			url := ts.URL + "/test"

			buf := &bytes.Buffer{}
			err := json.NewEncoder(buf).Encode(fixture)
			assert.Nil(t, err)

			r, err := http.NewRequest("GET", url, buf)
			assert.Nil(t, err)

			resp, err := client.Do(r)
			assert.Nil(t, err)
			defer resp.Body.Close() // nolint: errcheck
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			assert.Nil(t, err)

			expected := "cannot unmarshal bool into Go struct field Container.spec.template.spec.template.spec.containers.name of type string"
			assert.Contains(t, string(body), expected)

			assert.Equal(t, handles.expected.count, callCount, "[%v] /test should have been called for %#v", k, handles)
		})
	}
}
