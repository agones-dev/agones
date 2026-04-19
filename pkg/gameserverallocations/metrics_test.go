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

package gameserverallocations

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	gameserverv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	mt "agones.dev/agones/pkg/metrics"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/stats/view"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

type mockGameServerLister struct {
	gameServerNamespaceLister mockGameServerNamespaceLister
	gameServersCalled         bool
}

type mockGameServerNamespaceLister struct {
	gameServer *agonesv1.GameServer
}

func (s *mockGameServerLister) List(_ labels.Selector) (ret []*agonesv1.GameServer, err error) {
	return ret, nil
}

func (s *mockGameServerLister) GameServers(_ string) gameserverv1.GameServerNamespaceLister {
	s.gameServersCalled = true
	return s.gameServerNamespaceLister
}

func (s mockGameServerNamespaceLister) Get(_ string) (*agonesv1.GameServer, error) {
	return s.gameServer, nil
}

func (s mockGameServerNamespaceLister) List(_ labels.Selector) (ret []*agonesv1.GameServer, err error) {
	return ret, nil
}

func resetMetrics() {
	unRegisterViews()
	registerViews()
}

func TestSetResponse(t *testing.T) {
	subtests := []struct {
		name           string
		gameServer     *agonesv1.GameServer
		err            error
		allocation     *allocationv1.GameServerAllocation
		expectedState  allocationv1.GameServerAllocationState
		expectedCalled bool
	}{
		{
			name: "Try to get gs from local cluster for local allocation",
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{agonesv1.FleetNameLabel: "fleetName"},
				},
			},
			allocation: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "gameServerName",
					Source:         "local",
				},
			},
			expectedCalled: true,
		},
		{
			name: "Do not try to get gs from local cluster for remote allocation",
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{agonesv1.FleetNameLabel: "fleetName"},
				},
			},
			allocation: &allocationv1.GameServerAllocation{
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "gameServerName",
					Source:         "33.188.237.156:443",
				},
			},
			expectedCalled: false,
		},
	}

	for _, subtest := range subtests {
		gsl := mockGameServerLister{
			gameServerNamespaceLister: mockGameServerNamespaceLister{
				gameServer: subtest.gameServer,
			},
		}

		metrics := metrics{
			ctx:              context.Background(),
			gameServerLister: &gsl,
			logger:           runtime.NewLoggerWithSource("metrics_test"),
			start:            time.Now(),
		}

		t.Run(subtest.name, func(t *testing.T) {
			metrics.setResponse(subtest.allocation)
			assert.Equal(t, subtest.expectedCalled, gsl.gameServersCalled)
		})
	}
}

func TestAllocationMetrics(t *testing.T) {
	resetMetrics()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	conf := mt.Config{
		PrometheusMetrics: true,
	}
	server := &httpserver.Server{
		Logger: framework.TestLogger(t),
	}

	health, closer := mt.SetupMetrics(conf, server)
	defer t.Cleanup(closer)

	assert.NotNil(t, health, "Health check handler should not be nil")
	server.Handle("/", health)

	f, gsList := defaultFixtures(1)
	a, m := newFakeAllocator()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
		gsWatch.Modify(gs)

		return true, gs, nil
	})

	ctxAlloc, cancelAlloc := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
	defer cancelAlloc()

	require.NoError(t, a.Run(ctxAlloc))
	// wait for it to be up and running
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (done bool, err error) {
		return a.allocationCache.workerqueue.RunCount() == 1, nil
	})
	require.NoError(t, err)

	gsa := allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "gsa-1", Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}}}},
		}}
	gsa.ApplyDefaults()
	errs := gsa.Validate()
	require.Len(t, errs, 0)

	result, err := a.Allocate(ctxAlloc, &gsa)
	require.NoError(t, err)
	require.NotNil(t, result)

	metricsURL := startMetricsServerForTest(t, server)

	require.EventuallyWithT(t, func(e *assert.CollectT) {
		resp, err := http.Get(metricsURL)
		require.NoError(e, err, "Failed waiting for metrics endpoint readiness")
		defer func() {
			assert.NoError(e, resp.Body.Close())
		}()
		assert.Equal(e, http.StatusOK, resp.StatusCode)
	}, 5*time.Second, 10*time.Millisecond, "Failed waiting for metrics endpoint readiness")

	resp, err := http.Get(metricsURL)
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

func TestStartMetricsServer(t *testing.T) {
	t.Parallel()

	server := &httpserver.Server{
		Logger: framework.TestLogger(t),
	}
	server.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	metricsURL := startMetricsServerForTest(t, server)

	require.EventuallyWithT(t, func(e *assert.CollectT) {
		resp, err := http.Get(metricsURL)
		require.NoError(e, err)
		defer func() {
			assert.NoError(e, resp.Body.Close())
		}()
		assert.Equal(e, http.StatusOK, resp.StatusCode)
	}, time.Second, 10*time.Millisecond)
}

func startMetricsServerForTest(t *testing.T, handler http.Handler) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		Handler: handler,
	}

	var serveErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
		}
	}()

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, srv.Shutdown(shutdownCtx))
		<-done
		assert.NoError(t, serveErr)
	})

	return fmt.Sprintf("http://%s/metrics", listener.Addr().String())
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
