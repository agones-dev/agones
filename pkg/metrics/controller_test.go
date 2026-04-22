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
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stesting "k8s.io/client-go/testing"
)

const defaultNs = "default"

type metricExporter struct {
	metrics []*metricdata.Metric
}

func (e *metricExporter) ExportMetrics(_ context.Context, metrics []*metricdata.Metric) error {
	e.metrics = metrics
	return nil
}

func serialize(args []string) string {
	return strings.Join(args, "|")
}

type expectedMetricData struct {
	labels []string
	val    interface{}
}

// Asserts that the given metric exporter contains the expected metric data within a timeout (10s).
// For Distribution metrics, this function automatically handles the synchronization and reset pattern:
// it waits for the metric to be available, resets the view to clear accumulated data, collects once, then validates.
func assertMetricData(t *testing.T, c *fakeController, f func(), reader *metricexport.Reader, metricName string, expected []expectedMetricData) {
	t.Helper()

	// Check if any expected value is a Distribution type
	hasDistribution := false
	for _, e := range expected {
		if _, ok := e.val.(*metricdata.Distribution); ok {
			hasDistribution = true
			break
		}
	}

	// For Distribution metrics, handle the synchronization and reset pattern
	if hasDistribution {
		// Wait until the metric is available
		assert.Eventually(t, func() bool {
			f()
			exporter := &metricExporter{}
			c.lock.Lock()
			reader.ReadAndExport(exporter)
			c.lock.Unlock()
			for _, m := range exporter.metrics {
				if m.Descriptor.Name == metricName {
					return true
				}
			}
			return false
		}, 10*time.Second, 100*time.Millisecond)

		// Reset the Distribution view to clear accumulated data from the Eventually loop
		resetViews([]string{metricName})

		// Collect fresh metrics once after reset
		f()
	}

	// Now validate the metrics (without re-collecting for Distribution)
	validateFunc := f
	if hasDistribution {
		validateFunc = func() {} // Don't re-collect for Distribution metrics
	}

	assert.EventuallyWithT(t, func(ct *assert.CollectT) {
		validateFunc()

		exporter := &metricExporter{}
		c.lock.Lock()
		reader.ReadAndExport(exporter)
		c.lock.Unlock()
		expectedValuesAsMap := make(map[string]expectedMetricData)
		for _, e := range expected {
			expectedValuesAsMap[serialize(e.labels)] = e
		}

		var wantedMetric *metricdata.Metric
		for _, m := range exporter.metrics {
			if m.Descriptor.Name == metricName {
				wantedMetric = m
			}
		}
		require.NotNil(ct, wantedMetric, "No metric found with name: %s", metricName)

		assert.Equal(ct, len(expectedValuesAsMap), len(expected), "Multiple entries in 'expected' slice have the exact same labels")
		assert.Equalf(ct, len(expectedValuesAsMap), len(wantedMetric.TimeSeries), "number of timeseries does not match under metric: %v", metricName)
		for _, tsd := range wantedMetric.TimeSeries {
			actualLabelValues := make([]string, len(tsd.LabelValues))
			for i, k := range tsd.LabelValues {
				actualLabelValues[i] = k.Value
			}
			e, ok := expectedValuesAsMap[serialize(actualLabelValues)]
			assert.True(ct, ok, "no TimeSeries found with labels: %v", actualLabelValues)
			assert.Equal(ct, e.labels, actualLabelValues, "label values don't match")
			assert.Equal(ct, 1, len(tsd.Points), "assertMetricDataValues can only handle a single Point in a TimeSeries")
			assert.Equal(ct, e.val, tsd.Points[0].Value, "metric: %s, tags: %v, values don't match; got: %v, want: %v", metricName, tsd.LabelValues, tsd.Points[0].Value, e.val)
		}
	}, 10*time.Second, 100*time.Millisecond)
}

func resetMetrics() {
	unRegisterViews()
	registerViews()
}

var mu sync.Mutex

