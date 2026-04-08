// Copyright 2019 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/allocation/converters"
	pb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/gameservers"
	"agones.dev/agones/pkg/metrics"
	"agones.dev/agones/pkg/processor"
	"agones.dev/agones/pkg/util/fswatch"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"agones.dev/agones/pkg/util/httpserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
)

var (
	podReady bool
	logger   = runtime.NewLoggerWithSource("main")
)

const (
	certDir = "/home/allocator/client-ca/"
	tlsDir  = "/home/allocator/tls/"
)

const (
	httpPortFlag                     = "http-port"
	grpcPortFlag                     = "grpc-port"
	enableStackdriverMetricsFlag     = "stackdriver-exporter"
	enablePrometheusMetricsFlag      = "prometheus-exporter"
	projectIDFlag                    = "gcp-project-id"
	stackdriverLabels                = "stackdriver-labels"
	mTLSDisabledFlag                 = "disable-mtls"
	tlsDisabledFlag                  = "disable-tls"
	remoteAllocationTimeoutFlag      = "remote-allocation-timeout"
	totalRemoteAllocationTimeoutFlag = "total-remote-allocation-timeout"
	apiServerSustainedQPSFlag        = "api-server-qps"
	apiServerBurstQPSFlag            = "api-server-qps-burst"
	logLevelFlag                     = "log-level"
	allocationBatchWaitTime          = "allocation-batch-wait-time"
	readinessShutdownDuration        = "readiness-shutdown-duration"
	httpUnallocatedStatusCode        = "http-unallocated-status-code"
	processorGRPCAddress             = "processor-grpc-address"
	processorGRPCPort                = "processor-grpc-port"
	processorMaxBatchSize            = "processor-max-batch-size"
)

