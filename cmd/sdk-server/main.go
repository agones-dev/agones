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

// sidecar for the game server that the sdk connects to
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/sdk"
	sdkalpha "agones.dev/agones/pkg/sdk/alpha"
	sdkbeta "agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/sdkserver"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
)

const (
	defaultGRPCPort   = 9357
	defaultHTTPPort   = 9358
	defaultHealthPort = 8080

	// Flags (that can also be env vars)
	gameServerNameFlag      = "gameserver-name"
	podNamespaceFlag        = "pod-namespace"
	localFlag               = "local"
	fileFlag                = "file"
	testFlag                = "test"
	testSdkNameFlag         = "sdk-name"
	kubeconfigFlag          = "kubeconfig"
	gracefulTerminationFlag = "graceful-termination"
	addressFlag             = "address"
	delayFlag               = "delay"
	timeoutFlag             = "timeout"
	grpcPortFlag            = "grpc-port"
	httpPortFlag            = "http-port"
	healthPortFlag          = "health-port"
	logLevelFlag            = "log-level"
	requestRateLimitFlag    = "request-rate-limit"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

func main() {
	ctlConf := parseEnvFlags()
	logLevel, err := logrus.ParseLevel(ctlConf.LogLevel)
	if err != nil {
		logrus.WithError(err).Warn("Invalid LOG_LEVEL value. Defaulting to 'info'.")
		logLevel = logrus.InfoLevel
	}
	logger.Logger.SetLevel(logLevel)
	logger.WithField("version", pkg.Version).WithField("featureGates", runtime.EncodeFeatures()).
		WithField("ctlConf", ctlConf).Info("Starting sdk sidecar")

	if ctlConf.Delay > 0 {
		logger.Infof("Waiting %d seconds before starting", ctlConf.Delay)
		time.Sleep(time.Duration(ctlConf.Delay) * time.Second)
	}

	ctx, _ := signals.NewSigKillContext()

	grpcServer := grpc.NewServer()
	// don't graceful stop, because if we get a SIGKILL signal
	// then the gameserver is being shut down, and we no longer
	// care about running RPC calls.
	defer grpcServer.Stop()

	mux := runtime.NewServerMux()
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", ctlConf.Address, ctlConf.HTTPPort),
		Handler: wsproxy.WebsocketProxy(healthCheckWrapper(mux)),
	}
	defer httpServer.Close() // nolint: errcheck

	switch {
	case ctlConf.IsLocal:
		cancel, err := registerLocal(grpcServer, ctlConf)
		if err != nil {
			logger.WithError(err).Fatal("Could not start local SDK server")
		}
		defer cancel()

		if ctlConf.Timeout != 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(ctlConf.Timeout)*time.Second)
			defer cancel()
		}
	case ctlConf.Test != "":
		cancel, err := registerTestSdkServer(grpcServer, ctlConf)
		if err != nil {
			logger.WithError(err).Fatal("Could not start test SDK server")
		}
		defer cancel()

		if ctlConf.Timeout != 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(ctlConf.Timeout)*time.Second)
			defer cancel()
		}
	default:
		var config *rest.Config
		// if the kubeconfig fails InClusterBuildConfig will try in cluster config
		config, err := runtime.InClusterBuildConfig(logger, ctlConf.KubeConfig)
		if err != nil {
			logger.WithError(err).Fatal("Could not create in cluster config")
		}

		var kubeClient *kubernetes.Clientset
		kubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatal("Could not create the kubernetes clientset")
		}

		var agonesClient *versioned.Clientset
		agonesClient, err = versioned.NewForConfig(config)
		if err != nil {
			logger.WithError(err).Fatalf("Could not create the agones api clientset")
		}

		var s *sdkserver.SDKServer
		s, err = sdkserver.NewSDKServer(ctlConf.GameServerName, ctlConf.PodNamespace,
			kubeClient, agonesClient, logLevel, ctlConf.HealthPort, ctlConf.RequestsRateLimit)
		if err != nil {
			logger.WithError(err).Fatalf("Could not start sidecar")
		}
		// wait for networking prior to replacing context, otherwise we'll
		// end up waiting the full grace period if it fails.
		if err := s.WaitForConnection(ctx); err != nil {
			logger.WithError(err).Fatalf("Sidecar networking failure")
		}

		// if sidecar init container, we no longer need to wait for Shutdown as the sdkserver has an independent lifecycle
		// from the main pod containers.
		if ctlConf.GracefulTermination && !runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
			ctx = s.NewSDKServerContext(ctx)
		}
		go func() {
			err := s.Run(ctx)
			if err != nil {
				logger.WithError(err).Fatalf("Could not run sidecar")
			}
		}()
		sdk.RegisterSDKServer(grpcServer, s)
		sdkalpha.RegisterSDKServer(grpcServer, s)
		sdkbeta.RegisterSDKServer(grpcServer, s)
	}

	grpcEndpoint := fmt.Sprintf("%s:%d", ctlConf.Address, ctlConf.GRPCPort)
	go runGrpc(grpcServer, grpcEndpoint)
	go runGateway(ctx, grpcEndpoint, mux, httpServer)

	<-ctx.Done()
	logger.Info("Shutting down SDK server")
}

