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

// binary for the pinger service for RTT measurement.
package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/signals"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

const (
	httpResponseFlag = "http-response"
	udpRateLimitFlag = "udp-rate-limit"
)

var (
	logger = runtime.NewLoggerWithSource("main")
)

func main() {
	ctlConf := parseEnvFlags()
	if err := ctlConf.validate(); err != nil {
		logger.WithError(err).Fatal("Could not create controller from environment or flags")
	}

	logger.WithField("version", pkg.Version).WithField("featureGates", runtime.EncodeFeatures()).
		WithField("ctlConf", ctlConf).Info("Starting ping...")

	ctx, _ := signals.NewSigKillContext()

	udpSrv := serveUDP(ctx, ctlConf)
	defer udpSrv.close()

	h := healthcheck.NewHandler()
	h.AddLivenessCheck("udp-server", udpSrv.Health)

	cancel := serveHTTP(ctlConf, h)
	defer cancel()

	<-ctx.Done()
	logger.Info("Shutting down...")
}

func serveUDP(ctx context.Context, ctlConf config) *udpServer {
	s := newUDPServer(ctlConf.UDPRateLimit)
	s.run(ctx)
	return s
}

// serveHTTP starts the HTTP handler, and returns a cancel/shutdown function
func serveHTTP(ctlConf config, h healthcheck.Handler) func() {
	// we don't need a health checker, we already have a http endpoint that returns 200
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// add health check as well
	mux.HandleFunc("/live", h.LiveEndpoint)

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(ctlConf.HTTPResponse)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.WithError(err).Error("Error responding to http request")
		}
	})

	go func() {
		logger.Info("Starting HTTP Server...")
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Could not start HTTP server")
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.WithError(err).Fatal("Could not shut down HTTP server")
		}
		logger.Info("HTTP server was gracefully shut down")
	}
}

// config retains the configuration information
type config struct {
	HTTPResponse string
	UDPRateLimit rate.Limit
}

// validate returns an error if there is a validation problem
func (c *config) validate() error {
	if c.UDPRateLimit < 0 {
		return errors.New("UDP Rate limit must be greater that or equal to zero")
	}

	return nil
}

func parseEnvFlags() config {
	viper.SetDefault(httpResponseFlag, "ok")
	viper.SetDefault(udpRateLimitFlag, 20)

	pflag.String(httpResponseFlag, viper.GetString(httpResponseFlag), "Flag to set text value when a 200 response is returned. Can be useful to identify clusters. Defaults to 'ok' Can also use HTTP_RESPONSE env variable")
	pflag.Float64(udpRateLimitFlag, viper.GetFloat64(httpResponseFlag), "Flag to set how many UDP requests can be handled by a single source IP per second. Defaults to 20. Can also use UDP_RATE_LIMIT env variable")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(httpResponseFlag))
	runtime.Must(viper.BindEnv(udpRateLimitFlag))
	runtime.Must(runtime.FeaturesBindEnv())

	runtime.Must(runtime.ParseFeaturesFromEnv())

	return config{
		HTTPResponse: viper.GetString(httpResponseFlag),
		UDPRateLimit: rate.Limit(viper.GetFloat64(udpRateLimitFlag)),
	}
}
