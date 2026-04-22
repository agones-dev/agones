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
	"net/url"
	"time"

	"agones.dev/agones/pkg/util/runtime"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/metrics"
	"k8s.io/client-go/util/workqueue"
)

var (
	keyQueueName = MustTagKey("queue_name")

	httpRequestTotalStats   = stats.Int64("http/request_total", "The total of HTTP requests.", "1")
	httpRequestLatencyStats = stats.Float64("http/latency", "The duration of HTTP requests.", "s")

	cacheListTotalStats           = stats.Float64("cache/list_total", "The total number of list operations.", "1")
	cacheListLatencyStats         = stats.Float64("cache/list_latency", "Duration of a Kubernetes API call in seconds", "s")
	cacheListItemCountStats       = stats.Float64("cache/list_items_count", "Count of items in a list from the Kubernetes API.", "1")
	cacheWatchesTotalStats        = stats.Float64("cache/watches_total", "Total number of watch operations.", "1")
	cacheShortWatchesTotalStats   = stats.Float64("cache/short_watches_total", "Total number of short watch operations.", "1")
	cacheWatchesLatencyStats      = stats.Float64("cache/watches_latency", "Duration of watches on the Kubernetes API.", "s")
	cacheItemsInWatchesCountStats = stats.Float64("cache/watch_events", "Number of items in watches on the Kubernetes API.", "1")
	cacheLastResourceVersionStats = stats.Float64("cache/last_resource_version", "Last resource version from the Kubernetes API.", "1")

	workQueueDepthStats                   = stats.Float64("workqueue/depth", "Current depth of the work queue.", "1")
	workQueueItemsTotalStats              = stats.Float64("workqueue/items_total", "Total number of items added to the work queue.", "1")
	workQueueLatencyStats                 = stats.Float64("workqueue/latency", "How long an item stays in the work queue.", "s")
	workQueueWorkDurationStats            = stats.Float64("workqueue/work_duration", "How long processing an item from the work queue takes.", "s")
	workQueueRetriesTotalStats            = stats.Float64("workqueue/retries_total", "Total number of items retried to the work queue.", "1")
	workQueueLongestRunningProcessorStats = stats.Float64("workqueue/longest_running_processor", "How long the longest workqueue processors been running in microseconds.", "1")
	workQueueUnfinishedWorkStats          = stats.Float64("workqueue/unfinished_work", "How long has unfinished work been in the workqueue.", "1")
)

func init() {
	distributionSeconds := []float64{0, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2, 3}
	distributionNumbers := []float64{0, 10, 50, 100, 150, 250, 300}

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_http_request_total",
		Measure:     httpRequestTotalStats,
		Description: "The total of HTTP requests to the Kubernetes API by status code",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyVerb, keyStatusCode},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_http_request_duration_seconds",
		Measure:     httpRequestLatencyStats,
		Description: "The distribution of HTTP requests latencies to the Kubernetes API by status code",
		Aggregation: view.Distribution(distributionSeconds...),
		TagKeys:     []tag.Key{keyVerb, keyEndpoint},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_list_total",
		Measure:     cacheListTotalStats,
		Description: "The total number of list operations for client-go caches",
		Aggregation: view.Count(),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_list_duration_seconds",
		Measure:     cacheListLatencyStats,
		Description: "Duration of a Kubernetes list API call in seconds",
		Aggregation: view.Distribution(distributionSeconds...),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_list_items",
		Measure:     cacheListItemCountStats,
		Description: "Count of items in a list from the Kubernetes API.",
		Aggregation: view.Distribution(distributionNumbers...),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_watches_total",
		Measure:     cacheWatchesTotalStats,
		Description: "The total number of watch operations for client-go caches",
		Aggregation: view.Count(),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_short_watches_total",
		Measure:     cacheShortWatchesTotalStats,
		Description: "The total number of short watch operations for client-go caches",
		Aggregation: view.Count(),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_watch_duration_seconds",
		Measure:     cacheWatchesLatencyStats,
		Description: "Duration of watches on the Kubernetes API.",
		Aggregation: view.Distribution(distributionSeconds...),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_watch_events",
		Measure:     cacheItemsInWatchesCountStats,
		Description: "Number of items in watches on the Kubernetes API.",
		Aggregation: view.Distribution(distributionNumbers...),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_cache_last_resource_version",
		Measure:     cacheLastResourceVersionStats,
		Description: "Last resource version from the Kubernetes API.",
		Aggregation: view.LastValue(),
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_depth",
		Measure:     workQueueDepthStats,
		Description: "Current depth of the work queue.",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_items_total",
		Measure:     workQueueItemsTotalStats,
		Description: "Total number of items added to the work queue.",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_latency_seconds",
		Measure:     workQueueLatencyStats,
		Description: "How long an item stays in the work queue.",
		Aggregation: view.Distribution(distributionSeconds...),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_work_duration_seconds",
		Measure:     workQueueWorkDurationStats,
		Description: "How long processing an item from the work queue takes.",
		Aggregation: view.Distribution(distributionSeconds...),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_retries_total",
		Measure:     workQueueRetriesTotalStats,
		Description: "Total number of items retried to the work queue.",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_longest_running_processor",
		Measure:     workQueueLongestRunningProcessorStats,
		Description: "How long the longest running workqueue processor has been running in microseconds.",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	runtime.Must(view.Register(&view.View{
		Name:        "k8s_client_workqueue_unfinished_work_seconds",
		Measure:     workQueueUnfinishedWorkStats,
		Description: "How long unfinished work has been sitting in the workqueue in seconds.",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{keyQueueName},
	}))

	clientGoRequest := &clientGoMetricAdapter{}
	clientGoRequest.Register()
}