func TestControllerFleetReplicasCount_ResetMetricsOnDelete(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()

	resetMetrics()
	exporter := &metricExporter{}
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	f := fleet("fleet-test", 8, 2, 5, 1, 1)
	fd := fleet("fleet-deleted", 100, 100, 100, 100, 100)
	c.fleetWatch.Add(f)
	f = f.DeepCopy()
	f.Status.ReadyReplicas = 1
	f.Spec.Replicas = 5
	c.fleetWatch.Modify(f)
	c.fleetWatch.Add(fd)
	c.fleetWatch.Delete(fd)

	c.lock.Lock()
	reader.ReadAndExport(exporter)
	c.lock.Unlock()
	assertMetricData(t, c, func() {}, reader, fleetReplicaCountName, []expectedMetricData{
		{labels: []string{"fleet-test", defaultNs, "reserved"}, val: int64(1)},
		{labels: []string{"fleet-test", defaultNs, "allocated"}, val: int64(2)},
		{labels: []string{"fleet-test", defaultNs, "desired"}, val: int64(5)},
		{labels: []string{"fleet-test", defaultNs, "ready"}, val: int64(1)},
		{labels: []string{"fleet-test", defaultNs, "total"}, val: int64(8)},
	})
}

func TestControllerFleetAutoScalerState_ResetMetricsOnDelete(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()

	resetMetrics()
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	// testing fleet name change
	fasFleetNameChange := fleetAutoScaler("first-fleet", "name-switch")
	c.fasWatch.Add(fasFleetNameChange)
	fasFleetNameChange = fasFleetNameChange.DeepCopy()
	fasFleetNameChange.Spec.Policy.Buffer.BufferSize = intstr.FromInt(10)
	fasFleetNameChange.Spec.Policy.Buffer.MaxReplicas = 50
	fasFleetNameChange.Spec.Policy.Buffer.MinReplicas = 10
	fasFleetNameChange.Status.CurrentReplicas = 20
	fasFleetNameChange.Status.DesiredReplicas = 10
	fasFleetNameChange.Status.ScalingLimited = true
	c.fasWatch.Modify(fasFleetNameChange)
	fasFleetNameChange = fasFleetNameChange.DeepCopy()
	fasFleetNameChange.Spec.FleetName = "second-fleet"
	c.fasWatch.Modify(fasFleetNameChange)
	// testing deletion
	fasDeleted := fleetAutoScaler("deleted-fleet", "deleted")
	fasDeleted.Spec.Policy.Buffer.BufferSize = intstr.FromString("50%")
	fasDeleted.Spec.Policy.Buffer.MaxReplicas = 150
	fasDeleted.Spec.Policy.Buffer.MinReplicas = 15
	c.fasWatch.Add(fasDeleted)
	c.fasWatch.Delete(fasDeleted)
	require.True(t, c.sync())

	nop := func() {}
	assertMetricData(t, c, nop, reader, fleetAutoscalersAbleToScaleName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs}, val: int64(1)},
	})
	assertMetricData(t, c, nop, reader, fleetAutoscalerBufferLimitName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs, "max"}, val: int64(50)},
		{labels: []string{"second-fleet", "name-switch", defaultNs, "min"}, val: int64(10)},
	})
	assertMetricData(t, c, nop, reader, fleetAutoscalterBufferSizeName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs, "count"}, val: int64(10)},
	})
	assertMetricData(t, c, nop, reader, fleetAutoscalerCurrentReplicaCountName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs}, val: int64(20)},
	})
	assertMetricData(t, c, nop, reader, fleetAutoscalersDesiredReplicaCountName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs}, val: int64(10)},
	})
	assertMetricData(t, c, nop, reader, fleetAutoscalersLimitedName, []expectedMetricData{
		{labels: []string{"second-fleet", "name-switch", defaultNs}, val: int64(1)},
	})
}