func parseEnvFlags() config {
	viper.SetDefault(httpPortFlag, -1)
	viper.SetDefault(grpcPortFlag, -1)
	viper.SetDefault(apiServerSustainedQPSFlag, 400)
	viper.SetDefault(apiServerBurstQPSFlag, 500)
	viper.SetDefault(enablePrometheusMetricsFlag, true)
	viper.SetDefault(enableStackdriverMetricsFlag, false)
	viper.SetDefault(projectIDFlag, "")
	viper.SetDefault(stackdriverLabels, "")
	viper.SetDefault(mTLSDisabledFlag, false)
	viper.SetDefault(tlsDisabledFlag, false)
	viper.SetDefault(remoteAllocationTimeoutFlag, 10*time.Second)
	viper.SetDefault(totalRemoteAllocationTimeoutFlag, 30*time.Second)
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(allocationBatchWaitTime, 500*time.Millisecond)
	viper.SetDefault(httpUnallocatedStatusCode, http.StatusTooManyRequests)
	viper.SetDefault(processorGRPCAddress, "agones-processor.agones-system.svc.cluster.local")
	viper.SetDefault(processorGRPCPort, 9090)
	viper.SetDefault(processorMaxBatchSize, 100)

	pflag.Int32(httpPortFlag, viper.GetInt32(httpPortFlag), "Port to listen on for REST requests")
	pflag.Int32(grpcPortFlag, viper.GetInt32(grpcPortFlag), "Port to listen on for gRPC requests")
	pflag.Int32(apiServerSustainedQPSFlag, viper.GetInt32(apiServerSustainedQPSFlag), "Maximum sustained queries per second to send to the API server")
	pflag.Int32(apiServerBurstQPSFlag, viper.GetInt32(apiServerBurstQPSFlag), "Maximum burst queries per second to send to the API server")
	pflag.Bool(enablePrometheusMetricsFlag, viper.GetBool(enablePrometheusMetricsFlag), "Flag to activate metrics of Agones. Can also use PROMETHEUS_EXPORTER env variable.")
	pflag.Bool(enableStackdriverMetricsFlag, viper.GetBool(enableStackdriverMetricsFlag), "Flag to activate stackdriver monitoring metrics for Agones. Can also use STACKDRIVER_EXPORTER env variable.")
	pflag.String(projectIDFlag, viper.GetString(projectIDFlag), "GCP ProjectID used for Stackdriver, if not specified ProjectID from Application Default Credentials would be used. Can also use GCP_PROJECT_ID env variable.")
	pflag.String(stackdriverLabels, viper.GetString(stackdriverLabels), "A set of default labels to add to all stackdriver metrics generated. By default metadata are automatically added using Kubernetes API and GCP metadata enpoint.")
	pflag.Bool(mTLSDisabledFlag, viper.GetBool(mTLSDisabledFlag), "Flag to enable/disable mTLS in the allocator.")
	pflag.Bool(tlsDisabledFlag, viper.GetBool(tlsDisabledFlag), "Flag to enable/disable TLS in the allocator.")
	pflag.Duration(remoteAllocationTimeoutFlag, viper.GetDuration(remoteAllocationTimeoutFlag), "Flag to set remote allocation call timeout.")
	pflag.Duration(totalRemoteAllocationTimeoutFlag, viper.GetDuration(totalRemoteAllocationTimeoutFlag), "Flag to set total remote allocation timeout including retries.")
	pflag.String(logLevelFlag, viper.GetString(logLevelFlag), "Agones Log level")
	pflag.Duration(allocationBatchWaitTime, viper.GetDuration(allocationBatchWaitTime), "Flag to configure the waiting period between allocations batches")
	pflag.Duration(readinessShutdownDuration, viper.GetDuration(readinessShutdownDuration), "Time in seconds for SIGTERM/SIGINT handler to sleep for.")
	pflag.Int32(httpUnallocatedStatusCode, viper.GetInt32(httpUnallocatedStatusCode), "HTTP status code to return when no GameServer is available")
	pflag.String(processorGRPCAddress, viper.GetString(processorGRPCAddress), "The gRPC address of the Agones Processor service")
	pflag.Int32(processorGRPCPort, viper.GetInt32(processorGRPCPort), "The gRPC port of the Agones Processor service")
	pflag.Int32(processorMaxBatchSize, viper.GetInt32(processorMaxBatchSize), "The maximum batch size to send to the Agones Processor service")

	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(httpPortFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(apiServerSustainedQPSFlag))
	runtime.Must(viper.BindEnv(apiServerBurstQPSFlag))
	runtime.Must(viper.BindEnv(enablePrometheusMetricsFlag))
	runtime.Must(viper.BindEnv(enableStackdriverMetricsFlag))
	runtime.Must(viper.BindEnv(projectIDFlag))
	runtime.Must(viper.BindEnv(stackdriverLabels))
	runtime.Must(viper.BindEnv(mTLSDisabledFlag))
	runtime.Must(viper.BindEnv(tlsDisabledFlag))
	runtime.Must(viper.BindEnv(remoteAllocationTimeoutFlag))
	runtime.Must(viper.BindEnv(totalRemoteAllocationTimeoutFlag))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindEnv(allocationBatchWaitTime))
	runtime.Must(viper.BindEnv(readinessShutdownDuration))
	runtime.Must(viper.BindEnv(httpUnallocatedStatusCode))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		HTTPPort:                     int(viper.GetInt32(httpPortFlag)),
		GRPCPort:                     int(viper.GetInt32(grpcPortFlag)),
		APIServerSustainedQPS:        int(viper.GetInt32(apiServerSustainedQPSFlag)),
		APIServerBurstQPS:            int(viper.GetInt32(apiServerBurstQPSFlag)),
		PrometheusMetrics:            viper.GetBool(enablePrometheusMetricsFlag),
		Stackdriver:                  viper.GetBool(enableStackdriverMetricsFlag),
		GCPProjectID:                 viper.GetString(projectIDFlag),
		StackdriverLabels:            viper.GetString(stackdriverLabels),
		MTLSDisabled:                 viper.GetBool(mTLSDisabledFlag),
		TLSDisabled:                  viper.GetBool(tlsDisabledFlag),
		LogLevel:                     viper.GetString(logLevelFlag),
		remoteAllocationTimeout:      viper.GetDuration(remoteAllocationTimeoutFlag),
		totalRemoteAllocationTimeout: viper.GetDuration(totalRemoteAllocationTimeoutFlag),
		allocationBatchWaitTime:      viper.GetDuration(allocationBatchWaitTime),
		ReadinessShutdownDuration:    viper.GetDuration(readinessShutdownDuration),
		httpUnallocatedStatusCode:    int(viper.GetInt32(httpUnallocatedStatusCode)),
		processorGRPCAddress:         viper.GetString(processorGRPCAddress),
		processorGRPCPort:            int(viper.GetInt32(processorGRPCPort)),
		processorMaxBatchSize:        int(viper.GetInt32(processorMaxBatchSize)),
	}
}

