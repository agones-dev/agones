/*
 * Copyright Contributors to Agones a Series of LF Projects, LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleetautoscalers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultWasmFixtures creates default fixtures for testing WasmPolicy
func defaultWasmFixtures() (*autoscalingv1.FleetAutoscaler, *agonesv1.Fleet) {
	fas, f := defaultFixtures()
	fas.Spec.Policy.Type = autoscalingv1.WasmPolicyType
	fas.Spec.Policy.Buffer = nil

	// Set up WasmPolicy
	url := "plugin.wasm"
	fas.Spec.Policy.Wasm = &autoscalingv1.WasmPolicy{
		Function: "scale",
		Config: map[string]string{
			"buffer_size": "5",
		},
		From: autoscalingv1.WasmFrom{
			URL: &autoscalingv1.URLConfiguration{
				URL: &url,
			},
		},
	}

	return fas, f
}

func TestApplyWasmPolicy(t *testing.T) {
	t.Parallel()

	// Enable the WASM autoscaler feature flag for testing
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()
	utilruntime.EnableAllFeatures()

	// Find the WASM plugin file
	wasmFilePath, err := filepath.Abs(filepath.Join("..", "..", "examples", "autoscaler-wasm"))
	require.NoError(t, err)

	_, err = os.Stat(wasmFilePath)
	require.NoError(t, err, "WASM plugin file not found at %s", wasmFilePath)

	// Create a test server to serve the WASM plugin
	svrDir := http.Dir(wasmFilePath)
	sourcePluginPath := filepath.Join(wasmFilePath, "plugin.wasm")
	// Compute the SHA256 hash of the plugin for Hash tests
	pluginBytes, err := os.ReadFile(sourcePluginPath)
	require.NoError(t, err)
	sum := sha256.Sum256(pluginBytes)
	hashStr := hex.EncodeToString(sum[:])
	// Create an incorrect hash (same length, wrong value) for negative test
	badSum := make([]byte, len(sum))
	copy(badSum, sum[:])
	badSum[0] ^= 0xFF // flip first byte to ensure mismatch
	badHash := hex.EncodeToString(badSum)

	sourceFS := http.FileServer(svrDir)
	srv := httptest.NewServer(sourceFS)
	defer srv.Close()

	// Create test fixtures
	fas, f := defaultWasmFixtures()

	// Update the URL to point to our test server
	fileURL := srv.URL + "/plugin.wasm"
	fas.Spec.Policy.Wasm.From.URL.URL = &fileURL

	// Create a logger for testing
	logger := &FasLogger{
		fas:        fas,
		baseLogger: newTestLogger(),
	}

	type expected struct {
		replicas int32
		limited  bool
		err      string
	}

	type testCase struct {
		wasmPolicy              *autoscalingv1.WasmPolicy
		fleet                   *agonesv1.Fleet
		specReplicas            int32
		statusReplicas          int32
		statusAllocatedReplicas int32
		statusReadyReplicas     int32
		expected                expected
	}

	var testCases = map[string]testCase{
		"Correct Hash provided (sha256), scale up needed": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
				Hash: hashStr,
			},
			fleet:                   f,
			specReplicas:            10,
			statusReplicas:          10,
			statusAllocatedReplicas: 8,
			statusReadyReplicas:     2,
			expected: expected{
				replicas: 13,
				limited:  false,
				err:      "",
			},
		},
		"Incorrect Hash provided (sha256), plugin creation fails": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
				Hash: badHash,
			},
			fleet: f,
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "hash mismatch for module",
			},
		},
		"Default buffer size (5), scale up needed": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
			},
			fleet:                   f,
			specReplicas:            10,
			statusReplicas:          10,
			statusAllocatedReplicas: 8,
			statusReadyReplicas:     2,
			expected: expected{
				replicas: 13, // allocated (8) + buffer (5)
				limited:  false,
				err:      "",
			},
		},
		"Default buffer size (5), no scaling needed": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
			},
			fleet:                   f,
			specReplicas:            15,
			statusReplicas:          15,
			statusAllocatedReplicas: 10,
			statusReadyReplicas:     5,
			expected: expected{
				replicas: 15, // already at the right size
				limited:  false,
				err:      "",
			},
		},
		"Custom buffer size (10), scale up needed": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "10",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
			},
			fleet:                   f,
			specReplicas:            15,
			statusReplicas:          15,
			statusAllocatedReplicas: 10,
			statusReadyReplicas:     5,
			expected: expected{
				replicas: 20, // allocated (10) + buffer (10)
				limited:  false,
				err:      "",
			},
		},
		"nil WasmPolicy, error returned": {
			wasmPolicy: nil,
			fleet:      f,
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "wasmPolicy parameter must not be nil",
			},
		},
		"nil Fleet, error returned": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
			},
			fleet: nil,
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "fleet parameter must not be nil",
			},
		},
		"Invalid URL in WasmPolicy": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scale",
				Config: map[string]string{
					"buffer_size": "5",
				},
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: nil,
					},
				},
			},
			fleet: f,
			expected: expected{
				replicas: 0,
				limited:  false,
				err:      "service was not provided, either URL or Service must be provided",
			},
		},
		"Function set to scaleNone, no scaling occurs": {
			wasmPolicy: &autoscalingv1.WasmPolicy{
				Function: "scaleNone",
				From: autoscalingv1.WasmFrom{
					URL: &autoscalingv1.URLConfiguration{
						URL: &fileURL,
					},
				},
			},
			fleet:                   f,
			specReplicas:            10,
			statusReplicas:          10,
			statusAllocatedReplicas: 8,
			statusReadyReplicas:     2,
			expected: expected{
				replicas: 10,
				limited:  false,
				err:      "",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			var fleet *agonesv1.Fleet
			if tc.fleet != nil {
				fleet = tc.fleet.DeepCopy()
				fleet.Spec.Replicas = tc.specReplicas
				fleet.Status.Replicas = tc.statusReplicas
				fleet.Status.AllocatedReplicas = tc.statusAllocatedReplicas
				fleet.Status.ReadyReplicas = tc.statusReadyReplicas
			}

			// Create a new state for each test case
			state := fasState{}

			replicas, limited, err := applyWasmPolicy(context.Background(), &state, tc.wasmPolicy, fleet, logger)

			if tc.expected.err != "" {
				require.ErrorContains(t, err, tc.expected.err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected.replicas, replicas)
				assert.Equal(t, tc.expected.limited, limited)
			}
		})
	}
}
