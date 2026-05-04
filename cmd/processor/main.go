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

// Processor
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	allocationpb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/processor"
	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"

	"github.com/google/uuid"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	leaderelection "k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	allocationBatchWaitTime          = "allocation-batch-wait-time"
	apiServerBurstQPSFlag            = "api-server-qps-burst"
	apiServerSustainedQPSFlag        = "api-server-qps"
	defaultResync                    = 30 * time.Second
	enablePrometheusMetricsFlag      = "prometheus-exporter"
	enableStackdriverMetricsFlag     = "stackdriver-exporter"
	grpcPortFlag                     = "grpc-port"
	httpUnallocatedStatusCode        = "http-unallocated-status-code"
	leaderElectionFlag               = "leader-election"
	leaseDurationFlag                = "lease-duration"
	logLevelFlag                     = "log-level"
	podNamespace                     = "pod-namespace"
	projectIDFlag                    = "gcp-project-id"
	pullIntervalFlag                 = "pull-interval"
	remoteAllocationTimeoutFlag      = "remote-allocation-timeout"
	renewDeadlineFlag                = "renew-deadline"
	retryPeriodFlag                  = "retry-period"
	stackdriverLabels                = "stackdriver-labels"
	totalRemoteAllocationTimeoutFlag = "total-remote-allocation-timeout"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

type processorConfig struct {
	GCPProjectID                 string
	LogLevel                     string
	PodNamespace                 string
	StackdriverLabels            string
	APIServerBurstQPS            int
	APIServerSustainedQPS        int
	GRPCPort                     int
	HTTPUnallocatedStatusCode    int
	LeaderElection               bool
	PrometheusMetrics            bool
	Stackdriver                  bool
	AllocationBatchWaitTime      time.Duration
	LeaseDuration                time.Duration
	PullInterval                 time.Duration
	RenewDeadline                time.Duration
	RemoteAllocationTimeout      time.Duration
	RetryPeriod                  time.Duration
	TotalRemoteAllocationTimeout time.Duration
}

