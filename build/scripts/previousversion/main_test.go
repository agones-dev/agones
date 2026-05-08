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

	"github.com/pkg/errors"
	semver "github.com/blang/semver/v4"
)

// mockLister is a mock implementation of the gcsLister interface for testing.
type mockLister struct {
	output string
	err    error
}

func (m mockLister) List(prefix string) (string, error) {
	return m.output, m.err
}

func TestGetPreviousVersion(t *testing.T) {
	testCases := []struct {
		name         string
		version      string
		mockLister   mockLister
		expected     string
		expectErr    bool
	}{
		{
			name:    "Minor Release - gsutil success",
			version: "1.53.0",
			mockLister: mockLister{
				output: "gs://agones-chart/agones-1.52.0.tgz\ngs://agones-chart/agones-1.52.2.tgz\ngs://agones-chart/agones-1.52.1.tgz",
				err:    nil,
			},
			expected:  "1.52.2",
			expectErr: false,
		},
		{
			name:    "Minor Release - gsutil failure",
			version: "1.53.0",
			mockLister: mockLister{
				output: "",
				err:    errors.New("command failed"),
			},
			expected:  "1.52.0", // Falls back to .0
			expectErr: false,
		},
		{
			name:       "Patch Release",
			version:    "1.52.2",
			mockLister: mockLister{}, // Not used
			expected:   "1.52.1",
			expectErr:  false,
		},
		{
			name:      "Major Release - error",
			version:   "2.0.0",
			mockLister: mockLister{},
			expected:  "",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := semver.Parse(tc.version)
			result, err := getPreviousVersion(v, tc.mockLister)

			if (err != nil) != tc.expectErr {
				t.Fatalf("expected error: %v, but got: %v", tc.expectErr, err)
			}

			if !tc.expectErr {
				expectedVersion, _ := semver.Parse(tc.expected)
				if !result.Equals(expectedVersion) {
					t.Errorf("expected version %s, but got %s", tc.expected, result)
				}
			}
		})
	}
}

func TestGetLatestVersionFromGsutil(t *testing.T) {
	testCases := []struct {
		name          string
		gsutilOutput  string
		major, minor  uint64
		expected      semver.Version
	}{
		{
			name:         "No versions found",
			gsutilOutput: "",
			major:        1,
			minor:        52,
			expected:     semver.Version{Major: 1, Minor: 52, Patch: 0},
		},
		{
			name:         "One version found",
			gsutilOutput: "gs://agones-chart/agones-1.52.3.tgz",
			major:        1,
			minor:        52,
			expected:     semver.Version{Major: 1, Minor: 52, Patch: 3},
		},
		{
			name: "Multiple versions, out of order",
			gsutilOutput: `
gs://agones-chart/agones-1.52.2.tgz
gs://agones-chart/agones-1.52.0.tgz
gs://agones-chart/agones-1.52.10.tgz
gs://agones-chart/agones-1.52.1.tgz
`,
			major:    1,
			minor:    52,
			expected: semver.Version{Major: 1, Minor: 52, Patch: 10},
		},
		{
			name: "With non-matching lines",
			gsutilOutput: `
gs://agones-chart/agones-1.52.2.tgz
gs://agones-chart/agones-1.51.0.tgz
gs://agones-chart/agones-1.52.1.tgz
gs://agones-chart/index.yaml
`,
			major:    1,
			minor:    52,
			expected: semver.Version{Major: 1, Minor: 52, Patch: 2},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getLatestVersionFromGsutil(tc.gsutilOutput, tc.major, tc.minor)
			if !result.Equals(tc.expected) {
				t.Errorf("expected version %s, but got %s", tc.expected, result)
			}
		})
	}
}