func TestControllerGameServerCount(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	resetMetrics()
	reader := metricexport.NewReader()

	c := newFakeController()
	defer c.close()

	c.run(t)
	require.True(t, c.sync())

	gs1 := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateCreating)
	c.gsWatch.Add(gs1)
	require.Eventually(t, func() bool {
		gs, err := c.gameServerLister.GameServers(gs1.ObjectMeta.Namespace).Get(gs1.ObjectMeta.Name)
		if err != nil || gs == nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.State == agonesv1.GameServerStateCreating
	}, 10*time.Second, time.Second)
	c.collect()

	gs1 = gs1.DeepCopy()
	gs1.Status.State = agonesv1.GameServerStateReady
	c.gsWatch.Modify(gs1)
	require.Eventually(t, func() bool {
		gs, err := c.gameServerLister.GameServers(gs1.ObjectMeta.Namespace).Get(gs1.ObjectMeta.Name)
		if err != nil || gs == nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.State == agonesv1.GameServerStateReady
	}, 10*time.Second, time.Second)
	c.collect()

	gs1 = gs1.DeepCopy()
	gs1.Status.State = agonesv1.GameServerStateShutdown
	c.gsWatch.Modify(gs1)
	c.gsWatch.Add(gameServerWithFleetAndState("", agonesv1.GameServerStatePortAllocation))
	c.gsWatch.Add(gameServerWithFleetAndState("", agonesv1.GameServerStatePortAllocation))

	require.True(t, c.sync())

	assertMetricData(t, c, func() {
		c.collect()
	}, reader, gameServersCountName, []expectedMetricData{
		{labels: []string{"test-fleet", defaultNs, "Creating"}, val: int64(0)},
		{labels: []string{"test-fleet", defaultNs, "Ready"}, val: int64(0)},
		{labels: []string{"test-fleet", defaultNs, "Shutdown"}, val: int64(1)},
		{labels: []string{"none", defaultNs, "PortAllocation"}, val: int64(2)},
	})
}

func TestControllerGameServerPlayerConnectedCount(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	runtime.EnableAllFeatures()
	resetMetrics()
	reader := metricexport.NewReader()

	c := newFakeController()
	defer c.close()

	gs1 := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateReady)
	gs1.Status.Players = &agonesv1.PlayerStatus{
		Count: 0,
	}
	c.gsWatch.Add(gs1)
	gs1 = gs1.DeepCopy()
	gs1.Status.Players.Count = 1
	c.gsWatch.Modify(gs1)

	c.run(t)
	require.True(t, c.sync())
	require.Eventually(t, func() bool {
		gs, err := c.gameServerLister.GameServers(gs1.ObjectMeta.Namespace).Get(gs1.ObjectMeta.Name)
		if err != nil || gs == nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.Players.Count == 1
	}, 10*time.Second, time.Second)
	c.collect()

	gs1 = gs1.DeepCopy()
	gs1.Status.Players.Count = 4
	c.gsWatch.Modify(gs1)

	c.run(t)
	require.True(t, c.sync())
	require.Eventually(t, func() bool {
		gs, err := c.gameServerLister.GameServers(gs1.ObjectMeta.Namespace).Get(gs1.ObjectMeta.Name)
		if err != nil || gs == nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.Players.Count == 4
	}, 10*time.Second, time.Second)

	assertMetricData(t, c, func() {
		c.collect()
	}, reader, gameServersPlayerConnectedTotalName, []expectedMetricData{
		{labels: []string{"test-fleet", gs1.GetName(), defaultNs}, val: int64(4)},
	})
}

func TestControllerGameServerPlayerCapacityCount(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	runtime.EnableAllFeatures()
	resetMetrics()
	reader := metricexport.NewReader()

	c := newFakeController()
	defer c.close()

	gs1 := gameServerWithFleetAndState("test-fleet", agonesv1.GameServerStateReady)
	gs1.Status.Players = &agonesv1.PlayerStatus{
		Capacity: 4,
		Count:    0,
	}
	c.gsWatch.Add(gs1)
	gs1 = gs1.DeepCopy()
	gs1.Status.Players.Count = 1
	c.gsWatch.Modify(gs1)

	c.run(t)
	require.True(t, c.sync())
	require.Eventually(t, func() bool {
		gs, err := c.gameServerLister.GameServers(gs1.ObjectMeta.Namespace).Get(gs1.ObjectMeta.Name)
		if err != nil || gs == nil {
			return false
		}
		assert.NoError(t, err)
		return gs.Status.Players.Count == 1
	}, 10*time.Second, time.Second)

	assertMetricData(t, c, func() {}, reader, gameServersPlayerCapacityTotalName, []expectedMetricData{
		{labels: []string{"test-fleet", gs1.GetName(), defaultNs}, val: int64(3)},
	})
}

