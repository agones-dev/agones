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

// Package runtime handles runtime errors
// Wraps and reconfigures functionality in apimachinery/pkg/runtime
package runtime

import (
	"context"
	"fmt"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/runtime"
	restclient "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

const sourceKey = "source"

// stackTracer is the pkg/errors stacktrace interface
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// replace the standard glog error logger, with a logrus one
func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
	})

	runtime.ErrorHandlers[0] = func(_ context.Context, err error, _ string, _ ...interface{}) {
		if stackTrace, ok := err.(stackTracer); ok {
			var stack []string
			for _, f := range stackTrace.StackTrace() {
				stack = append(stack, fmt.Sprintf("%+v", f))
			}
			logrus.WithField("stack", stack).Error(err)
		} else {
			logrus.Error(err)
		}
	}
}

// SetLevel select level to filter logger output
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// HandleError wraps runtime.HandleError so that it is possible to
// use WithField with logrus.
func HandleError(logger *logrus.Entry, err error) {
	if logger != nil {
		// it's a bit of a double handle, but I can't see a better way to do it
		logger.WithError(err).Error()
	}
	runtime.HandleError(err)
}

// Must panics if there is an error
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// NewLoggerWithSource returns a logrus.Entry to use when you want to specify an source
func NewLoggerWithSource(source string) *logrus.Entry {
	return logrus.WithField(sourceKey, source)
}

// NewLoggerWithType returns a logrus.Entry to use when you want to use a data type as the source
// such as when you have a struct with methods
func NewLoggerWithType(obj interface{}) *logrus.Entry {
	return NewLoggerWithSource(fmt.Sprintf("%T", obj))
}

// NewServerMux returns a ServeMux which is a request multiplexer for grpc-gateway.
// It matches http requests to pattern and invokes the corresponding handler.
// ref: https://grpc-ecosystem.github.io/grpc-gateway/docs/development/grpc-gateway_v2_migration_guide/#we-now-emit-default-values-for-all-fields
func NewServerMux() *gwruntime.ServeMux {
	mux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.HTTPBodyMarshaler{
			Marshaler: &gwruntime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
	)
	return mux
}

// InClusterBuildConfig is a helper function that first attempts to build configurations
// using InClusterConfig(). If InClusterConfig is unsuccessful, it then tries to build
// configurations from a kubeconfigPath. This path is typically passed in as a command line
// flag for cluster components. If neither the InClusterConfig nor the kubeconfigPath
// are successful, the function logs a warning and falls back to a default configuration.
func InClusterBuildConfig(logger *logrus.Entry, kubeconfigPath string) (*restclient.Config, error) {
	kubeconfig, err := restclient.InClusterConfig()
	if err == nil {
		return kubeconfig, nil
	}
	logger.WithError(err).Warning("Error creating inClusterConfig, trying to build config from flags", err)

	if kubeconfigPath == "" {
		logrus.Warning("No kubeconfigPath provided. Attempting to use a default configuration.")
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{}).ClientConfig()
}