type config struct {
	GRPCPort                     int
	HTTPPort                     int
	APIServerSustainedQPS        int
	APIServerBurstQPS            int
	TLSDisabled                  bool
	MTLSDisabled                 bool
	PrometheusMetrics            bool
	Stackdriver                  bool
	GCPProjectID                 string
	StackdriverLabels            string
	LogLevel                     string
	totalRemoteAllocationTimeout time.Duration
	remoteAllocationTimeout      time.Duration
	allocationBatchWaitTime      time.Duration
	ReadinessShutdownDuration    time.Duration
	httpUnallocatedStatusCode    int
	processorGRPCAddress         string
	processorGRPCPort            int
	processorMaxBatchSize        int
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from https://github.com/philips/grpc-gateway-example.
func grpcHandlerFunc(grpcServer http.Handler, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a partial recreation of gRPC's internal checks https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		// We switch on HTTP/1.1 or HTTP/2 by checking the ProtoMajor
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
func main() {
	conf := parseEnvFlags()

	logger.WithField("version", pkg.Version).WithField("ctlConf", conf).
		WithField("featureGates", runtime.EncodeFeatures()).
		Info("Starting agones-allocator")

	logger.WithField("logLevel", conf.LogLevel).Info("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(conf.LogLevel))
	if err == nil {
		runtime.SetLevel(level)
	} else {
		logger.WithError(err).Info("Specified wrong Logging.SdkServer. Setting default loglevel - Info")
		runtime.SetLevel(logrus.InfoLevel)
	}

	if !validPort(conf.GRPCPort) && !validPort(conf.HTTPPort) {
		logger.WithField("grpc-port", conf.GRPCPort).WithField("http-port", conf.HTTPPort).Fatal("Must specify a valid gRPC port or an HTTP port for the allocator service")
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
		logger.WithError(err).Fatal("could not create clients")
	}

	listenCtx, cancelListenCtx := context.WithCancel(context.Background())

	// This will test the connection to agones on each readiness probe
	// so if one of the allocator pod can't reach Kubernetes it will be removed
	// from the Kubernetes service.
	podReady = true
	grpcHealth := grpchealth.NewServer() // only used for gRPC, ignored o/w
	health.AddReadinessCheck("allocator-agones-client", func() error {
		if !podReady {
			return errors.New("asked to shut down, failed readiness check")
		}
		_, err := agonesClient.ServerVersion()
		if err != nil {
			return fmt.Errorf("failed to reach Kubernetes: %w", err)
		}
		return nil
	})

	signals.NewSigTermHandler(func() {
		logger.Info("Pod shutdown has been requested, failing readiness check")
		podReady = false
		grpcHealth.Shutdown()
		time.Sleep(conf.ReadinessShutdownDuration)
		cancelListenCtx()
	})

	workerCtx, cancelWorkerCtx := context.WithCancel(context.Background())

	var h *serviceHandler
	if runtime.FeatureEnabled(runtime.FeatureProcessorAllocator) {
		processorConfig := processor.Config{
			ClientID:          os.Getenv("POD_NAME"),
			ProcessorAddress:  fmt.Sprintf("%s:%d", conf.processorGRPCAddress, conf.processorGRPCPort),
			MaxBatchSize:      conf.processorMaxBatchSize,
			AllocationTimeout: 30 * time.Second,
			ReconnectInterval: 5 * time.Second,
		}

		processorClient := processor.NewClient(processorConfig, logger.WithField("component", "processor-client"))

		go func() {
			if err := processorClient.Run(workerCtx); err != nil {
				if workerCtx.Err() != nil {
					logger.WithError(err).Error("Processor client stopped due to context error")
					return
				}
				logger.WithError(err).Error("Processor client failed, initiating graceful shutdown")
			}
		}()

		h = newProcessorServiceHandler(processorClient, conf.MTLSDisabled, conf.TLSDisabled)
	} else {
		grpcUnallocatedStatusCode := grpcCodeFromHTTPStatus(conf.httpUnallocatedStatusCode)
		h = newServiceHandler(workerCtx, kubeClient, agonesClient, health, conf.MTLSDisabled, conf.TLSDisabled, conf.remoteAllocationTimeout, conf.totalRemoteAllocationTimeout, conf.allocationBatchWaitTime, grpcUnallocatedStatusCode)
	}

	if !h.tlsDisabled {
		cancelTLS, err := fswatch.Watch(logger, tlsDir, time.Second, func() {
			tlsCert, err := readTLSCert()
			if err != nil {
				logger.WithError(err).Error("could not load TLS certs; keeping old one")
				return
			}
			h.tlsMutex.Lock()
			defer h.tlsMutex.Unlock()
			h.tlsCert = tlsCert
			logger.Info("TLS certs updated")
		})
		if err != nil {
			logger.WithError(err).Fatal("could not create watcher for TLS certs")
		}
		defer cancelTLS()

		if !h.mTLSDisabled {
			cancelCert, err := fswatch.Watch(logger, certDir, time.Second, func() {
				h.certMutex.Lock()
				defer h.certMutex.Unlock()
				caCertPool, err := getCACertPool(certDir)
				if err != nil {
					logger.WithError(err).Error("could not load CA certs; keeping old ones")
					return
				}
				h.caCertPool = caCertPool
				logger.Info("CA certs updated")
			})
			if err != nil {
				logger.WithError(err).Fatal("could not create watcher for CA certs")
			}
			defer cancelCert()
		}
	}

	// If grpc and http use the same port then use a mux.
	if conf.GRPCPort == conf.HTTPPort {
		runMux(listenCtx, workerCtx, h, grpcHealth, conf.HTTPPort)
	} else {
		// Otherwise, run each on a dedicated port.
		if validPort(conf.HTTPPort) {
			runREST(listenCtx, workerCtx, h, conf.HTTPPort)
		}
		if validPort(conf.GRPCPort) {
			runGRPC(listenCtx, h, grpcHealth, conf.GRPCPort)
		}
	}

	// Finally listen on 8080 (http), used to serve /live and /ready handlers for Kubernetes probes.
	healthserver.Handle("/", health)
	go func() { _ = healthserver.Run(listenCtx, 0) }()

	// TODO: This is messy. Contexts are the wrong way to handle this - we should be using shutdown,
	// and a cascading graceful shutdown instead of multiple contexts and sleeps.
	<-listenCtx.Done()
	logger.Infof("Listen context cancelled")
	time.Sleep(5 * time.Second)
	cancelWorkerCtx()
	logger.Infof("Worker context cancelled")
	time.Sleep(1 * time.Second)
	logger.Info("Shut down allocator")
}

func validPort(port int) bool {
	const maxPort = 65535
	return port >= 0 && port < maxPort
}

func runMux(listenCtx context.Context, workerCtx context.Context, h *serviceHandler, grpcHealth *grpchealth.Server, httpPort int) {
	logger.Infof("Running the mux handler on port %d", httpPort)
	grpcServer := grpc.NewServer(h.getMuxServerOptions()...)
	pb.RegisterAllocationServiceServer(grpcServer, h)
	grpc_health_v1.RegisterHealthServer(grpcServer, grpcHealth)

	mux := runtime.NewServerMux()
	if err := pb.RegisterAllocationServiceHandlerServer(context.Background(), mux, h); err != nil {
		panic(err)
	}

	runHTTP(listenCtx, workerCtx, h, httpPort, grpcHandlerFunc(grpcServer, mux))
}

func runREST(listenCtx context.Context, workerCtx context.Context, h *serviceHandler, httpPort int) {
	logger.WithField("port", httpPort).Info("Running the rest handler")
	mux := runtime.NewServerMux()
	if err := pb.RegisterAllocationServiceHandlerServer(context.Background(), mux, h); err != nil {
		panic(err)
	}
	runHTTP(listenCtx, workerCtx, h, httpPort, mux)
}

func runHTTP(listenCtx context.Context, workerCtx context.Context, h *serviceHandler, httpPort int, handler http.Handler) {
	cfg := &tls.Config{}
	if !h.tlsDisabled {
		cfg.GetCertificate = h.getTLSCert
	}
	if !h.mTLSDisabled {
		cfg.ClientAuth = tls.RequireAnyClientCert
		cfg.VerifyPeerCertificate = h.verifyClientCertificate
	}

	// Create a Server instance to listen on the http port with the TLS config.
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", httpPort),
		TLSConfig: cfg,
		Handler:   handler,
	}

	go func() {
		go func() {
			<-listenCtx.Done()
			_ = server.Shutdown(workerCtx)
		}()

		var err error
		if !h.tlsDisabled {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}

		if err == http.ErrServerClosed {
			logger.WithError(err).Info("HTTP/HTTPS server closed")
			os.Exit(0)
		}
		logger.WithError(err).Fatal("Unable to start HTTP/HTTPS listener")
		os.Exit(1)

	}()
}

func runGRPC(ctx context.Context, h *serviceHandler, grpcHealth *grpchealth.Server, grpcPort int) {
	logger.WithField("port", grpcPort).Info("Running the grpc handler on port")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		logger.WithError(err).Fatalf("failed to listen on TCP port %d", grpcPort)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(h.getGRPCServerOptions()...)
	pb.RegisterAllocationServiceServer(grpcServer, h)
	grpc_health_v1.RegisterHealthServer(grpcServer, grpcHealth)

	go func() {
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop()
		}()

		err := grpcServer.Serve(listener)
		if err != nil {
			logger.WithError(err).Fatal("allocation service crashed")
			os.Exit(1)
		}
		logger.Info("allocation server closed")
		os.Exit(0)

	}()
}

func newProcessorServiceHandler(processorClient processor.Client, mTLSDisabled, tlsDisabled bool) *serviceHandler {
	h := serviceHandler{
		mTLSDisabled:    mTLSDisabled,
		tlsDisabled:     tlsDisabled,
		processorClient: processorClient,
	}

	if !h.tlsDisabled {
		tlsCert, err := readTLSCert()
		if err != nil {
			logger.WithError(err).Fatal("could not load TLS certs.")
		}
		h.tlsMutex.Lock()
		h.tlsCert = tlsCert
		h.tlsMutex.Unlock()

		if !h.mTLSDisabled {
			caCertPool, err := getCACertPool(certDir)
			if err != nil {
				logger.WithError(err).Fatal("could not load CA certs.")
			}
			h.certMutex.Lock()
			h.caCertPool = caCertPool
			h.certMutex.Unlock()
		}
	}

	return &h
}

func newServiceHandler(ctx context.Context, kubeClient kubernetes.Interface, agonesClient versioned.Interface, health healthcheck.Handler, mTLSDisabled bool, tlsDisabled bool, remoteAllocationTimeout time.Duration, totalRemoteAllocationTimeout time.Duration, allocationBatchWaitTime time.Duration, grpcUnallocatedStatusCode codes.Code) *serviceHandler {
	defaultResync := 30 * time.Second
	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, defaultResync)
	kubeInformerFactory := informers.NewSharedInformerFactory(kubeClient, defaultResync)
	gsCounter := gameservers.NewPerNodeCounter(kubeInformerFactory, agonesInformerFactory)

	allocator := gameserverallocations.NewAllocator(
		agonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		kubeInformerFactory.Core().V1().Secrets(),
		agonesClient.AgonesV1(),
		kubeClient,
		gameserverallocations.NewAllocationCache(agonesInformerFactory.Agones().V1().GameServers(), gsCounter, health),
		remoteAllocationTimeout,
		totalRemoteAllocationTimeout,
		allocationBatchWaitTime)

	h := serviceHandler{
		allocationCallback: func(allocationCtx context.Context, gsa *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return allocator.Allocate(allocationCtx, gsa)
		},
		mTLSDisabled:              mTLSDisabled,
		tlsDisabled:               tlsDisabled,
		grpcUnallocatedStatusCode: grpcUnallocatedStatusCode,
	}

	kubeInformerFactory.Start(ctx.Done())
	agonesInformerFactory.Start(ctx.Done())
	if err := allocator.Run(ctx); err != nil {
		logger.WithError(err).Fatal("starting allocator failed.")
	}

	if !h.tlsDisabled {
		tlsCert, err := readTLSCert()
		if err != nil {
			logger.WithError(err).Fatal("could not load TLS certs.")
		}
		h.tlsMutex.Lock()
		h.tlsCert = tlsCert
		h.tlsMutex.Unlock()

		if !h.mTLSDisabled {
			caCertPool, err := getCACertPool(certDir)
			if err != nil {
				logger.WithError(err).Fatal("could not load CA certs.")
			}
			h.certMutex.Lock()
			h.caCertPool = caCertPool
			h.certMutex.Unlock()
		}
	}

	return &h
}

