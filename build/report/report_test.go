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

package main

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplateRendering(t *testing.T) {
	require.NoError(t, newReportTemplate().Execute(io.Discard, report{
		WindowStart: "2014-09-01",
		WindowEnd:   "2100-09-01",
		BuildCount:  1000,
		FlakeCount:  10000,
		FlakeRatio:  0.12,
		Flakes:      []flake{{ID: "awesome-id", CreateTime: "1978-04-28"}},
	}))

	require.NoError(t, newRedirectTemplate().Execute(io.Discard, redirect{"2014-09-01"}))
}
