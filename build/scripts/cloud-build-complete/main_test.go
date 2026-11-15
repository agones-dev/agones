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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-github/v74/github"
)

const (
	testBuildID    = "afde3318-6650-4eaf-b0c4-0e990e00962a"
	testSHA        = "a9a989f23ef3fd1cfa526555d048cab5e2deb233"
	testPreviewURL = "https://a9a989f-dot-preview-dot-agones-images.uc.r.appspot.com/"
	testDetailsURL = "https://console.cloud.google.com/cloud-build/builds/afde3318-6650-4eaf-b0c4-0e990e00962a?project=258182270954"
)

func TestPreviewURL(t *testing.T) {
	testCases := []struct {
		name    string
		file    string
		wantURL string
		wantOK  bool
	}{
		{name: "successful build deploys the preview", file: "testdata/success.log", wantURL: testPreviewURL, wantOK: true},
		{name: "build that failed before deploying", file: "testdata/failure.log", wantOK: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := previewURL(readFile(t, tc.file))
			if got != tc.wantURL || ok != tc.wantOK {
				t.Errorf("previewURL() = %q, %v; want %q, %v", got, ok, tc.wantURL, tc.wantOK)
			}
		})
	}
}

func TestReadEvent(t *testing.T) {
	event, err := readEvent("testdata/check_run.json")
	if err != nil {
		t.Fatalf("readEvent() error: %v", err)
	}
	got := []string{
		event.GetRepo().GetOwner().GetLogin(),
		event.GetRepo().GetName(),
		event.GetCheckRun().GetHeadSHA(),
		event.GetCheckRun().GetOutput().GetSummary(),
		event.GetCheckRun().GetDetailsURL(),
	}
	want := []string{"agones-dev", "agones", testSHA, "Build " + testBuildID + " successful", testDetailsURL}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readEvent() = %q; want %q", got, want)
	}
}

