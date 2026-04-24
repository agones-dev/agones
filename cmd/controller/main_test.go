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
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/portallocator"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestControllerConfigValidation(t *testing.T) {
	t.Parallel()

	c := config{PortRanges: map[string]portallocator.PortRange{
		agonesv1.DefaultPortRange: {MinPort: 10, MaxPort: 2},
	}}
	errs := c.validate()
	assert.Len(t, errs, 1)
	errorsContainString(t, errs, "max Port cannot be set less that the Min Port")

	c.PortRanges["game"] = portallocator.PortRange{MinPort: 20, MaxPort: 12}
	errs = c.validate()
	assert.Len(t, errs, 2)
	errorsContainString(t, errs, "max Port cannot be set less that the Min Port for port range game")

	c.SidecarMemoryRequest = resource.MustParse("2Gi")
	c.SidecarMemoryLimit = resource.MustParse("1Gi")
	errs = c.validate()
	assert.Len(t, errs, 3)
	errorsContainString(t, errs, "Request must be less than or equal to memory limit")

	c.SidecarMemoryLimit = resource.MustParse("2Gi")
	c.SidecarCPURequest = resource.MustParse("2m")
	c.SidecarCPULimit = resource.MustParse("1m")
	errs = c.validate()
	assert.Len(t, errs, 3)
	errorsContainString(t, errs, "Request must be less than or equal to cpu limit")

	c.SidecarMemoryLimit = resource.MustParse("2Gi")
	c.SidecarCPURequest = resource.MustParse("-2m")
	c.SidecarCPULimit = resource.MustParse("2m")
	errs = c.validate()
	assert.Len(t, errs, 3)
	errorsContainString(t, errs, "Resource cpu request value must be non negative")
}

func TestControllerConfigValidation_PortRangeOverlap(t *testing.T) {
	t.Parallel()

	c := config{
		PortRanges: map[string]portallocator.PortRange{
			agonesv1.DefaultPortRange: {MinPort: 10, MaxPort: 20},
			"game":                    {MinPort: 15, MaxPort: 25},
			"other":                   {MinPort: 21, MaxPort: 31},
		},
	}
	errs := c.validate()
	assert.Len(t, errs, 2)
	errorsContainString(t, errs, "port range game overlaps with min/max port")
}

func errorsContainString(t *testing.T, errs []error, expected string) {
	found := false
	for _, v := range errs {
		if expected == v.Error() {
			found = true
			break
		}
	}
	assert.True(t, found, "Was not able to find '%s'", expected)
}
