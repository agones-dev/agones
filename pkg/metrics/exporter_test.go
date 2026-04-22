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

package metrics

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/test/e2e/framework"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestRegisterPrometheusExporter(t *testing.T) {
	resetMetrics()
	registry := prometheus.NewRegistry()

	handler, err := RegisterPrometheusExporter(registry)
	assert.NoError(t, err, "RegisterPrometheusExporter should not return an error")
	assert.NotNil(t, handler, "Handler should not be nil")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	require.NoError(t, err, "Creating request to /metrics should not fail")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	resp := rr.Result()
	defer func() {
		assert.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	body := string(bodyBytes)

	assert.Contains(t, body, "go_gc_duration_seconds", "Should contain default Go metrics")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain", "Expected text/plain content type")
}

func TestMetrics_Endpoint_ExposesAllMetrics(t *testing.T) {
	resetMetrics()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	runtime.EnableAllFeatures()

	conf := Config{
		PrometheusMetrics: true,
	}
	server := &httpserver.Server{
		Port:   "3001",
		Logger: framework.TestLogger(t),
	}

	m := newMockWithReactorNodesAndGameServers()
	ctrl := newFakeControllerWithMock(m)
	defer ctrl.close()

	ctrl.run(t)
	require.True(t, ctrl.sync(), "Controller failed to sync")

	// ---- Setup steps ----
	setupSteps := []func(t *testing.T, c *fakeController){
		setupGameServer,
		setupFleet,
		setupFleetAutoScalers,
		setupFleetWithCountersAndLists,
		setupGameServerPlayerConnect,
		setupGameServerStateDuration,
	}

	for _, stepFn := range setupSteps {
		stepFn(t, ctrl)
	}

	ctrl.collect()

	health, closer := SetupMetrics(conf, server)
	defer t.Cleanup(closer)

	assert.NotNil(t, health, "Health check handler should not be nil")
	server.Handle("/", health)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the HTTP server
	go func() {
		_ = server.Run(ctx, 0)
	}()
	time.Sleep(300 * time.Millisecond)

	resp, err := http.Get("http://localhost:3001/metrics")
	require.NoError(t, err, "Failed to GET metrics endpoint")
	defer func() {
		assert.NoError(t, resp.Body.Close())
	}()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200")

	metricsSet := collectMetricNames(resp)
	expectedMetrics := getMetricNames()

	for _, metric := range expectedMetrics {
		assert.Contains(t, metricsSet, metric, "Missing expected metric: %s", metric)
	}

}

func TestSetupMetrics_StackdriverOnly_NoPanic(t *testing.T) {
	// Set required env vars
	require.NoError(t, os.Setenv("POD_NAMESPACE", "default"))
	require.NoError(t, os.Setenv("POD_NAME", "test-pod"))
	require.NoError(t, os.Setenv("CONTAINER_NAME", "test-container"))

	// Fake metadata server
	handler := http.NewServeMux()
	handler.HandleFunc("/computeMetadata/v1/instance/zone", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		_, _ = w.Write([]byte("projects/123456789/zones/fake-zone"))
	})
	handler.HandleFunc("/computeMetadata/v1/instance/attributes/cluster-name", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Metadata-Flavor", "Google")
		_, _ = w.Write([]byte("fake-cluster"))
	})
	fakeMetadataServer := httptest.NewServer(handler)
	t.Cleanup(fakeMetadataServer.Close)

	// Set env var to point to the fake metadata server
	host := strings.TrimPrefix(fakeMetadataServer.URL, "http://")
	require.NoError(t, os.Setenv("GCE_METADATA_HOST", host))

	// Config for Stackdriver metrics
	conf := Config{
		Stackdriver:       true,
		GCPProjectID:      "fake-project",
		StackdriverLabels: "env=dev",
	}
	server := &httpserver.Server{
		Port:   "3001",
		Logger: framework.TestLogger(t),
	}

	health, closer := SetupMetrics(conf, server)
	defer t.Cleanup(closer)
	assert.NotNil(t, health, "Health check handler should not be nil")
}

func newMockWithReactorNodesAndGameServers() agtesting.Mocks {
	m := agtesting.NewMocks()

	m.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		n1 := nodeWithName("node1")
		n2 := nodeWithName("node2")
		n3 := nodeWithName("node3")
		return true, &corev1.NodeList{Items: []corev1.Node{*n1, *n2, *n3}}, nil
	})

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		gs1 := gameServerWithNode("node1")
		gs2 := gameServerWithNode("node2")
		gs3 := gameServerWithNode("node2")
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs1, *gs2, *gs3}}, nil
	})

	return m
}

