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

// Controller for gameservers
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct"
	"agones.dev/agones/pkg/fleetautoscalers"
	"agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/gameserversets"
	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	"github.com/google/uuid"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	enableStackdriverMetricsFlag       = "stackdriver-exporter"
	stackdriverLabels                  = "stackdriver-labels"
	enablePrometheusMetricsFlag        = "prometheus-exporter"
	projectIDFlag                      = "gcp-project-id"
	sidecarImageFlag                   = "sidecar-image"
	sidecarCPURequestFlag              = "sidecar-cpu-request"
	sidecarCPULimitFlag                = "sidecar-cpu-limit"
	sidecarMemoryRequestFlag           = "sidecar-memory-request"
	sidecarMemoryLimitFlag             = "sidecar-memory-limit"
	sidecarRunAsUserFlag               = "sidecar-run-as-user"
	sidecarRequestsRateLimitFlag       = "sidecar-requests-rate-limit"
	sdkServerAccountFlag               = "sdk-service-account"
	pullSidecarFlag                    = "always-pull-sidecar"
	minPortFlag                        = "min-port"
	maxPortFlag                        = "max-port"
	additionalPortRangesFlag           = "additional-port-ranges"
	certFileFlag                       = "cert-file"
	keyFileFlag                        = "key-file"
	numWorkersFlag                     = "num-workers"
	apiServerSustainedQPSFlag          = "api-server-qps"
	apiServerBurstQPSFlag              = "api-server-qps-burst"
	logDirFlag                         = "log-dir"
	logLevelFlag                       = "log-level"
	logSizeLimitMBFlag                 = "log-size-limit-mb"
	kubeconfigFlag                     = "kubeconfig"
	allocationBatchWaitTime            = "allocation-batch-wait-time"
	defaultResync                      = 30 * time.Second
	podNamespace                       = "pod-namespace"
	leaderElectionFlag                 = "leader-election"
	maxCreationParallelismFlag         = "max-creation-parallelism"
	maxGameServerCreationsPerBatchFlag = "max-game-server-creations-per-batch"
	maxDeletionParallelismFlag         = "max-deletion-parallelism"
	maxGameServerDeletionsPerBatchFlag = "max-game-server-deletions-per-batch"
	maxPodPendingCountFlag             = "max-pod-pending-count"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

func setupLogging(logDir string, logSizeLimitMB int) {
	logFileName := filepath.Join(logDir, "agones-controller-"+time.Now().Format("20060102_150405")+".log")

	const maxLogSizeMB = 100
	maxBackups := (logSizeLimitMB - maxLogSizeMB) / maxLogSizeMB
	logger.WithField("filename", logFileName).WithField("numbackups", maxBackups).Info("logging to file")
	logrus.SetOutput(
		io.MultiWriter(
			logrus.StandardLogger().Out,
			&lumberjack.Logger{
				Filename:   logFileName,
				MaxSize:    maxLogSizeMB,
				MaxBackups: maxBackups,
			},
		),
	)
}