func TestWebsitePreviewRun(t *testing.T) {
	testCases := []struct {
		name string
		// logResponses are the status codes the log server returns, in order; the last one repeats.
		logResponses []int
		logFile      string
		// partialLogs is how many successful log responses serve only the start of logFile, as if
		// it were still being written.
		partialLogs int
		summary     string
		// existingStatuses, if not nil, are the states of an existing deployment for the commit.
		existingStatuses []string
		// statusCode is the response code for creating the deployment status, default 201.
		statusCode         int
		wantErr            bool
		wantLogRequests    int
		wantDeployment     map[string]any
		wantStatus         map[string]any
		wantGitHubRequests int
		// wantNotice is whether a "not deployed" notice and job summary are written.
		wantNotice bool
	}{
		{
			name:            "creates a deployment for the preview",
			logResponses:    []int{http.StatusOK},
			logFile:         "testdata/success.log",
			summary:         "Build " + testBuildID + " successful",
			wantLogRequests: 1,
			wantDeployment: map[string]any{
				"ref":                    testSHA,
				"environment":            previewEnvironment,
				"description":            "Website preview",
				"auto_merge":             false,
				"required_contexts":      []any{},
				"transient_environment":  true,
				"production_environment": false,
			},
			wantStatus: map[string]any{
				"state":           "success",
				"environment_url": testPreviewURL,
				"log_url":         testDetailsURL,
				"auto_inactive":   false,
			},
			wantGitHubRequests: 3,
		},
		{
			name:               "retries until the log is available",
			logResponses:       []int{http.StatusNotFound, http.StatusNotFound, http.StatusOK},
			logFile:            "testdata/success.log",
			summary:            "Build " + testBuildID + " successful",
			wantLogRequests:    3,
			wantDeployment:     map[string]any{"ref": testSHA},
			wantStatus:         map[string]any{"environment_url": testPreviewURL},
			wantGitHubRequests: 3,
		},
		{
			name:               "retries a partly written log",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/success.log",
			partialLogs:        2,
			summary:            "Build " + testBuildID + " successful",
			wantLogRequests:    3,
			wantDeployment:     map[string]any{"ref": testSHA},
			wantStatus:         map[string]any{"environment_url": testPreviewURL},
			wantGitHubRequests: 3,
		},
		{
			name:               "uses a log that never looks complete",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/success.log",
			partialLogs:        5,
			summary:            "Build " + testBuildID + " cancelled",
			wantLogRequests:    5,
			wantGitHubRequests: 0,
			wantNotice:         true,
		},
		{
			name:               "skips a build that did not deploy the preview",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/failure.log",
			summary:            "Build " + testBuildID + " failed",
			wantLogRequests:    1,
			wantGitHubRequests: 0,
			wantNotice:         true,
		},
		{
			name:               "skips a commit whose preview deployment has a success status",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/success.log",
			summary:            "Build " + testBuildID + " successful",
			existingStatuses:   []string{"success"},
			wantLogRequests:    1,
			wantGitHubRequests: 2,
		},
		{
			name:               "adds the status to a preview deployment that has none",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/success.log",
			summary:            "Build " + testBuildID + " successful",
			existingStatuses:   []string{},
			wantLogRequests:    1,
			wantStatus:         map[string]any{"state": "success", "environment_url": testPreviewURL},
			wantGitHubRequests: 3,
		},
		{
			name:               "fails when the deployment status can't be created",
			logResponses:       []int{http.StatusOK},
			logFile:            "testdata/success.log",
			summary:            "Build " + testBuildID + " successful",
			statusCode:         http.StatusInternalServerError,
			wantErr:            true,
			wantLogRequests:    1,
			wantDeployment:     map[string]any{"ref": testSHA},
			wantStatus:         map[string]any{"environment_url": testPreviewURL},
			wantGitHubRequests: 3,
		},
		{
			name:               "skips a summary with no build id",
			summary:            "Needs /gcbrun from a collaborator",
			wantLogRequests:    0,
			wantGitHubRequests: 0,
		},
		{
			name:               "fails when the log never becomes available",
			logResponses:       []int{http.StatusNotFound},
			summary:            "Build " + testBuildID + " successful",
			wantErr:            true,
			wantLogRequests:    5,
			wantGitHubRequests: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				mu             sync.Mutex
				logRequests    int
				githubRequests int
				deployment     map[string]any
				status         map[string]any
			)
			mux := http.NewServeMux()
			mux.HandleFunc("GET /logs/{file}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				code := tc.logResponses[min(logRequests, len(tc.logResponses)-1)]
				logRequests++
				okResponses := logRequests - (len(tc.logResponses) - 1)
				mu.Unlock()
				if got, want := r.PathValue("file"), "log-"+testBuildID+".txt"; got != want {
					t.Errorf("log requested = %q; want %q", got, want)
				}
				if code != http.StatusOK {
					w.WriteHeader(code)
					return
				}
				body := readFile(t, tc.logFile)
				if okResponses <= tc.partialLogs {
					// Cut the log off before the deploy-site-static step and the end of the build.
					before, _, found := bytes.Cut(body, []byte(`Starting Step #23 - "deploy-site-static"`))
					if !found {
						t.Errorf("%s has no deploy-site-static step to cut the log at", tc.logFile)
					}
					body = before
				}
				_, _ = w.Write(body)
			})
			mux.HandleFunc("GET /repos/agones-dev/agones/deployments", func(w http.ResponseWriter, r *http.Request) {
				countRequest(&mu, &githubRequests)
				if got, want := r.URL.Query(), (url.Values{"sha": {testSHA}, "environment": {previewEnvironment}}); !reflect.DeepEqual(got, want) {
					t.Errorf("list deployments query = %v; want %v", got, want)
				}
				list := []map[string]any{}
				if tc.existingStatuses != nil {
					list = append(list, map[string]any{"id": 42})
				}
				writeJSON(t, w, list)
			})
			mux.HandleFunc("GET /repos/agones-dev/agones/deployments/42/statuses", func(w http.ResponseWriter, _ *http.Request) {
				countRequest(&mu, &githubRequests)
				list := []map[string]any{}
				for i, state := range tc.existingStatuses {
					list = append(list, map[string]any{"id": i + 1, "state": state})
				}
				writeJSON(t, w, list)
			})
			mux.HandleFunc("POST /repos/agones-dev/agones/deployments", func(w http.ResponseWriter, r *http.Request) {
				countRequest(&mu, &githubRequests)
				deployment = decodeJSON(t, r.Body)
				w.WriteHeader(http.StatusCreated)
				writeJSON(t, w, map[string]any{"id": 42})
			})
			mux.HandleFunc("POST /repos/agones-dev/agones/deployments/42/statuses", func(w http.ResponseWriter, r *http.Request) {
				countRequest(&mu, &githubRequests)
				status = decodeJSON(t, r.Body)
				if tc.statusCode != 0 {
					w.WriteHeader(tc.statusCode)
					return
				}
				w.WriteHeader(http.StatusCreated)
				writeJSON(t, w, map[string]any{"id": 1})
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			gh := github.NewClient(nil)
			baseURL, err := url.Parse(srv.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			gh.BaseURL = baseURL

			stdout := &strings.Builder{}
			summaryPath := filepath.Join(t.TempDir(), "summary.md")
			w := &websitePreview{
				gh:           gh,
				http:         srv.Client(),
				logURLFormat: srv.URL + "/logs/log-%s.txt",
				retryDelay:   0,
				stdout:       stdout,
				summaryPath:  summaryPath,
			}
			event := &github.CheckRunEvent{
				Repo: &github.Repository{Name: github.Ptr("agones"), Owner: &github.User{Login: github.Ptr("agones-dev")}},
				CheckRun: &github.CheckRun{
					HeadSHA:    github.Ptr(testSHA),
					DetailsURL: github.Ptr(testDetailsURL),
					Output:     &github.CheckRunOutput{Summary: github.Ptr(tc.summary)},
				},
			}

			err = w.run(context.Background(), event)
			if (err != nil) != tc.wantErr {
				t.Fatalf("run() error = %v; wantErr %v", err, tc.wantErr)
			}
			if logRequests != tc.wantLogRequests {
				t.Errorf("log requests = %d; want %d", logRequests, tc.wantLogRequests)
			}
			if githubRequests != tc.wantGitHubRequests {
				t.Errorf("GitHub API requests = %d; want %d", githubRequests, tc.wantGitHubRequests)
			}
			summary, err := os.ReadFile(summaryPath)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			if tc.wantNotice {
				wantNotice := "::notice title=Website preview not deployed::Build " + testBuildID + " for commit " + testSHA + " did not deploy the website preview.\n"
				if stdout.String() != wantNotice {
					t.Errorf("stdout = %q; want %q", stdout.String(), wantNotice)
				}
				for _, want := range []string{"Website preview not deployed", "[Build " + testBuildID + "](" + testDetailsURL + ")", "/agones-dev/agones/commit/" + testSHA} {
					if !strings.Contains(string(summary), want) {
						t.Errorf("job summary %q does not contain %q", summary, want)
					}
				}
			} else if stdout.Len() > 0 || len(summary) > 0 {
				t.Errorf("unexpected notice %q or job summary %q", stdout.String(), summary)
			}
			assertFields(t, "deployment", deployment, tc.wantDeployment)
			assertFields(t, "deployment status", status, tc.wantStatus)
		})
	}
}

// assertFields checks that got has each of the fields in want, and is nil if want is.
func assertFields(t *testing.T, name string, got, want map[string]any) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("unexpected %s request: %v", name, got)
		}
		return
	}
	for k, v := range want {
		gotV, ok := got[k]
		if !ok {
			t.Errorf("%s is missing field %q; got %v", name, k, got)
			continue
		}
		if !reflect.DeepEqual(gotV, v) {
			t.Errorf("%s field %q = %#v; want %#v", name, k, gotV, v)
		}
	}
}

func countRequest(mu *sync.Mutex, n *int) {
	mu.Lock()
	defer mu.Unlock()
	*n++
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decodeJSON(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	m := map[string]any{}
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Errorf("could not decode request body: %v", err)
	}
	return m
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Error(fmt.Errorf("could not encode response: %w", err))
	}
}
