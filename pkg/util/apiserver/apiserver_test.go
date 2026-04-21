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

package apiserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kube-openapi/pkg/handler3"
)

var (
	gv       = metav1.GroupVersion{Group: "allocation.agones.dev", Version: "v1"}
	resource = metav1.APIResource{
		Name:         "gameserverallocations",
		SingularName: "gameserverallocation",
		Namespaced:   true,
		Kind:         "GameServerAllocation",
		Verbs: []string{
			"create",
		},
		ShortNames: []string{"gsa"},
	}
)

func TestApiServerStubs(t *testing.T) {
	t.Parallel()
	// testing API endpoints that don't do a whole lot, but are needed for APIService endpoints.
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	_ = NewAPIServer(mux)

	ts.Start()
	defer ts.Close()
	client := ts.Client()

	resp, err := client.Get(ts.URL + "/openapi/v2")
	require.NoError(t, err)
	assert.Equal(t, k8sruntime.ContentTypeJSON, resp.Header.Get(ContentTypeHeader))
	v2data := spec.Swagger{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v2data))
	require.NoError(t, resp.Body.Close())

	resp, err = client.Get(ts.URL + "/openapi/v3")
	require.NoError(t, err)
	assert.Equal(t, k8sruntime.ContentTypeJSON, resp.Header.Get(ContentTypeHeader))
	v3data := handler3.OpenAPIV3Discovery{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v3data))
	require.NoError(t, resp.Body.Close())

	resp, err = client.Get(ts.URL + "/apis")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func TestAPIServerAddAPIResourceCRDHandler(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	api := NewAPIServer(mux)
	handled := false

	api.AddAPIResource(gv.String(), resource, func(_ http.ResponseWriter, _ *http.Request, ns string) error {
		handled = true
		assert.Equal(t, "default", ns)
		return nil
	})

	ts.Start()
	defer ts.Close()

	client := ts.Client()
	path := ts.URL + "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations"

	resp, err := client.Get(path)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.True(t, handled, "not handled!")
	defer resp.Body.Close() // nolint: errcheck

	handled = false
	path = ts.URL + "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallZZZZions"
	resp, err = client.Get(path)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.False(t, handled, "not handled!")
	defer resp.Body.Close() // nolint: errcheck
}

func TestAPIServerAddAPIResourceDiscovery(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	api := NewAPIServer(mux)

	api.AddAPIResource(gv.String(), resource, func(_ http.ResponseWriter, _ *http.Request, _ string) error {
		return nil
	})

	ts.Start()
	defer ts.Close()

	client := ts.Client()
	path := ts.URL + "/apis/allocation.agones.dev/v1"

	t.Run("No Accept Header", func(t *testing.T) {
		resp, err := client.Get(path)
		if resp != nil {
			defer resp.Body.Close() // nolint: errcheck
		}
		if !assert.NoError(t, err) {
			assert.FailNow(t, "should not error")
		}
		assert.Equal(t, k8sruntime.ContentTypeJSON, resp.Header.Get("Content-Type"))

		// default is json
		list := &metav1.APIResourceList{}
		err = json.NewDecoder(resp.Body).Decode(list)
		assert.NoError(t, err)

		assert.Equal(t, "v1", list.TypeMeta.APIVersion)
		assert.Equal(t, "APIResourceList", list.TypeMeta.Kind)
		assert.Equal(t, gv.String(), list.GroupVersion)
		assert.Equal(t, resource, list.APIResources[0])
	})

	t.Run("Accept */*", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, path, nil)
		assert.NoError(t, err)

		request.Header.Set("Accept", "*/*")
		resp, err := client.Do(request)
		assert.NoError(t, err)
		if resp != nil {
			defer resp.Body.Close() // nolint: errcheck
		}
		assert.Equal(t, k8sruntime.ContentTypeJSON, resp.Header.Get("Content-Type"))

		list := &metav1.APIResourceList{}
		err = json.NewDecoder(resp.Body).Decode(list)
		assert.NoError(t, err)

		assert.Equal(t, "v1", list.TypeMeta.APIVersion)
		assert.Equal(t, "APIResourceList", list.TypeMeta.Kind)
		assert.Equal(t, gv.String(), list.GroupVersion)
		assert.Equal(t, resource, list.APIResources[0])
	})

	t.Run("Accept Protobuf, */*", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, path, nil)
		assert.NoError(t, err)

		request.Header.Set("Accept", "application/vnd.kubernetes.protobuf, */*")
		resp, err := client.Do(request)
		assert.NoError(t, err)
		if resp != nil {
			defer resp.Body.Close() // nolint: errcheck
		}
		assert.Equal(t, "application/vnd.kubernetes.protobuf", resp.Header.Get("Content-Type"))

		list := &metav1.APIResourceList{}
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		info, ok := k8sruntime.SerializerInfoForMediaType(Codecs.SupportedMediaTypes(), "application/vnd.kubernetes.protobuf")
		assert.True(t, ok)

		gvk := unversionedVersion.WithKind("APIResourceList")
		_, _, err = info.Serializer.Decode(b, &gvk, list)
		assert.NoError(t, err)

		assert.Equal(t, "v1", list.TypeMeta.APIVersion)
		assert.Equal(t, "APIResourceList", list.TypeMeta.Kind)
		assert.Equal(t, gv.String(), list.GroupVersion)
		assert.Equal(t, resource, list.APIResources[0])
	})
}

