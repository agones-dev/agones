// Copyright Contributors to Agones a Series of LF Projects, LLC.
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
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
)

func init() {
	registerMetricViews()
}

func registerMetricViews() {
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		logger.WithError(err).Error("could not register view")
	}
}