// Definition of client-go metrics adapter for HTTP requests, caches and workerqueues observations
type clientGoMetricAdapter struct{}

func (c *clientGoMetricAdapter) Register() {
	metrics.Register(metrics.RegisterOpts{
		RequestLatency: c,
		RequestResult:  c,
	})
	workqueue.SetProvider(c)
}

func (clientGoMetricAdapter) Increment(ctx context.Context, code string, method string, _ string) {
	RecordWithTags(ctx, []tag.Mutator{tag.Insert(keyStatusCode, code),
		tag.Insert(keyVerb, method)}, httpRequestTotalStats.M(int64(1)))
}

func (clientGoMetricAdapter) Observe(ctx context.Context, verb string, u url.URL, latency time.Duration) {
	// url is without {namespace} and {name}, so cardinality of resulting metrics is low.
	RecordWithTags(ctx, []tag.Mutator{tag.Insert(keyVerb, verb),
		tag.Insert(keyEndpoint, u.Path)}, httpRequestLatencyStats.M(latency.Seconds()))
}

// ocMetric adapts OpenCensus measures to cache metrics
type ocMetric struct {
	*stats.Float64Measure
	ctx context.Context
}

func newOcMetric(m *stats.Float64Measure) *ocMetric {
	return &ocMetric{
		Float64Measure: m,
		ctx:            context.Background(),
	}
}

func (m *ocMetric) withTag(key tag.Key, value string) *ocMetric {
	ctx, err := tag.New(m.ctx, tag.Upsert(key, value))
	if err != nil {
		panic(err)
	}
	m.ctx = ctx
	return m
}

func (m *ocMetric) Inc() {
	stats.Record(m.ctx, m.Float64Measure.M(float64(1)))
}

func (m *ocMetric) Dec() {
	stats.Record(m.ctx, m.Float64Measure.M(float64(-1)))
}

// observeFunc is an adapter that allows the use of functions as summary metric.
// useful for converting metrics unit before sending them to OC
type observeFunc func(float64)

func (o observeFunc) Observe(f float64) {
	o(f)
}

func (m *ocMetric) Observe(f float64) {
	stats.Record(m.ctx, m.Float64Measure.M(f))
}

func (m *ocMetric) Set(f float64) {
	stats.Record(m.ctx, m.Float64Measure.M(f))
}

func (clientGoMetricAdapter) NewListsMetric(string) cache.CounterMetric {
	return newOcMetric(cacheListTotalStats)
}