func readTLSCert() (*tls.Certificate, error) {
	tlsCert, err := tls.LoadX509KeyPair(tlsDir+"tls.crt", tlsDir+"tls.key")
	if err != nil {
		return nil, err
	}
	return &tlsCert, nil
}

// getMuxServerOptions returns a list of GRPC server option to use when
// serving gRPC and REST over an HTTP multiplexer.
// Current options are opencensus stats handler.
func (h *serviceHandler) getMuxServerOptions() []grpc.ServerOption {
	// Add options for  OpenCensus stats handler to enable stats and tracing.
	// The keepalive options are useful for efficiency purposes (keeping a single connection alive
	// instead of constantly recreating connections), when placing the Agones allocator behind load balancers.
	return []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           10 * time.Minute,
		}),
	}
}

// getGRPCServerOptions returns a list of GRPC server options to use when
// only serving gRPC requests.
// Current options are TLS certs and opencensus stats handler.
func (h *serviceHandler) getGRPCServerOptions() []grpc.ServerOption {
	// Add options for  OpenCensus stats handler to enable stats and tracing.
	// The keepalive options are useful for efficiency purposes (keeping a single connection alive
	// instead of constantly recreating connections), when placing the Agones allocator behind load balancers.
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           10 * time.Minute,
		}),
	}
	if h.tlsDisabled {
		return opts
	}

	cfg := &tls.Config{
		GetCertificate: h.getTLSCert,
	}

	if !h.mTLSDisabled {
		cfg.ClientAuth = tls.RequireAnyClientCert
		cfg.VerifyPeerCertificate = h.verifyClientCertificate
	}

	return append([]grpc.ServerOption{grpc.Creds(credentials.NewTLS(cfg))}, opts...)
}