func TestAPIServerAddAPIResourceDiscoveryParallel(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	ts := httptest.NewUnstartedServer(mux)
	api := NewAPIServer(mux)

	api.AddAPIResource(gv.String(), resource, func(_ http.ResponseWriter, _ *http.Request, _ string) error {
		return nil
	})

	ts.Start()
	defer ts.Close()

	t.Run("Parallel Tests", func(t *testing.T) {
		// Run 10 concurrent requests to exercise multithreading
		for i := 0; i < 10; i++ {
			t.Run("Accept */*", func(t *testing.T) {
				t.Parallel()
				client := ts.Client()
				path := ts.URL + "/apis/allocation.agones.dev/v1"
				request, err := http.NewRequest(http.MethodGet, path, nil)
				assert.NoError(t, err)

				request.Header.Set("Accept", "*/*")
				resp, err := client.Do(request)
				assert.NoError(t, err)
				if resp != nil {
					defer resp.Body.Close() // nolint: errcheck
				}
				assert.Equal(t, k8sruntime.ContentTypeJSON, resp.Header.Get("Content-Type"))

				list := &metav1.APIResourceList{}
				err = json.NewDecoder(resp.Body).Decode(list)
				assert.NoError(t, err)

				assert.Equal(t, "v1", list.TypeMeta.APIVersion)
				assert.Equal(t, "APIResourceList", list.TypeMeta.Kind)
				assert.Equal(t, gv.String(), list.GroupVersion)
				assert.Equal(t, resource, list.APIResources[0])
			})
		}
	})
}

func TestSplitNameSpaceResource(t *testing.T) {
	type expected struct {
		namespace string
		resource  string
		isError   bool
	}

	fixtures := []struct {
		path     string
		expected expected
	}{
		{
			path: "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations",
			expected: expected{
				namespace: "default",
				resource:  "gameserverallocations",
			},
		},
		{
			path: "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations/",
			expected: expected{
				namespace: "default",
				resource:  "gameserverallocations",
			},
		},
		{
			path: "/apis/allocation.agones.dev/v1/",
			expected: expected{
				isError: true,
			},
		},
		{
			path: "/apis/allocation.agones.dev/v1/blarg/default/gameserverallocations/",
			expected: expected{
				isError: true,
			},
		},
	}

	for _, test := range fixtures {
		t.Run(test.path, func(t *testing.T) {
			n, r, err := splitNameSpaceResource(test.path)
			if test.expected.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, test.expected.namespace, n)
			assert.Equal(t, test.expected.resource, r)
		})
	}
}