func parseEnvFlags() processorConfig {
	viper.SetDefault(allocationBatchWaitTime, 50*time.Millisecond)
	viper.SetDefault(apiServerBurstQPSFlag, 500)
	viper.SetDefault(apiServerSustainedQPSFlag, 400)
	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(grpcPortFlag, 9090)
	viper.SetDefault(httpUnallocatedStatusCode, http.StatusTooManyRequests)
	viper.SetDefault(leaderElectionFlag, false)
	viper.SetDefault(leaseDurationFlag, 15*time.Second)
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(podNamespace, "")
	viper.SetDefault(projectIDFlag, "")
	viper.SetDefault(pullIntervalFlag, 200*time.Millisecond)
	viper.SetDefault(remoteAllocationTimeoutFlag, 10*time.Second)
	viper.SetDefault(renewDeadlineFlag, 10*time.Second)
	viper.SetDefault(retryPeriodFlag, 2*time.Second)
	viper.SetDefault(stackdriverLabels, "")
	viper.SetDefault(totalRemoteAllocationTimeoutFlag, 30*time.Second)

	pflag.Duration(allocationBatchWaitTime, viper.GetDuration(allocationBatchWaitTime), "Flag to configure the waiting period between allocations batches")
	pflag.Int32(apiServerBurstQPSFlag, viper.GetInt32(apiServerBurstQPSFlag), "Maximum burst queries per second to send to the API server")
	pflag.Int32(apiServerSustainedQPSFlag, viper.GetInt32(apiServerSustainedQPSFlag), "Maximum sustained queries per second to send to the API server")
	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "Port to listen on for gRPC requests")
	pflag.Int32(httpUnallocatedStatusCode, viper.GetInt32(httpUnallocatedStatusCode), "HTTP status code to return when no GameServer is available")
	pflag.Bool(leaderElectionFlag, viper.GetBool(leaderElectionFlag), "Enable leader election")
	pflag.Duration(leaseDurationFlag, viper.GetDuration(leaseDurationFlag), "Leader election lease duration")
	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Log level")
	pflag.String(podNamespace, viper.GetString(podNamespace), "Pod namespace")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.Duration(pullIntervalFlag, viper.GetDuration(pullIntervalFlag), "Interval between pull requests sent to processor clients")
	pflag.Duration(remoteAllocationTimeoutFlag, viper.GetDuration(remoteAllocationTimeoutFlag), "Flag to set remote allocation call timeout.")
	pflag.Duration(renewDeadlineFlag, viper.GetDuration(renewDeadlineFlag), "Leader election renew deadline")
	pflag.Duration(retryPeriodFlag, viper.GetDuration(retryPeriodFlag), "Leader election retry period")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	pflag.Duration(totalRemoteAllocationTimeoutFlag, viper.GetDuration(totalRemoteAllocationTimeoutFlag), "Flag to set total remote allocation timeout including retries.")
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(allocationBatchWaitTime))
	runtime.Must(viper.BindEnv(apiServerBurstQPSFlag))
	runtime.Must(viper.BindEnv(apiServerSustainedQPSFlag))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(httpUnallocatedStatusCode))
	runtime.Must(viper.BindEnv(leaderElectionFlag))
	runtime.Must(viper.BindEnv(leaseDurationFlag))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindEnv(podNamespace))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(pullIntervalFlag))
	runtime.Must(viper.BindEnv(remoteAllocationTimeoutFlag))
	runtime.Must(viper.BindEnv(renewDeadlineFlag))
	runtime.Must(viper.BindEnv(retryPeriodFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindEnv(totalRemoteAllocationTimeoutFlag))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return processorConfig{
		AllocationBatchWaitTime:      viper.GetDuration(allocationBatchWaitTime),
		APIServerBurstQPS:            int(viper.GetInt32(apiServerBurstQPSFlag)),
		APIServerSustainedQPS:        int(viper.GetInt32(apiServerSustainedQPSFlag)),
		GCPProjectID:                 viper.GetString(projectIDFlag),
		GRPCPort:                     int(viper.GetInt32(grpcPortFlag)),
		HTTPUnallocatedStatusCode:    int(viper.GetInt32(httpUnallocatedStatusCode)),
		LeaderElection:               viper.GetBool(leaderElectionFlag),
		LeaseDuration:                viper.GetDuration(leaseDurationFlag),
		LogLevel:                     viper.GetString(logLevelFlag),
		PodNamespace:                 viper.GetString(podNamespace),
		PrometheusMetrics:            viper.GetBool(enablePrometheusMetricsFlag),
		PullInterval:                 viper.GetDuration(pullIntervalFlag),
		RenewDeadline:                viper.GetDuration(renewDeadlineFlag),
		RemoteAllocationTimeout:      viper.GetDuration(remoteAllocationTimeoutFlag),
		RetryPeriod:                  viper.GetDuration(retryPeriodFlag),
		Stackdriver:                  viper.GetBool(enableStackdriverMetricsFlag),
		StackdriverLabels:            viper.GetString(stackdriverLabels),
		TotalRemoteAllocationTimeout: viper.GetDuration(totalRemoteAllocationTimeoutFlag),
	}
}