func TestControllerGameServersTotal(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	resetMetrics()
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	// deleted gs should not be counted
	gs := gameServerWithFleetAndState("deleted", agonesv1.GameServerStateCreating)
	c.gsWatch.Add(gs)
	c.gsWatch.Delete(gs)

	generateGsEvents(16, agonesv1.GameServerStateCreating, "test", c.gsWatch)
	generateGsEvents(15, agonesv1.GameServerStateScheduled, "test", c.gsWatch)
	generateGsEvents(10, agonesv1.GameServerStateStarting, "test", c.gsWatch)
	generateGsEvents(1, agonesv1.GameServerStateUnhealthy, "test", c.gsWatch)
	generateGsEvents(19, agonesv1.GameServerStateCreating, "", c.gsWatch)
	generateGsEvents(18, agonesv1.GameServerStateScheduled, "", c.gsWatch)
	generateGsEvents(16, agonesv1.GameServerStateStarting, "", c.gsWatch)
	generateGsEvents(1, agonesv1.GameServerStateUnhealthy, "", c.gsWatch)

	expected := 96
	assert.Eventually(t, func() bool {
		list, err := c.gameServerLister.GameServers(gs.ObjectMeta.Namespace).List(labels.Everything())
		if err != nil || list == nil {
			return false
		}
		require.NoError(t, err)
		return len(list) == expected
	}, 10*time.Second, time.Second)
	// While these values are tested above, the following test checks will provide a more detailed diff output
	// in the case where the assert.Eventually(...) case fails, which makes failing tests easier to debug.

	list, err := c.gameServerLister.GameServers(gs.ObjectMeta.Namespace).List(labels.Everything())
	require.NoError(t, err)
	require.Len(t, list, expected)
	assertMetricData(t, c, func() {}, reader, gameServersTotalName, []expectedMetricData{
		{labels: []string{"test", defaultNs, "Creating"}, val: int64(16)},
		{labels: []string{"test", defaultNs, "Scheduled"}, val: int64(15)},
		{labels: []string{"test", defaultNs, "Starting"}, val: int64(10)},
		{labels: []string{"test", defaultNs, "Unhealthy"}, val: int64(1)},
		{labels: []string{"none", defaultNs, "Creating"}, val: int64(19)},
		{labels: []string{"none", defaultNs, "Scheduled"}, val: int64(18)},
		{labels: []string{"none", defaultNs, "Starting"}, val: int64(16)},
		{labels: []string{"none", defaultNs, "Unhealthy"}, val: int64(1)},
	})
}

func TestControllerFleetOnDeleting(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	resetMetrics()
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	deletionTime := metav1.NewTime(time.Now())

	fd := fleet("fleet-deleting", 100, 100, 100, 100, 100)
	ft := fleet("fleet-test", 8, 2, 5, 1, 1)
	c.fleetWatch.Add(fd)

	fd = fd.DeepCopy()
	fd.DeletionTimestamp = &deletionTime
	c.fleetWatch.Modify(fd)

	c.fleetWatch.Add(ft)
	ft = ft.DeepCopy()
	ft.Status.Replicas = 15
	c.fleetWatch.Modify(ft)

	assertMetricData(t, c, func() {}, reader, fleetReplicaCountName, []expectedMetricData{
		{labels: []string{"fleet-deleting", defaultNs, "total"}, val: int64(100)},
		{labels: []string{"fleet-deleting", defaultNs, "allocated"}, val: int64(100)},
		{labels: []string{"fleet-deleting", defaultNs, "ready"}, val: int64(100)},
		{labels: []string{"fleet-deleting", defaultNs, "desired"}, val: int64(100)},
		{labels: []string{"fleet-deleting", defaultNs, "reserved"}, val: int64(100)},

		{labels: []string{"fleet-test", defaultNs, "total"}, val: int64(15)},
		{labels: []string{"fleet-test", defaultNs, "allocated"}, val: int64(2)},
		{labels: []string{"fleet-test", defaultNs, "ready"}, val: int64(5)},
		{labels: []string{"fleet-test", defaultNs, "desired"}, val: int64(1)},
		{labels: []string{"fleet-test", defaultNs, "reserved"}, val: int64(1)},
	})
}

func TestControllerFleetAutoScalerOnDeleting(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	resetMetrics()
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()
	c.run(t)

	deletionTime := metav1.NewTime(time.Now())

	fas := fleetAutoScaler("fleet-deleting", "fas-deleting")
	fast := fleetAutoScaler("fleet-test", "fas-test")
	c.fasWatch.Add(fas)

	fas = fas.DeepCopy()
	fas.Status.CurrentReplicas = 15
	c.fasWatch.Modify(fas)
	fas = fas.DeepCopy()
	fas.DeletionTimestamp = &deletionTime
	c.fasWatch.Modify(fas)

	c.fasWatch.Add(fast)

	fast = fast.DeepCopy()
	fast.Status.CurrentReplicas = 5
	c.fasWatch.Modify(fast)

	assertMetricData(t, c, func() {}, reader, fleetAutoscalerCurrentReplicaCountName, []expectedMetricData{
		{labels: []string{"fleet-deleting", "fas-deleting", defaultNs}, val: int64(15)},
		{labels: []string{"fleet-test", "fas-test", defaultNs}, val: int64(5)},
	})
}