func (clientGoMetricAdapter) NewListDurationMetric(string) cache.SummaryMetric {
	return newOcMetric(cacheListLatencyStats)
}

func (clientGoMetricAdapter) NewItemsInListMetric(string) cache.SummaryMetric {
	return newOcMetric(cacheListItemCountStats)
}

func (clientGoMetricAdapter) NewWatchesMetric(string) cache.CounterMetric {
	return newOcMetric(cacheWatchesTotalStats)
}

func (clientGoMetricAdapter) NewShortWatchesMetric(string) cache.CounterMetric {
	return newOcMetric(cacheShortWatchesTotalStats)
}

func (clientGoMetricAdapter) NewWatchDurationMetric(string) cache.SummaryMetric {
	return newOcMetric(cacheWatchesLatencyStats)
}

func (clientGoMetricAdapter) NewItemsInWatchMetric(string) cache.SummaryMetric {
	return newOcMetric(cacheItemsInWatchesCountStats)
}

func (clientGoMetricAdapter) NewLastResourceVersionMetric(string) cache.GaugeMetric {
	return newOcMetric(cacheLastResourceVersionStats)
}

func (clientGoMetricAdapter) NewDepthMetric(name string) workqueue.GaugeMetric {
	return newOcMetric(workQueueDepthStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewAddsMetric(name string) workqueue.CounterMetric {
	return newOcMetric(workQueueItemsTotalStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewLatencyMetric(name string) workqueue.HistogramMetric {
	m := newOcMetric(workQueueLatencyStats).withTag(keyQueueName, name)
	// Convert microseconds to seconds for consistency across metrics.
	return observeFunc(func(f float64) {
		m.Observe(f / 1e6)
	})
}

func (clientGoMetricAdapter) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	m := newOcMetric(workQueueWorkDurationStats).withTag(keyQueueName, name)
	// Convert microseconds to seconds for consistency across metrics.
	return observeFunc(func(f float64) {
		m.Observe(f / 1e6)
	})
}

func (clientGoMetricAdapter) NewRetriesMetric(name string) workqueue.CounterMetric {
	return newOcMetric(workQueueRetriesTotalStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewLongestRunningProcessorSecondsMetric(string) workqueue.SettableGaugeMetric {
	return newOcMetric(workQueueLongestRunningProcessorStats)
}

func (clientGoMetricAdapter) NewUnfinishedWorkSecondsMetric(string) workqueue.SettableGaugeMetric {
	return newOcMetric(workQueueUnfinishedWorkStats)
}

func (clientGoMetricAdapter) NewDeprecatedDepthMetric(name string) workqueue.GaugeMetric {
	return newOcMetric(workQueueDepthStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewDeprecatedAddsMetric(name string) workqueue.CounterMetric {
	return newOcMetric(workQueueItemsTotalStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewDeprecatedLatencyMetric(name string) workqueue.SummaryMetric {
	m := newOcMetric(workQueueLatencyStats).withTag(keyQueueName, name)
	// Convert microseconds to seconds for consistency across metrics.
	return observeFunc(func(f float64) {
		m.Observe(f / 1e6)
	})
}

func (clientGoMetricAdapter) NewDeprecatedLongestRunningProcessorMicrosecondsMetric(string) workqueue.SettableGaugeMetric {
	return newOcMetric(workQueueLongestRunningProcessorStats)
}

func (clientGoMetricAdapter) NewDeprecatedRetriesMetric(name string) workqueue.CounterMetric {
	return newOcMetric(workQueueRetriesTotalStats).withTag(keyQueueName, name)
}

func (clientGoMetricAdapter) NewDeprecatedUnfinishedWorkSecondsMetric(string) workqueue.SettableGaugeMetric {
	return newOcMetric(workQueueUnfinishedWorkStats)
}

func (clientGoMetricAdapter) NewDeprecatedWorkDurationMetric(name string) workqueue.SummaryMetric {
	m := newOcMetric(workQueueWorkDurationStats).withTag(keyQueueName, name)
	// Convert microseconds to seconds for consistency across metrics.
	return observeFunc(func(f float64) {
		m.Observe(f / 1e6)
	})
}