// main starts the operator for the gameserver CRD
func main() {
	ctx, cancel := signals.NewSigKillContext()
	ctlConf := parseEnvFlags()

	if ctlConf.LogDir != "" {
		setupLogging(ctlConf.LogDir, ctlConf.LogSizeLimitMB)
	}

	logger.WithField("logLevel", ctlConf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(ctlConf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging.SdkServer. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	logger.WithField("version", pkg.Version).WithField("featureGates", runtime.EncodeFeatures()).
		WithField("ctlConf", ctlConf).Info("starting gameServer operator...")

	if errs := ctlConf.validate(); len(errs) != 0 {
		for _, err := range errs {
			logger.Error(err)
		}
		logger.Fatal("Could not create controller from environment or flags")
	}

	// if the kubeconfig fails InClusterBuildConfig will try in cluster config
	clientConf, err := runtime.InClusterBuildConfig(logger, ctlConf.KubeConfig)
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	clientConf.QPS = float32(ctlConf.APIServerSustainedQPS)
	clientConf.Burst = ctlConf.APIServerBurstQPS

	kubeClient, err := kubernetes.NewForConfig(clientConf)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the kubernetes clientset")
	}

	extClient, err := extclientset.NewForConfig(clientConf)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the api extension clientset")
	}

	agonesClient, err := versioned.NewForConfig(clientConf)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	controllerHooks, err := cloudproduct.NewFromFlag(ctx, kubeClient)
	if err != nil {
		logger.WithError(err).Fatal("Could not initialize cloud product")
	}

	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)

	server := &httpserver.Server{Logger: logger}
	var rs []runner
	var health healthcheck.Handler

	metricsConf := metrics.Config{
		Stackdriver:       ctlConf.Stackdriver,
		PrometheusMetrics: ctlConf.PrometheusMetrics,
		GCPProjectID:      ctlConf.GCPProjectID,
		StackdriverLabels: ctlConf.StackdriverLabels,
	}

	health, closer := metrics.SetupMetrics(metricsConf, server)
	defer closer()

	// If we are using Prometheus only exporter we can make reporting more often,
	// every 1 seconds, if we are using Stackdriver we would use 60 seconds reporting period,
	// which is a requirements of Stackdriver, otherwise most of time series would be invalid for Stackdriver
	metrics.SetReportingPeriod(ctlConf.PrometheusMetrics, ctlConf.Stackdriver)

	// Add metrics controller only if we configure one of metrics exporters
	if ctlConf.PrometheusMetrics || ctlConf.Stackdriver {
		rs = append(rs, metrics.NewController(kubeClient, agonesClient, kubeInformerFactory, agonesInformerFactory))
	}

	server.Handle("/", health)

	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	gsController := gameservers.NewController(controllerHooks, health,
		ctlConf.PortRanges, ctlConf.SidecarImage, ctlConf.AlwaysPullSidecar,
		ctlConf.SidecarCPURequest, ctlConf.SidecarCPULimit,
		ctlConf.SidecarMemoryRequest, ctlConf.SidecarMemoryLimit, ctlConf.SidecarRunAsUser, ctlConf.SidecarRequestsRateLimit, ctlConf.SdkServiceAccount,
		kubeClient, kubeInformerFactory, extClient, agonesClient, agonesInformerFactory)
	gsSetController := gameserversets.NewController(health, gsCounter,
		kubeClient, extClient, agonesClient, agonesInformerFactory, ctlConf.MaxCreationParallelism, ctlConf.MaxDeletionParallelism, ctlConf.MaxGameServerCreationsPerBatch, ctlConf.MaxGameServerDeletionsPerBatch, ctlConf.MaxPodPendingCount)
	fleetController := fleets.NewController(health, kubeClient, extClient, agonesClient, agonesInformerFactory)
	fasController := fleetautoscalers.NewController(health,
		kubeClient, extClient, agonesClient, agonesInformerFactory, gsCounter)

	rs = append(rs,
		gsCounter, gsController, gsSetController, fleetController, fasController)

	runRunner := func(r runner) {
		if err := r.Run(ctx, ctlConf.NumWorkers); err != nil {
			logger.WithError(err).Fatalf("could not start runner! %T", r)
		}
	}

	// Server has to be started earlier because it contains the health check.
	// This allows the controller to not fail health check during install when there is replication
	go runRunner(server)

	whenLeader(ctx, cancel, logger, ctlConf.LeaderElection, kubeClient, ctlConf.PodNamespace, func(ctx context.Context) {
		kubeInformerFactory.Start(ctx.Done())
		agonesInformerFactory.Start(ctx.Done())

		for _, r := range rs {
			go runRunner(r)
		}

		<-ctx.Done()
		logger.Info("Shut down agones controllers")
	})
}