func setupGameServer(t *testing.T, ctrl *fakeController) {
	gs := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateCreating)
	ctrl.gsWatch.Add(gs)

	require.Eventually(t, func() bool {
		gs, err := ctrl.gameServerLister.GameServers(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name)
		if gs == nil || err != nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.State == agonesv1.GameServerStateCreating
	}, 5*time.Second, time.Second)
	ctrl.collect()
}

func setupFleet(_ *testing.T, ctrl *fakeController) {
	flt := fleet("fleet-test", 8, 2, 5, 1, 1)
	ctrl.fleetWatch.Add(flt)

	flt = flt.DeepCopy()
	flt.Status.Replicas = 15
	ctrl.fleetWatch.Modify(flt)
	ctrl.collect()
}

func setupFleetAutoScalers(_ *testing.T, ctrl *fakeController) {
	ctrl.fasWatch.Add(fleetAutoScaler("fleet-test", "fas-test"))
	ctrl.collect()
}

func setupFleetWithCountersAndLists(_ *testing.T, ctrl *fakeController) {
	flt := fleet("cl-fleet-test", 8, 3, 5, 8, 0)
	ctrl.fleetWatch.Add(flt)
	flt = flt.DeepCopy()
	flt.Status.Counters = map[string]agonesv1.AggregatedCounterStatus{
		"players": {
			AllocatedCount:    24,
			AllocatedCapacity: 30,
			Count:             28,
			Capacity:          50,
		},
	}
	flt.Status.Lists = map[string]agonesv1.AggregatedListStatus{
		"rooms": {
			AllocatedCount:    4,
			AllocatedCapacity: 6,
			Count:             1,
			Capacity:          100,
		},
	}
	ctrl.fleetWatch.Modify(flt)
	ctrl.collect()
}

func setupGameServerPlayerConnect(t *testing.T, ctrl *fakeController) {
	gs := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateReady)
	gs.Status.Players = &agonesv1.PlayerStatus{
		Count: 0,
	}
	ctrl.gsWatch.Add(gs)
	gs = gs.DeepCopy()
	gs.Status.Players.Count = 1
	ctrl.gsWatch.Modify(gs)

	require.Eventually(t, func() bool {
		gs, err := ctrl.gameServerLister.GameServers(gs.ObjectMeta.Namespace).Get(gs.ObjectMeta.Name)
		if gs == nil || err != nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.Players.Count == 1
	}, 5*time.Second, time.Second)
	ctrl.collect()
}

func setupGameServerStateDuration(_ *testing.T, ctrl *fakeController) {
	creationTimestamp := metav1.Now()
	currentTime := creationTimestamp.Local()
	// Add one second each time Duration is calculated
	ctrl.now = func() time.Time {
		currentTime = currentTime.Add(1 * time.Second)
		return currentTime
	}

	gs1 := gameServerWithFleetStateCreationTimestamp("test-fleet", "exampleGameServer1", "", creationTimestamp)
	gs2 := gameServerWithFleetStateCreationTimestamp("test-fleet", "exampleGameServer1", agonesv1.GameServerStateCreating, creationTimestamp)

	ctrl.gsWatch.Modify(gs1)
	ctrl.gsWatch.Modify(gs2)
	ctrl.collect()
}

// getMetricNames returns all metric view names.
func getMetricNames() []string {
	var metricNames []string
	for _, v := range stateViews {
		metricName := "agones_" + v.Name

		// Check if the aggregation type is Distribution
		if v.Aggregation.Type == view.AggTypeDistribution {
			// If it's a distribution, we append _bucket, _sum, and _count
			metricNames = append(metricNames,
				metricName+"_bucket",
				metricName+"_sum",
				metricName+"_count",
			)
		} else {
			metricNames = append(metricNames, metricName)

		}
	}
	return metricNames
}

func collectMetricNames(resp *http.Response) map[string]bool {
	metrics := make(map[string]bool)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			// Extract only the metric name, excluding labels
			metricName := fields[0]
			if idx := strings.Index(metricName, "{"); idx != -1 {
				metricName = metricName[:idx]
			}
			metrics[metricName] = true
		}
	}
	return metrics
}
