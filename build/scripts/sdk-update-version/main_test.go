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
	"testing"
)

func TestIncrementVersionAfterRelease(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Standard case", "1.2.3", "1.3.3"},
		{"Zero case", "1.0.0", "1.1.0"},
		{"Double digit", "1.10.10", "1.11.10"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := incrementVersionAfterRelease(tc.input)
			if result != tc.expected {
				t.Errorf("expected %s, but got %s", tc.expected, result)
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
		{"Standard case", "1.2.3", "1.2.4"},
		{"Zero case", "1.0.0", "1.0.1"},
		{"Double digit", "1.10.9", "1.10.10"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := incrementPatchVersion(tc.input)
			if result != tc.expected {
				t.Errorf("expected %s, but got %s", tc.expected, result)
			}
		})
	}
}

func TestUpdateFile(t *testing.T) {
	// Create a temporary directory for our test files
	tmpDir, err := os.MkdirTemp("", "test-update-file")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		// Log error to make linter happy. CI / CD env will clean up, so no need to handle err.
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Log(err)
		}
	}()

	// Test cases
	testCases := []struct {
		name            string
		releaseStage    string
		initialFile     string
		initialContent  string
		version         string
		expectedContent string
	}{
		{
			"before-json",
			"before",
			"test.json",
			`{"version": "1.2.3-dev"}`,
			"1.2.3",
			`{"version": "1.2.3"}`,
		},
		{
			"before-yaml",
			"before",
			"test.yaml",
			`version: 1.2.3-dev`,
			"1.2.3",
			`version: 1.2.3`,
		},
		{
			"after-makefile",
			"after",
			"build/Makefile",
			`version = 1.2.3`,
			"1.2.3",
			`version = 1.3.3`,
		},
		{
			"after-json",
			"after",
			"test.json",
			`{"version": "1.2.3"}`,
			"1.2.3",
			`{"version": "1.3.3-dev"}`,
		},
		{
			"patch-yaml",
			"patch",
			"test.yaml",
			`version: 1.2.3`,
			"1.2.3",
			`version: 1.2.4`,
		},
		{
			"patch-json",
			"patch",
			"test.json",
			`{"version": "1.2.3"}`,
			"1.2.3",
			`{"version": "1.2.4"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the global release stage for this test
			releaseStage = tc.releaseStage

			// Create the test file
			filePath := filepath.Join(tmpDir, tc.initialFile)
			err := os.MkdirAll(filepath.Dir(filePath), 0o755)
			if err != nil {
				t.Fatalf("Failed to create dir for test file: %v", err)
			}
			err = os.WriteFile(filePath, []byte(tc.initialContent), 0o644)
			if err != nil {
				t.Fatalf("Failed to write initial test file: %v", err)
			}

			// Run the function
			err = updateFile(filePath, tc.version)
			if err != nil {
				t.Errorf("UpdateFile returned an error: %v", err)
			}

			// Read the file and check the content
			updatedContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read updated test file: %v", err)
			}

			if string(updatedContent) != tc.expectedContent {
				t.Errorf("expected content %q, but got %q", tc.expectedContent, string(updatedContent))
			}
		})
	}
}
