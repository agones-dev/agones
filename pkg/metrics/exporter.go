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
	"net/http"
	"os"
	"time"

	"agones.dev/agones/pkg/util/httpserver"
	"cloud.google.com/go/compute/metadata"
	"contrib.go.opencensus.io/exporter/prometheus"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opencensus.io/stats/view"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

// Config holds configuration for metrics reporting
type Config struct {
	GCPProjectID      string
	StackdriverLabels string
	Stackdriver       bool
	PrometheusMetrics bool
}

// RegisterPrometheusExporter register a prometheus exporter to OpenCensus with a given prometheus metric registry.
// It will automatically add go runtime and process metrics using default prometheus collectors.
// The function return an http.handler that you can use to expose the prometheus endpoint.
func RegisterPrometheusExporter(registry *prom.Registry) (http.Handler, error) {
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "agones",
		Registry:  registry,
	})
	if err != nil {
		return nil, err
	}
	if err := registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}
	if err := registry.Register(collectors.NewGoCollector()); err != nil {
		return nil, err
	}
	view.RegisterExporter(pe)

	return pe, nil
}

// RegisterStackdriverExporter register a Stackdriver exporter to OpenCensus.
// It will add Agones metrics into Stackdriver on Google Cloud.
func RegisterStackdriverExporter(projectID string, defaultLabels string) (*stackdriver.Exporter, error) {
	monitoredRes, err := getMonitoredResource(projectID)
	if err != nil {
		logger.WithError(err).Warn("error discovering monitored resource")
	}
	labels, err := parseLabels(defaultLabels)
	if err != nil {
		return nil, err
	}

	sd, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: projectID,
		// MetricPrefix helps uniquely identify your metrics.
		MetricPrefix:            "agones",
		Resource:                monitoredRes,
		DefaultMonitoringLabels: labels,
	})
	if err != nil {
		return nil, err
	}

	// Register it as a metrics exporter
	view.RegisterExporter(sd)
	return sd, nil
}

// SetReportingPeriod set appropriate reporting period which depends on exporters
// we are going to use
func SetReportingPeriod(forPrometheus, forStackdriver bool) {
	// if we're using only prometheus we can report faster as we're only exposing metrics in memory
	reportingPeriod := 15 * time.Second
	if forStackdriver {
		// There is a limitation on Stackdriver that reporting should
		// be equal or more than 1 minute
		reportingPeriod = 60 * time.Second
	}

	if forStackdriver || forPrometheus {
		view.SetReportingPeriod(reportingPeriod)
	}
}

func getMonitoredResource(projectID string) (*monitoredres.MonitoredResource, error) {
	zone, err := metadata.ZoneWithContext(context.TODO())
	if err != nil {
		return nil, errors.Wrap(err, "error getting zone")
	}
	clusterName, err := metadata.InstanceAttributeValueWithContext(context.TODO(), "cluster-name")
	if err != nil {
		return nil, errors.Wrap(err, "error getting cluster-name")
	}

	return &monitoredres.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"project_id":   projectID,
			"location":     zone,
			"cluster_name": clusterName,

			// See: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/
			"namespace_name": os.Getenv("POD_NAMESPACE"),
			"pod_name":       os.Getenv("POD_NAME"),
			"container_name": os.Getenv("CONTAINER_NAME"),
		},
	}, nil
}

// SetupMetrics initializes metrics reporting with the provided configuration
func SetupMetrics(conf Config, server *httpserver.Server) (healthcheck.Handler, func()) {
	var health healthcheck.Handler
	var closer = func() {}

	// Stackriver Metrics
	if conf.Stackdriver {
		sd, err := RegisterStackdriverExporter(conf.GCPProjectID, conf.StackdriverLabels)
		if err != nil {
			logger.WithError(err).Fatal("Could not register Stackdriver exporter")
		}
		closer = func() { sd.Flush() }
	}

	// Prometheus Metrics
	if conf.PrometheusMetrics {
		registry := prom.NewRegistry()
		metricHandler, err := RegisterPrometheusExporter(registry)
		if err != nil {
			logger.WithError(err).Fatal("Could not register Prometheus exporter")
		}
		server.Handle("/metrics", metricHandler)
		health = healthcheck.NewMetricsHandler(registry, "agones")
	} else {
		health = healthcheck.NewHandler()
	}

	return health, closer
}