func (h *serviceHandler) getTLSCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	h.tlsMutex.RLock()
	defer h.tlsMutex.RUnlock()
	return h.tlsCert, nil
}

// verifyClientCertificate verifies that the client certificate is accepted
// This method is used as GetConfigForClient is cross lang incompatible.
func (h *serviceHandler) verifyClientCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	opts := x509.VerifyOptions{
		Roots:         h.caCertPool,
		CurrentTime:   time.Now(),
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, rawCert := range rawCerts[1:] {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			logger.WithError(err).Warning("cannot parse intermediate certificate")
			return errors.New("bad intermediate certificate: " + err.Error())
		}
		opts.Intermediates.AddCert(cert)
	}

	c, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		logger.WithError(err).Warning("cannot parse client certificate")
		return errors.New("bad client certificate: " + err.Error())
	}

	h.certMutex.RLock()
	defer h.certMutex.RUnlock()
	_, err = c.Verify(opts)
	if err != nil {
		logger.WithError(err).Warning("failed to verify client certificate")
		return errors.New("failed to verify client certificate: " + err.Error())
	}
	return nil
}

// Set up our client which we will use to call the API
func getClients(ctlConfig config) (*kubernetes.Clientset, *versioned.Clientset, error) {
	// Create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, nil, errors.New("Could not create in cluster config")
	}

	config.QPS = float32(ctlConfig.APIServerSustainedQPS)
	config.Burst = ctlConfig.APIServerBurstQPS

	// Access to the Agones resources through the Agones Clientset
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the kubernetes api clientset")
	}

	// Access to the Agones resources through the Agones Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.New("Could not create the agones api clientset")
	}
	return kubeClient, agonesClient, nil
}