func parseEnvFlags() config {
	exec, err := os.Executable()
	if err != nil {
		logger.WithError(err).Fatal("Could not get executable path")
	}

	base := filepath.Dir(exec)
	viper.SetDefault(sidecarImageFlag, "us-docker.pkg.dev/agones-images/release/agones-sdk:"+pkg.Version)
	viper.SetDefault(sidecarCPURequestFlag, "0")
	viper.SetDefault(sidecarCPULimitFlag, "0")
	viper.SetDefault(sidecarMemoryRequestFlag, "0")
	viper.SetDefault(sidecarMemoryLimitFlag, "0")
	viper.SetDefault(sidecarRunAsUserFlag, "1000")
	viper.SetDefault(sidecarRequestsRateLimitFlag, "500ms")
	viper.SetDefault(pullSidecarFlag, false)
	viper.SetDefault(sdkServerAccountFlag, "agones-sdk")
	viper.SetDefault(certFileFlag, filepath.Join(base, "certs", "server.crt"))
	viper.SetDefault(keyFileFlag, filepath.Join(base, "certs", "server.key"))
	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(stackdriverLabels, "")
	viper.SetDefault(allocationBatchWaitTime, 500*time.Millisecond)
	viper.SetDefault(podNamespace, "agones-system")
	viper.SetDefault(leaderElectionFlag, false)

	viper.SetDefault(projectIDFlag, "")
	viper.SetDefault(numWorkersFlag, 64)
	viper.SetDefault(apiServerSustainedQPSFlag, 100)
	viper.SetDefault(apiServerBurstQPSFlag, 200)
	viper.SetDefault(logDirFlag, "")
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(logSizeLimitMBFlag, 10000) // 10 GB, will be split into 100 MB chunks

	viper.SetDefault(maxCreationParallelismFlag, 16)
	viper.SetDefault(maxGameServerCreationsPerBatchFlag, 64)
	viper.SetDefault(maxDeletionParallelismFlag, 64)
	viper.SetDefault(maxGameServerDeletionsPerBatchFlag, 64)
	viper.SetDefault(maxPodPendingCountFlag, 5000)

	pflag.String(sidecarImageFlag, viper.GetString(sidecarImageFlag), "Flag to overwrite the GameServer sidecar image that is used. Can also use SIDECAR env variable")
	pflag.String(sidecarCPULimitFlag, viper.GetString(sidecarCPULimitFlag), "Flag to overwrite the GameServer sidecar container's cpu limit. Can also use SIDECAR_CPU_LIMIT env variable")
	pflag.String(sidecarCPURequestFlag, viper.GetString(sidecarCPURequestFlag), "Flag to overwrite the GameServer sidecar container's cpu request. Can also use SIDECAR_CPU_REQUEST env variable")
	pflag.String(sidecarMemoryLimitFlag, viper.GetString(sidecarMemoryLimitFlag), "Flag to overwrite the GameServer sidecar container's memory limit. Can also use SIDECAR_MEMORY_LIMIT env variable")
	pflag.String(sidecarMemoryRequestFlag, viper.GetString(sidecarMemoryRequestFlag), "Flag to overwrite the GameServer sidecar container's memory request. Can also use SIDECAR_MEMORY_REQUEST env variable")
	pflag.Int32(sidecarRunAsUserFlag, viper.GetInt32(sidecarRunAsUserFlag), "Flag to indicate the GameServer sidecar container's UID. Can also use SIDECAR_RUN_AS_USER env variable")
	pflag.String(sidecarRequestsRateLimitFlag, viper.GetString(sidecarRequestsRateLimitFlag), "Flag to indicate the GameServer sidecar requests rate limit. Can also use SIDECAR_REQUESTS_RATE_LIMIT env variable")
	pflag.Bool(pullSidecarFlag, viper.GetBool(pullSidecarFlag), "For development purposes, set the sidecar image to have a ImagePullPolicy of Always. Can also use ALWAYS_PULL_SIDECAR env variable")
	pflag.String(sdkServerAccountFlag, viper.GetString(sdkServerAccountFlag), "Overwrite what service account default for GameServer Pods. Defaults to Can also use SDK_SERVICE_ACCOUNT")
	pflag.Int32(minPortFlag, 0, "Required. The minimum port that that a GameServer can be allocated to. Can also use MIN_PORT env variable.")
	pflag.Int32(maxPortFlag, 0, "Required. The maximum port that that a GameServer can be allocated to. Can also use MAX_PORT env variable.")
	pflag.String(additionalPortRangesFlag, viper.GetString(additionalPortRangesFlag), `Optional. Named set of port ranges in JSON object format: '{"game": [5000, 6000]}'. Can Also use ADDITIONAL_PORT_RANGES env variable.`)
	pflag.String(keyFileFlag, viper.GetString(keyFileFlag), "Optional. Path to the key file")
	pflag.String(certFileFlag, viper.GetString(certFileFlag), "Optional. Path to the crt file")
	pflag.String(kubeconfigFlag, viper.GetString(kubeconfigFlag), "Optional. kubeconfig to run the controller out of the cluster. Only use it for debugging as webhook won't works.")
	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.Int32(numWorkersFlag, 64, "Number of controller workers per resource type")
	pflag.Int32(apiServerSustainedQPSFlag, 100, "Maximum sustained queries per second to send to the API server")
	pflag.Int32(apiServerBurstQPSFlag, 200, "Maximum burst queries per second to send to the API server")
	pflag.String(logDirFlag, viper.GetString(logDirFlag), "If set, store logs in a given directory.")
	pflag.Int32(logSizeLimitMBFlag, 1000, "Log file size limit in MB")
	pflag.Int32(maxCreationParallelismFlag, viper.GetInt32(maxCreationParallelismFlag), "Maximum number of parallelizing creation calls in GSS controller")
	pflag.Int32(maxGameServerCreationsPerBatchFlag, viper.GetInt32(maxGameServerCreationsPerBatchFlag), "Maximum number of GameServer creation calls per batch")
	pflag.Int32(maxDeletionParallelismFlag, viper.GetInt32(maxDeletionParallelismFlag), "Maximum number of parallelizing deletion calls in GSS controller")
	pflag.Int32(maxGameServerDeletionsPerBatchFlag, viper.GetInt32(maxGameServerDeletionsPerBatchFlag), "Maximum number of GameServers deletion calls per batch")
	pflag.Int32(maxPodPendingCountFlag, viper.GetInt32(maxPodPendingCountFlag), "Maximum number of pending pods per game server set")
	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Agones Log level")
	pflag.Duration(allocationBatchWaitTime, viper.GetDuration(allocationBatchWaitTime), "Flag to configure the waiting period between allocations batches")
	pflag.String(podNamespace, viper.GetString(podNamespace), "namespace of current pod")
	pflag.Bool(leaderElectionFlag, viper.GetBool(leaderElectionFlag), "Flag to enable/disable leader election for controller pod")
	cloudproduct.BindFlags()
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(sidecarImageFlag))
	runtime.Must(viper.BindEnv(sidecarCPULimitFlag))
	runtime.Must(viper.BindEnv(sidecarCPURequestFlag))
	runtime.Must(viper.BindEnv(sidecarMemoryLimitFlag))
	runtime.Must(viper.BindEnv(sidecarMemoryRequestFlag))
	runtime.Must(viper.BindEnv(sidecarRunAsUserFlag))
	runtime.Must(viper.BindEnv(sidecarRequestsRateLimitFlag))
	runtime.Must(viper.BindEnv(pullSidecarFlag))
	runtime.Must(viper.BindEnv(sdkServerAccountFlag))
	runtime.Must(viper.BindEnv(minPortFlag))
	runtime.Must(viper.BindEnv(maxPortFlag))
	runtime.Must(viper.BindEnv(additionalPortRangesFlag))
	runtime.Must(viper.BindEnv(keyFileFlag))
	runtime.Must(viper.BindEnv(certFileFlag))
	runtime.Must(viper.BindEnv(kubeconfigFlag))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(numWorkersFlag))
	runtime.Must(viper.BindEnv(apiServerSustainedQPSFlag))
	runtime.Must(viper.BindEnv(apiServerBurstQPSFlag))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindEnv(logDirFlag))
	runtime.Must(viper.BindEnv(logSizeLimitMBFlag))
	runtime.Must(viper.BindEnv(maxCreationParallelismFlag))
	runtime.Must(viper.BindEnv(maxGameServerCreationsPerBatchFlag))
	runtime.Must(viper.BindEnv(maxDeletionParallelismFlag))
	runtime.Must(viper.BindEnv(maxGameServerDeletionsPerBatchFlag))
	runtime.Must(viper.BindEnv(maxPodPendingCountFlag))
	runtime.Must(viper.BindEnv(allocationBatchWaitTime))
	runtime.Must(viper.BindEnv(podNamespace))
	runtime.Must(viper.BindEnv(leaderElectionFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(cloudproduct.BindEnv())
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	requestCPU, err := resource.ParseQuantity(viper.GetString(sidecarCPURequestFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", sidecarCPURequestFlag)
	}

	limitCPU, err := resource.ParseQuantity(viper.GetString(sidecarCPULimitFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", sidecarCPULimitFlag)
	}

	requestMemory, err := resource.ParseQuantity(viper.GetString(sidecarMemoryRequestFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", sidecarMemoryRequestFlag)
	}

	limitMemory, err := resource.ParseQuantity(viper.GetString(sidecarMemoryLimitFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", sidecarMemoryLimitFlag)
	}

	requestsRateLimit, err := time.ParseDuration(viper.GetString(sidecarRequestsRateLimitFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", sidecarRequestsRateLimitFlag)
	}

	portRanges, err := parsePortRanges(viper.GetString(additionalPortRangesFlag))
	if err != nil {
		logger.WithError(err).Fatalf("could not parse %s", additionalPortRangesFlag)
	}
	portRanges[agonesv1.DefaultPortRange] = portallocator.PortRange{
		MinPort: int32(viper.GetInt64(minPortFlag)),
		MaxPort: int32(viper.GetInt64(maxPortFlag)),
	}

	return config{
		PortRanges:                     portRanges,
		SidecarImage:                   viper.GetString(sidecarImageFlag),
		SidecarCPURequest:              requestCPU,
		SidecarCPULimit:                limitCPU,
		SidecarMemoryRequest:           requestMemory,
		SidecarMemoryLimit:             limitMemory,
		SidecarRunAsUser:               int(viper.GetInt32(sidecarRunAsUserFlag)),
		SidecarRequestsRateLimit:       requestsRateLimit,
		SdkServiceAccount:              viper.GetString(sdkServerAccountFlag),
		AlwaysPullSidecar:              viper.GetBool(pullSidecarFlag),
		KeyFile:                        viper.GetString(keyFileFlag),
		CertFile:                       viper.GetString(certFileFlag),
		KubeConfig:                     viper.GetString(kubeconfigFlag),
		PrometheusMetrics:              viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:                    viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:                   viper.GetString(projectIDFlag),
		NumWorkers:                     int(viper.GetInt32(numWorkersFlag)),
		APIServerSustainedQPS:          int(viper.GetInt32(apiServerSustainedQPSFlag)),
		APIServerBurstQPS:              int(viper.GetInt32(apiServerBurstQPSFlag)),
		LogDir:                         viper.GetString(logDirFlag),
		LogLevel:                       viper.GetString(logLevelFlag),
		LogSizeLimitMB:                 int(viper.GetInt32(logSizeLimitMBFlag)),
		MaxGameServerCreationsPerBatch: int(viper.GetInt32(maxGameServerCreationsPerBatchFlag)),
		MaxCreationParallelism:         int(viper.GetInt32(maxCreationParallelismFlag)),
		MaxGameServerDeletionsPerBatch: int(viper.GetInt32(maxGameServerDeletionsPerBatchFlag)),
		MaxDeletionParallelism:         int(viper.GetInt32(maxDeletionParallelismFlag)),
		MaxPodPendingCount:             int(viper.GetInt32(maxPodPendingCountFlag)),
		StackdriverLabels:              viper.GetString(stackdriverLabels),
		AllocationBatchWaitTime:        viper.GetDuration(allocationBatchWaitTime),
		PodNamespace:                   viper.GetString(podNamespace),
		LeaderElection:                 viper.GetBool(leaderElectionFlag),
	}
}

func parsePortRanges(s string) (map[string]portallocator.PortRange, error) {
	if s == "" || !runtime.FeatureEnabled(runtime.FeaturePortRanges) {
		return map[string]portallocator.PortRange{}, nil
	}

	prs := map[string][]int32{}
	if err := json.Unmarshal([]byte(s), &prs); err != nil {
		return nil, fmt.Errorf("invlaid additional port range format: %w", err)
	}

	portRanges := map[string]portallocator.PortRange{}
	for k, v := range prs {
		if len(v) != 2 {
			return nil, fmt.Errorf("invalid port range ports for %s: requires both min and max port", k)
		}
		portRanges[k] = portallocator.PortRange{
			MinPort: v[0],
			MaxPort: v[1],
		}
	}
	return portRanges, nil
}

// config stores all required configuration to create a game server controller.
type config struct {
	PortRanges                     map[string]portallocator.PortRange
	SidecarImage                   string
	SidecarCPURequest              resource.Quantity
	SidecarCPULimit                resource.Quantity
	SidecarMemoryRequest           resource.Quantity
	SidecarMemoryLimit             resource.Quantity
	SidecarRunAsUser               int
	SidecarRequestsRateLimit       time.Duration
	SdkServiceAccount              string
	AlwaysPullSidecar              bool
	PrometheusMetrics              bool
	Stackdriver                    bool
	StackdriverLabels              string
	KeyFile                        string
	CertFile                       string
	KubeConfig                     string
	GCPProjectID                   string
	NumWorkers                     int
	APIServerSustainedQPS          int
	APIServerBurstQPS              int
	LogDir                         string
	LogLevel                       string
	LogSizeLimitMB                 int
	MaxGameServerCreationsPerBatch int
	MaxCreationParallelism         int
	MaxGameServerDeletionsPerBatch int
	MaxDeletionParallelism         int
	MaxPodPendingCount             int
	AllocationBatchWaitTime        time.Duration
	PodNamespace                   string
	LeaderElection                 bool
}

// validate ensures the ctlConfig data is valid.
func (c *config) validate() []error {
	validationErrors := make([]error, 0)
	portErrors := validatePorts(c.PortRanges)
	validationErrors = append(validationErrors, portErrors...)
	resourceErrors := validateResource(c.SidecarCPURequest, c.SidecarCPULimit, corev1.ResourceCPU)
	validationErrors = append(validationErrors, resourceErrors...)
	resourceErrors = validateResource(c.SidecarMemoryRequest, c.SidecarMemoryLimit, corev1.ResourceMemory)
	validationErrors = append(validationErrors, resourceErrors...)
	return validationErrors
}

// validateResource validates limit or Memory CPU resources used for containers in pods
// If a GameServer is invalid there will be > 0 values in
// the returned array
//
// Moved from agones.dev/agones/pkg/apis/agones/v1 (#3255)
func validateResource(request resource.Quantity, limit resource.Quantity, resourceName corev1.ResourceName) []error {
	validationErrors := make([]error, 0)
	if !limit.IsZero() && request.Cmp(limit) > 0 {
		validationErrors = append(validationErrors, errors.Errorf("Request must be less than or equal to %s limit", resourceName))
	}
	if request.Cmp(resource.Quantity{}) < 0 {
		validationErrors = append(validationErrors, errors.Errorf("Resource %s request value must be non negative", resourceName))
	}
	if limit.Cmp(resource.Quantity{}) < 0 {
		validationErrors = append(validationErrors, errors.Errorf("Resource %s limit value must be non negative", resourceName))
	}

	return validationErrors
}

func validatePorts(portRanges map[string]portallocator.PortRange) []error {
	validationErrors := make([]error, 0)
	for k, r := range portRanges {
		portErrors := validatePortRange(r.MinPort, r.MaxPort, k)
		validationErrors = append(validationErrors, portErrors...)

	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	keys := make([]string, 0, len(portRanges))
	values := make([]portallocator.PortRange, 0, len(portRanges))
	for k, v := range portRanges {
		keys = append(keys, k)
		values = append(values, v)
	}

	for i, pr := range values {
		for j := i + 1; j < len(values); j++ {
			if overlaps(values[j].MinPort, values[j].MaxPort, pr.MinPort, pr.MaxPort) {
				switch {
				case keys[j] == agonesv1.DefaultPortRange:
					validationErrors = append(validationErrors, errors.Errorf("port range %s overlaps with min/max port", keys[i]))
				case keys[i] == agonesv1.DefaultPortRange:
					validationErrors = append(validationErrors, errors.Errorf("port range %s overlaps with min/max port", keys[j]))
				default:
					validationErrors = append(validationErrors, errors.Errorf("port range %s overlaps with min/max port of range %s", keys[i], keys[j]))
				}
			}
		}
	}
	return validationErrors
}

func validatePortRange(minPort, maxPort int32, rangeName string) []error {
	validationErrors := make([]error, 0)
	var rangeCtx string
	if rangeName != agonesv1.DefaultPortRange {
		rangeCtx = " for port range " + rangeName
	}
	if minPort <= 0 || maxPort <= 0 {
		validationErrors = append(validationErrors, errors.New("min Port and Max Port values are required"+rangeCtx))
	}
	if maxPort < minPort {
		validationErrors = append(validationErrors, errors.New("max Port cannot be set less that the Min Port"+rangeCtx))
	}
	return validationErrors
}

func overlaps(minA, maxA, minB, maxB int32) bool {
	return max(minA, minB) < min(maxA, maxB)
}

type runner interface {
	Run(ctx context.Context, workers int) error
}

func whenLeader(ctx context.Context, cancel context.CancelFunc, logger *logrus.Entry, doLeaderElection bool, kubeClient *kubernetes.Clientset, namespace string, start func(_ context.Context)) {
	if !doLeaderElection {
		start(ctx)
		return
	}

	id := uuid.New().String()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "agones-controller-lock",
			Namespace: namespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	logger.WithField("id", id).Info("Leader Election ID")

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: start,
			OnStoppedLeading: func() {
				logger.WithField("id", id).Info("Leader Lost")
				cancel()
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == id {
					return
				}
				logger.WithField("id", id).Info("New Leader Elected")
			},
		},
	})
}
