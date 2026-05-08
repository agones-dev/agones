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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncrementMinorVersion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard minor increment",`release_version = "1.52.0"`, `release_version = "1.53.0"`},
		{"Increment with existing patch", `release_version = "1.52.1"`, `release_version = "1.53.0"`},
		{"Double digit minor", `release_version = "1.9.5"`, `release_version = "1.10.0"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := incrementMinorVersion(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestIncrementPatchVersion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard patch increment", `release_version = "1.52.1"`, `release_version = "1.52.2"`},
		{"Patch increment to double digit", `release_version = "1.52.9"`, `release_version = "1.52.10"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := incrementPatchVersion(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestUpdateReleaseValues(t *testing.T) {
	testCases := []struct {
		name     string
		stage    string
		input    []string
		expected []string
	}{
		{
			"minor",
			"minor",
			[]string{`release_version = "1.52.0"`},
			[]string{`release_version = "1.53.0"`},
		},
		{
			"patch",
			"patch",
			[]string{`release_version = "1.52.1"`},
			[]string{`release_version = "1.52.2"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := updateReleaseValues(tc.input, tc.stage)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d lines, but got %d", len(tc.expected), len(result))
			}
			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("expected line %d to be %q, but got %q", i, tc.expected[i], result[i])
				}
			}
		})
	}
}

func TestTomlArray(t *testing.T) {
	input := []string{"1.26", "1.27"}
	expected := `["1.26", "1.27"]`
	result := tomlArray(input)
	if result != expected {
		t.Errorf("expected %s, but got %s", expected, result)
	}
}

func TestUpdateSiteConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-run-logic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.toml")

	initialContent := `
[params]
  supported_k8s = ["1.25", "1.26"]
  k8s_api_version = "1.26"
  release_version = "1.52.0"

[params.dev]
  dev_supported_k8s = ["1.26", "1.27"]
  dev_k8s_api_version = "1.27"
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial config file: %v", err)
	}

	// Test Minor Release
	if err := updateSiteConfig(configFile, "minor"); err != nil {
		t.Fatalf("run(minor) failed: %v", err)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read updated config file: %v", err)
	}

	if !strings.Contains(string(content), `supported_k8s = ["1.26", "1.27"]`) {
		t.Errorf("supported_k8s was not updated correctly for minor release")
	}
	if !strings.Contains(string(content), `release_version = "1.53.0"`) {
		t.Errorf("release_version was not updated correctly for minor release")
	}

	// Test Patch Release
	if err := updateSiteConfig(configFile, "patch"); err != nil {
		t.Fatalf("run(patch) failed: %v", err)
	}

	content, err = os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read updated config file: %v", err)
	}

	if !strings.Contains(string(content), `release_version = "1.53.1"`) {
		t.Errorf("release_version was not updated correctly for patch release")
	}
}