func getCACertPool(path string) (*x509.CertPool, error) {
	// Add all certificates under client-certs path because there could be multiple clusters
	// and all client certs should be added.
	caCertPool := x509.NewCertPool()
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading certs from dir %s: %s", path, err.Error())
	}

	for _, dirEntry := range dirEntries {
		if !strings.HasSuffix(dirEntry.Name(), ".crt") && !strings.HasSuffix(dirEntry.Name(), ".pem") {
			continue
		}
		certFile := filepath.Join(path, dirEntry.Name())
		caCert, err := os.ReadFile(certFile)
		if err != nil {
			logger.Errorf("CA cert is not readable or missing: %s", err.Error())
			continue
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			logger.Errorf("client cert %s cannot be installed", certFile)
			continue
		}
		logger.Infof("client cert %s is installed", certFile)
	}

	return caCertPool, nil
}

type serviceHandler struct {
	allocationCallback func(context.Context, *allocationv1.GameServerAllocation) (k8sruntime.Object, error)

	certMutex  sync.RWMutex
	caCertPool *x509.CertPool

	tlsMutex sync.RWMutex
	tlsCert  *tls.Certificate

	mTLSDisabled bool
	tlsDisabled  bool

	grpcUnallocatedStatusCode codes.Code

	processorClient processor.Client
}