func TestControllerGameServersNodeState(t *testing.T) {
	mu.Lock()
	defer mu.Unlock()
	resetMetrics()
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

	c := newFakeControllerWithMock(m)
	defer c.close()
	require.True(t, c.sync())
	reader := metricexport.NewReader()

	assertMetricData(t, c, func() { c.collect() }, reader, gameServersNodeCountName, []expectedMetricData{
		{labels: []string{}, val: &metricdata.Distribution{
			Count:                 3,
			Sum:                   3,
			SumOfSquaredDeviation: 2,
			BucketOptions:         &metricdata.BucketOptions{Bounds: []float64{0.00001, 1.00001, 2.00001, 3.00001, 4.00001, 5.00001, 6.00001, 7.00001, 8.00001, 9.00001, 10.00001, 11.00001, 12.00001, 13.00001, 14.00001, 15.00001, 16.00001, 32.00001, 40.00001, 50.00001, 60.00001, 70.00001, 80.00001, 90.00001, 100.00001, 110.00001, 120.00001}},
			Buckets:               []metricdata.Bucket{{Count: 1}, {Count: 1}, {Count: 1}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}, {Count: 0}}}},
	})
	assertMetricData(t, c, func() { c.collect() }, reader, nodeCountName, []expectedMetricData{
		{labels: []string{"true"}, val: int64(1)},
		{labels: []string{"false"}, val: int64(2)},
	})
}

func TestFleetCountersAndListsMetrics(t *testing.T) {
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	mu.Lock()
	defer mu.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	resetMetrics()
	reader := metricexport.NewReader()
	c := newFakeController()
	defer c.close()

	fleetName := "cl-fleet-test"
	counterName := "players"
	counters := map[string]agonesv1.AggregatedCounterStatus{
		counterName: {
			AllocatedCount:    24,
			AllocatedCapacity: 30,
			Count:             28,
			Capacity:          50,
		},
	}
	listName := "rooms"
	lists := map[string]agonesv1.AggregatedListStatus{
		listName: {
			AllocatedCount:    4,
			AllocatedCapacity: 6,
			Count:             1,
			Capacity:          100,
		},
	}

	f := fleet(fleetName, 8, 3, 5, 8, 0)
	c.fleetWatch.Add(f)
	f = f.DeepCopy()
	f.Status.Lists = lists
	f.Status.Counters = counters
	c.fleetWatch.Modify(f)

	c.run(t)
	require.True(t, c.sync())
	require.Eventually(t, func() bool {
		fl, err := c.fleetLister.Fleets(f.GetObjectMeta().GetNamespace()).Get(fleetName)
		if fl == nil || err != nil {
			return false
		}
		assert.NoError(t, err)
		return cmp.Equal(counters, fl.Status.Counters) && cmp.Equal(lists, fl.Status.Lists)

	}, 10*time.Second, time.Second)
	c.collect()

	assertMetricData(t, c, func() {
		c.collect()
	}, reader, fleetCountersName, []expectedMetricData{
		// keyCounter, keyName, keyNamespace, keyType
		{labels: []string{counterName, fleetName, defaultNs, "allocated_count"}, val: int64(24)},
		{labels: []string{counterName, fleetName, defaultNs, "allocated_capacity"}, val: int64(30)},
		{labels: []string{counterName, fleetName, defaultNs, "total_count"}, val: int64(28)},
		{labels: []string{counterName, fleetName, defaultNs, "total_capacity"}, val: int64(50)},
	})
	assertMetricData(t, c, func() {
		c.collect()
	}, reader, fleetListsName, []expectedMetricData{
		// keyList, keyName, keyNamespace, keyType
		{labels: []string{listName, fleetName, defaultNs, "allocated_count"}, val: int64(4)},
		{labels: []string{listName, fleetName, defaultNs, "allocated_capacity"}, val: int64(6)},
		{labels: []string{listName, fleetName, defaultNs, "total_count"}, val: int64(1)},
		{labels: []string{listName, fleetName, defaultNs, "total_capacity"}, val: int64(100)},
	})
}