// registerLocal registers the local SDK servers, and returns a cancel func that
// closes all the SDK implementations
func registerLocal(grpcServer *grpc.Server, ctlConf config) (func(), error) {
	filePath := ""
	if ctlConf.LocalFile != "" {
		var err error
		filePath, err = filepath.Abs(ctlConf.LocalFile)
		if err != nil {
			return nil, err
		}

		if _, err = os.Stat(filePath); os.IsNotExist(err) {
			return nil, errors.Errorf("Could not find file: %s", filePath)
		}
	}

	s, err := sdkserver.NewLocalSDKServer(filePath, ctlConf.TestSdkName)
	if err != nil {
		return nil, err
	}

	sdk.RegisterSDKServer(grpcServer, s)
	sdkalpha.RegisterSDKServer(grpcServer, s)
	sdkbeta.RegisterSDKServer(grpcServer, s)

	return func() {
		s.Close()
	}, err
}

// registerLocal registers the local test SDK servers, and returns a cancel func that
// closes all the SDK implementations
func registerTestSdkServer(grpcServer *grpc.Server, ctlConf config) (func(), error) {
	s, err := sdkserver.NewLocalSDKServer("", "")
	if err != nil {
		return nil, err
	}

	s.SetTestMode(true)
	s.GenerateUID()
	expectedFuncs := strings.Split(ctlConf.Test, ",")
	s.SetExpectedSequence(expectedFuncs)
	s.SetSdkName(ctlConf.TestSdkName)

	sdk.RegisterSDKServer(grpcServer, s)
	sdkalpha.RegisterSDKServer(grpcServer, s)
	sdkbeta.RegisterSDKServer(grpcServer, s)
	return func() {
		s.Close()
	}, err
}

// runGrpc runs the grpc service
func runGrpc(grpcServer *grpc.Server, grpcEndpoint string) {
	lis, err := net.Listen("tcp", grpcEndpoint)
	if err != nil {
		logger.WithField("grpcEndpoint", grpcEndpoint).Fatal("Could not listen on grpc endpoint")
	}

	logger.WithField("grpcEndpoint", grpcEndpoint).Info("Starting SDKServer grpc service...")
	if err := grpcServer.Serve(lis); err != nil {
		logger.WithError(err).Fatal("Could not serve grpc server")
	}
}

// runGateway runs the grpc-gateway
func runGateway(ctx context.Context, grpcEndpoint string, mux *gwruntime.ServeMux, httpServer *http.Server) {
	// nolint: staticcheck
	conn, err := grpc.NewClient(grpcEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.WithError(err).Fatal("Could not dial grpc server...")
	}

	if err := sdk.RegisterSDKHandler(ctx, mux, conn); err != nil {
		logger.WithError(err).Fatal("Could not register sdk grpc-gateway")
	}

	if err := sdkalpha.RegisterSDKHandler(ctx, mux, conn); err != nil {
		logger.WithError(err).Fatal("Could not register alpha sdk grpc-gateway")
	}

	if err := sdkbeta.RegisterSDKHandler(ctx, mux, conn); err != nil {
		logger.WithError(err).Fatal("Could not register beta sdk grpc-gateway")
	}

	logger.WithField("httpEndpoint", httpServer.Addr).Info("Starting SDKServer grpc-gateway...")
	if err := httpServer.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			logger.WithError(err).Info("http server closed")
		} else {
			logger.WithError(err).Fatal("Could not serve http server")
		}
	}
}