func main() {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	conf := parseEnvFlags()

	logger.WithField("version", pkg.Version).WithField("processorConf", conf).
		WithField("featureGates", runtime.EncodeFeatures()).
		Info("Starting agones-processor")

	logger.WithField("logLevel", conf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(conf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	healthserver := &httpserver.Server{Logger: logger}
	var health healthcheck.Handler

	metricsConf := metrics.Config{
		Stackdriver:       conf.Stackdriver,
		PrometheusMetrics: conf.PrometheusMetrics,
		GCPProjectID:      conf.GCPProjectID,
		StackdriverLabels: conf.StackdriverLabels,
	}
	health, closer := metrics.SetupMetrics(metricsConf, healthserver)
	defer closer()

	metrics.SetReportingPeriod(conf.PrometheusMetrics, conf.Stackdriver)

	kubeClient, agonesClient, err := getClients(conf)
	if err != nil {
		logger.WithError(err).Fatal("Could not create clients")
	}

	grpcUnallocatedStatusCode := processor.GRPCCodeFromHTTPStatus(conf.HTTPUnallocatedStatusCode)
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)
	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		agonesClient.AgonesV1(),
		kubeClient,
		gameserverallocations.NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), gsCounter, health),
		conf.RemoteAllocationTimeout,
		conf.TotalRemoteAllocationTimeout,
		conf.AllocationBatchWaitTime)
	kubeInformerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())
	if err := allocator.Run(ctx); err != nil {
		logger.WithError(err).Fatal("Starting allocator failed.")
	}

	processorService := processor.NewServiceHandler(allocator, conf.PullInterval, grpcUnallocatedStatusCode)

	grpcHealth := grpchealth.NewServer()
	grpcHealth.SetServingStatus("processor", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	runGRPC(ctx, processorService, grpcHealth, conf.GRPCPort)

	go func() {
		healthserver.Handle("/", health)
		_ = healthserver.Run(ctx, 0)
	}()

	signals.NewSigTermHandler(func() {
		logger.Info("Pod shutdown has been requested, failing readiness check")
		grpcHealth.Shutdown()
		cancelCtx()
	})

	whenLeader(ctx, cancelCtx, logger, conf, kubeClient, grpcHealth, func(_ context.Context) {
		logger.Info("Starting processor work as leader")
		grpcHealth.SetServingStatus("processor", grpc_health_v1.HealthCheckResponse_SERVING)
		processorService.StartPullRequestTicker(ctx)
	})

	logger.Info("Processor exited gracefully.")
}

func whenLeader(ctx context.Context, cancel context.CancelFunc, logger *logrus.Entry,
	conf processorConfig, kubeClient *kubernetes.Clientset, grpcHealth *grpchealth.Server, start func(_ context.Context)) {
	logger.WithField("leaderElectionEnabled", conf.LeaderElection).Info("Leader election configuration")

	if !conf.LeaderElection {
		start(ctx)
		<-ctx.Done()

		return
	}

	id := uuid.New().String()
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "agones-processor-lock",
			Namespace: conf.PodNamespace,
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
		LeaseDuration:   conf.LeaseDuration,
		RenewDeadline:   conf.RenewDeadline,
		RetryPeriod:     conf.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: start,
			OnStoppedLeading: func() {
				logger.WithField("id", id).Info("Leader Lost")
				grpcHealth.SetServingStatus("processor", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
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

func runGRPC(ctx context.Context, h *processor.Handler, grpcHealth *grpchealth.Server, grpcPort int) {
	logger.WithField("port", grpcPort).Info("Running the grpc handler on port")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.WithError(err).Fatalf("Failed to listen on TCP port %d", grpcPort)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(h.GetGRPCServerOptions()...)
	allocationpb.RegisterProcessorServer(grpcServer, h)
	grpc_health_v1.RegisterHealthServer(grpcServer, grpcHealth)

	go func() {
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()

		err := grpcServer.Serve(listener)
		if err != nil {
			logger.WithError(err).Fatal("Allocation service crashed")
		}
		logger.Info("Allocation server closed")
		os.Exit(0)

	}()
}

// Set up our client which we will use to call the API
func getClients(ctlConfig processorConfig) (*kubernetes.Clientset, *versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not create in cluster config")
	}

	config.QPS = float32(ctlConfig.APIServerSustainedQPS)
	config.Burst = ctlConfig.APIServerBurstQPS

	// Access to the Agones resources through the Agones Clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not create the kubernetes api clientset")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Could not create the agones api clientset")
	}
	return kubeClient, agonesClient, nil
}