// Allocate implements the Allocate gRPC method definition
func (h *serviceHandler) Allocate(ctx context.Context, in *pb.AllocationRequest) (*pb.AllocationResponse, error) {
	logger.WithField("request", in).Infof("allocation request received.")

	gsa := converters.ConvertAllocationRequestToGSA(in)
	gsa.ApplyDefaults()

	if runtime.FeatureEnabled(runtime.FeatureProcessorAllocator) {
		req := converters.ConvertGSAToAllocationRequest(gsa)

		resp, err := h.processorClient.Allocate(ctx, req)
		if err != nil {
			logger.WithField("gsa", gsa).WithError(err).Error("allocation failed")
			return nil, err
		}

		allocatedGsa := converters.ConvertAllocationResponseToGSA(resp, resp.Source)
		response, err := converters.ConvertGSAToAllocationResponse(allocatedGsa, h.grpcUnallocatedStatusCode)
		logger.WithField("response", response).WithError(err).Info("allocation response is being sent")

		return response, err
	}

	resultObj, err := h.allocationCallback(ctx, gsa)
	if err != nil {
		logger.WithField("gsa", gsa).WithError(err).Error("allocation failed")
		return nil, err
	}

	if s, ok := resultObj.(*metav1.Status); ok {
		return nil, status.Errorf(codes.Code(s.Code), s.Message, resultObj)
	}

	allocatedGsa, ok := resultObj.(*allocationv1.GameServerAllocation)
	if !ok {
		logger.Errorf("internal server error - Bad GSA format %v", resultObj)
		return nil, status.Errorf(codes.Internal, "internal server error- Bad GSA format %v", resultObj)
	}
	response, err := converters.ConvertGSAToAllocationResponse(allocatedGsa, h.grpcUnallocatedStatusCode)
	logger.WithField("response", response).WithError(err).Infof("allocation response is being sent")

	return response, err
}

// grpcCodeFromHTTPStatus converts an HTTP status code to the corresponding gRPC status code.
func grpcCodeFromHTTPStatus(httpUnallocatedStatusCode int) codes.Code {
	switch httpUnallocatedStatusCode {
	case http.StatusOK:
		return codes.OK
	case 499:
		return codes.Canceled
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		logger.WithField("httpStatusCode", httpUnallocatedStatusCode).Warnf("received unknown http status code, defaulting to codes.ResourceExhausted / 429")
		return codes.ResourceExhausted
	}
}
