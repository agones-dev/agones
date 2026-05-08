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
	"fmt"
	"strconv"

	"github.com/extism/go-pdk"
)

const DEFAULT_BUFFER_SIZE = 5

// Scale makes sure there is a buffer of replicas available for the fleet.
//
//go:wasmexport scale
func Scale() int32 {
	var review FleetAutoscaleReview
	err := pdk.InputJSON(&review)
	if err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to decode FleetAutoscaleReview: %v", err))
		return 1
	}

	if review.Request == nil {
		pdk.SetErrorString("FleetAutoscaleReview Request is nil")
		return 1
	}

	if review.Response == nil {
		review.Response = &FleetAutoscaleResponse{}
	}
	review.Response.UID = review.Request.UID
	review.Response.Scale = false

	// allow for an overwrite of the buffer size
	var bufferSize int32
	b, ok := pdk.GetConfig("buffer_size")
	if !ok {
		bufferSize = DEFAULT_BUFFER_SIZE
	} else {
		// convert b to an int32, and store it in bufferSize
		i, err := strconv.Atoi(b)
		if err != nil {
			pdk.SetErrorString(fmt.Sprintf("Failed to convert buffer_size to int32: %v", err))
			return 1
		}
		bufferSize = int32(i)
	}

	// Example basic logic to determine if we should scale with a simple buffer size.
	expected := bufferSize + review.Request.Status.AllocatedReplicas
	if expected != review.Request.Status.Replicas {
		review.Response.Scale = true
		review.Response.Replicas = expected
	}

	err = pdk.OutputJSON(&review)
	if err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to encode FleetAutoscaleReview: %v", err))
		return 1
	}

	return 0
}

// ScaleNone is a second export that does not scale, demonstrating multiple export functions.
//
//go:wasmexport scaleNone
func ScaleNone() int32 {
	var review FleetAutoscaleReview
	err := pdk.InputJSON(&review)
	if err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to decode FleetAutoscaleReview: %v", err))
		return 1
	}

	if review.Request == nil {
		pdk.SetErrorString("FleetAutoscaleReview Request is nil")
		return 1
	}

	if review.Response == nil {
		review.Response = &FleetAutoscaleResponse{}
	}
	review.Response.UID = review.Request.UID
	review.Response.Replicas = review.Request.Status.Replicas

	err = pdk.OutputJSON(&review)
	if err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to encode FleetAutoscaleReview: %v", err))
		return 1
	}

	return 0
}

func main() {}