// parseEnvFlags parses all the flags and environment variables and returns
// a configuration structure
func parseEnvFlags() config {
	viper.AllowEmptyEnv(true)
	viper.SetDefault(localFlag, false)
	viper.SetDefault(fileFlag, "")
	viper.SetDefault(testFlag, "")
	viper.SetDefault(testSdkNameFlag, "")
	viper.SetDefault(addressFlag, "localhost")
	viper.SetDefault(delayFlag, 0)
	viper.SetDefault(timeoutFlag, 0)
	viper.SetDefault(gracefulTerminationFlag, true)
	viper.SetDefault(grpcPortFlag, defaultGRPCPort)
	viper.SetDefault(httpPortFlag, defaultHTTPPort)
	viper.SetDefault(healthPortFlag, defaultHealthPort)
	viper.SetDefault(logLevelFlag, "Info")
	viper.SetDefault(requestRateLimitFlag, "500ms")
	pflag.String(gameServerNameFlag, viper.GetString(gameServerNameFlag),
		"Optional flag to set GameServer name. Overrides value given from `GAMESERVER_NAME` environment variable.")
	pflag.String(podNamespaceFlag, viper.GetString(gameServerNameFlag),
		"Optional flag to set Kubernetes namespace which the GameServer/pod is in. Overrides value given from `POD_NAMESPACE` environment variable.")
	pflag.Bool(localFlag, viper.GetBool(localFlag),
		"Set this, or LOCAL env, to 'true' to run this binary in local development mode. Defaults to 'false'")
	pflag.StringP(fileFlag, "f", viper.GetString(fileFlag), "Set this, or FILE env var to the path of a local yaml or json file that contains your GameServer resoure configuration")
	pflag.String(addressFlag, viper.GetString(addressFlag), "The Address to bind the server grpcPort to. Defaults to 'localhost'")
	pflag.Int(grpcPortFlag, viper.GetInt(grpcPortFlag), fmt.Sprintf("Port on which to bind the gRPC server. Defaults to %d", defaultGRPCPort))
	pflag.Int(httpPortFlag, viper.GetInt(httpPortFlag), fmt.Sprintf("Port on which to bind the HTTP server. Defaults to %d", defaultHTTPPort))
	pflag.Int(healthPortFlag, viper.GetInt(healthPortFlag), fmt.Sprintf("Port on which to bind the healthcheck port on the HTTP server. Defaults to %d", defaultHealthPort))
	pflag.Int(delayFlag, viper.GetInt(delayFlag), "Time to delay (in seconds) before starting to execute main. Useful for tests")
	pflag.Int(timeoutFlag, viper.GetInt(timeoutFlag), "Time of execution (in seconds) before close. Useful for tests")
	pflag.String(testFlag, viper.GetString(testFlag), "List functions which should be called during the SDK Conformance test run.")
	pflag.String(testSdkNameFlag, viper.GetString(testSdkNameFlag), "SDK name which is tested by this SDK Conformance test.")
	pflag.String(kubeconfigFlag, viper.GetString(kubeconfigFlag),
		"Optional. kubeconfig to run the SDK server out of the cluster.")
	pflag.Bool(gracefulTerminationFlag, viper.GetBool(gracefulTerminationFlag),
		"When false, immediately quits when receiving interrupt instead of waiting for GameServer state to progress to \"Shutdown\".")
	pflag.String(requestRateLimitFlag, viper.GetString(requestRateLimitFlag), "Time to delay between requests to the API server. Defaults to 500ms.")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(gameServerNameFlag))
	runtime.Must(viper.BindEnv(podNamespaceFlag))
	runtime.Must(viper.BindEnv(localFlag))
	runtime.Must(viper.BindEnv(fileFlag))
	runtime.Must(viper.BindEnv(addressFlag))
	runtime.Must(viper.BindEnv(testFlag))
	runtime.Must(viper.BindEnv(testSdkNameFlag))
	runtime.Must(viper.BindEnv(kubeconfigFlag))
	runtime.Must(viper.BindEnv(delayFlag))
	runtime.Must(viper.BindEnv(timeoutFlag))
	runtime.Must(viper.BindEnv(grpcPortFlag))
	runtime.Must(viper.BindEnv(httpPortFlag))
	runtime.Must(viper.BindEnv(healthPortFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(viper.BindEnv(logLevelFlag))
	runtime.Must(viper.BindEnv(requestRateLimitFlag))
	runtime.Must(runtime.FeaturesBindEnv())
	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		GameServerName:      viper.GetString(gameServerNameFlag),
		PodNamespace:        viper.GetString(podNamespaceFlag),
		IsLocal:             viper.GetBool(localFlag),
		Address:             viper.GetString(addressFlag),
		LocalFile:           viper.GetString(fileFlag),
		Delay:               viper.GetInt(delayFlag),
		Timeout:             viper.GetInt(timeoutFlag),
		Test:                viper.GetString(testFlag),
		TestSdkName:         viper.GetString(testSdkNameFlag),
		KubeConfig:          viper.GetString(kubeconfigFlag),
		GracefulTermination: viper.GetBool(gracefulTerminationFlag),
		GRPCPort:            viper.GetInt(grpcPortFlag),
		HTTPPort:            viper.GetInt(httpPortFlag),
		HealthPort:          viper.GetInt(healthPortFlag),
		LogLevel:            viper.GetString(logLevelFlag),
		RequestsRateLimit:   viper.GetDuration(requestRateLimitFlag),
	}
}

// config is all the configuration for this program
type config struct {
	GameServerName      string
	PodNamespace        string
	Address             string
	IsLocal             bool
	LocalFile           string
	Delay               int
	Timeout             int
	Test                string
	TestSdkName         string
	KubeConfig          string
	GracefulTermination bool
	GRPCPort            int
	HTTPPort            int
	HealthPort          int
	LogLevel            string
	RequestsRateLimit   time.Duration
}

// healthCheckWrapper ensures that an http 400 response is returned
// if the healthcheck receives a request with an empty post body
func healthCheckWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" && r.Body == http.NoBody {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	})
}
