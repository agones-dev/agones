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

package allocator

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	helper "agones.dev/agones/test/e2e/allochelper"
	e2eframework "agones.dev/agones/test/e2e/framework"
	log "github.com/sirupsen/logrus"
)

var framework *e2eframework.Framework

func TestMain(m *testing.M) {
	log.SetFormatter(&log.TextFormatter{
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05.000",
	})

	var (
		err      error
		exitCode int
	)

	if err = e2eframework.ParseTestFlags(); err != nil {
		log.WithError(err).Error("failed to parse go test flags")
		os.Exit(1)
	}

	if framework, err = e2eframework.NewFromFlags(); err != nil {
		log.WithError(err).Error("failed to setup framework")
		os.Exit(1)
	}

	if err = helper.CleanupNamespaces(context.Background(), framework); err != nil {
		log.WithError(err).Error("failed to cleanup e2e namespaces")
		os.Exit(1)
	}

	if framework.Namespace == "" {
		// use a custom namespace - Unix timestamp
		framework.Namespace = strconv.Itoa(int(time.Now().Unix()))
		log.Infof("Custom namespace is set: %s", framework.Namespace)

		if err := framework.CreateNamespace(framework.Namespace); err != nil {
			log.WithError(err).Error("failed to create a custom namespace")
			os.Exit(1)
		}

		defer func() {
			if derr := framework.DeleteNamespace(framework.Namespace); derr != nil {
				log.Error(derr)
			}
			os.Exit(exitCode)
		}()
	} else {
		// use an already existing namespace
		// run cleanup before tests to ensure no resources from previous runs exist
		err = framework.CleanUp(framework.Namespace)
		if err != nil {
			log.WithError(err).Error("failed to cleanup resources")
		}

		defer func() {
			err = framework.CleanUp(framework.Namespace)
			if err != nil {
				log.WithError(err).Error("failed to cleanup resources")
			}
			os.Exit(exitCode)
		}()
	}

	exitCode = m.Run()
}