func TestCalcDuration(t *testing.T) {
	m := agtesting.NewMocks()
	c := NewController(
		m.KubeClient,
		m.AgonesClient,
		m.KubeInformerFactory,
		m.AgonesInformerFactory)
	creationTimestamp := metav1.Now()
	futureTimestamp := metav1.NewTime(time.Now().Add(24 * time.Hour))
	gsName1 := "exampleGameServer1"
	gsName2 := "exampleGameServer2"
	currentTime := creationTimestamp.Local()
	// Add one second each time Duration is calculated
	c.now = func() time.Time {
		currentTime = currentTime.Add(1 * time.Second)
		return currentTime
	}
	type result struct {
		duration float64
		err      error
	}
	fleet1 := "test-fleet"
	fleet2 := ""
	var testCases = []struct {
		description string
		gs1         *agonesv1.GameServer
		gs2         *agonesv1.GameServer
		expected    result
	}{
		{
			description: "GameServer creating - first measurement",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, "", creationTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateCreating, creationTimestamp),
			expected: result{
				err:      nil,
				duration: 1,
			},
		},
		{
			description: "Test state change of a GameServer",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateCreating, creationTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateRequestReady, creationTimestamp),
			expected: result{
				err:      nil,
				duration: 1,
			},
		},
		{
			description: "gs1 state should already be deleted, error should be generated (emulation of evicted key for gs1)",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, "", creationTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateRequestReady, creationTimestamp),
			expected: result{
				err:      errors.Errorf("unable to calculate '' state duration of '%s' GameServer", gsName1),
				duration: 0,
			},
		},
		{
			description: "Shutdown state should remove the key in LRU cache",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateRequestReady, creationTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateShutdown, creationTimestamp),
			expected: result{
				err:      nil,
				duration: 2,
			},
		},
		{
			description: "Cache miss, no key in LRU cache",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateRequestReady, creationTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet1, gsName1, agonesv1.GameServerStateShutdown, creationTimestamp),
			expected: result{
				err:      errors.Errorf("unable to calculate 'RequestReady' state duration of '%s' GameServer", gsName1),
				duration: 0,
			},
		},
		{
			description: "Future timestamp was used",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet2, gsName2, "", futureTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet2, gsName2, agonesv1.GameServerStateCreating, futureTimestamp),
			expected: result{
				err:      errors.Errorf("negative duration for '' state of '%s' GameServer", gsName2),
				duration: 0,
			},
		},
		{
			description: "Shutdown state - remove a key from the LRU",
			gs1:         gameServerWithFleetStateCreationTimestamp(fleet2, gsName2, agonesv1.GameServerStateCreating, futureTimestamp),
			gs2:         gameServerWithFleetStateCreationTimestamp(fleet2, gsName2, agonesv1.GameServerStateShutdown, futureTimestamp),
			expected: result{
				err:      nil,
				duration: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Do not use t.Parallel(), because test cases should be executed as serial tests
			// Test case 3 depends on key eviction in test case 2
			duration, err := c.calcDuration(tc.gs1, tc.gs2)
			if tc.expected.err != nil {
				assert.EqualError(t, err, tc.expected.err.Error(), "We should receive an error, metric should not be measured")
			} else {
				assert.NoError(t, err, "Unable to caculate duration of a particular state")
			}
			assert.Equal(t, tc.expected.duration, duration, "Time diff should be calculated properly")
		})
	}
	assert.Len(t, c.gameServerStateLastChange.Keys(), 0, "We should not have any keys after the test")
}

func TestIsSystemNode(t *testing.T) {
	cases := []struct {
		desc     string
		node     *corev1.Node
		expected bool
	}{
		{
			desc: "Is system node, true expected",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{{Key: "agones.dev/test"}},
				},
			},
			expected: true,
		},
		{
			desc: "Not a system node, false expected",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{{Key: "qwerty.dev/test"}},
				},
			},
			expected: false,
		},
		{
			desc: "Empty taints, false expected",
			node: &corev1.Node{
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{},
				},
			},
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := isSystemNode(tc.node)
			assert.Equal(t, tc.expected, res)
		})
	}
}
