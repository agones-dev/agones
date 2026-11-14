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

// Package main runs the follow-up tasks for a finished Agones Cloud Build PR build. It is run by the
// .github/workflows/cloud_build_complete.yaml workflow, and works from the check_run event and the
// public build log, so it needs no Google Cloud credentials.
//
// Usage: cloud-build-complete website-preview
//
// It reads the event from GITHUB_EVENT_PATH, authenticates to GitHub with GITHUB_TOKEN, and writes
// notable outcomes to the job summary at GITHUB_STEP_SUMMARY.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/google/go-github/v74/github"
)

const (
	logURLFormat       = "https://storage.googleapis.com/agones-build-logs/log-%s.txt"
	previewEnvironment = "website-preview"
)

var (
	// The check run summary is "Build <id> successful" or "Build <id> failed".
	buildIDRegexp = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	// Printed by `gcloud app deploy` in the deploy-site-static step of cloudbuild.yaml.
	previewRegexp = regexp.MustCompile(`(?m)^Step #\d+ - "deploy-site-static": Deployed service \[preview\] to \[(https://[^\]]+)\]`)
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 || flag.Arg(0) != "website-preview" {
		log.Fatalf("usage: %s website-preview", os.Args[0])
	}

	event, err := readEvent(os.Getenv("GITHUB_EVENT_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	w := &websitePreview{
		gh:           github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_TOKEN")),
		http:         &http.Client{Timeout: time.Minute},
		logURLFormat: logURLFormat,
		retryDelay:   10 * time.Second,
		stdout:       os.Stdout,
		summaryPath:  os.Getenv("GITHUB_STEP_SUMMARY"),
	}
	if err := w.run(context.Background(), event); err != nil {
		log.Fatal(err)
	}
}

func readEvent(path string) (*github.CheckRunEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open event file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read only

	event := &github.CheckRunEvent{}
	if err := json.NewDecoder(f).Decode(event); err != nil {
		return nil, fmt.Errorf("could not decode check_run event: %w", err)
	}
	return event, nil
}

// websitePreview records the website preview deployed by a Cloud Build PR build as a GitHub
// Deployment on the build's commit, so that it shows on the pull request.
type websitePreview struct {
	gh   *github.Client
	http *http.Client
	// stdout receives GitHub Actions workflow commands, such as notices.
	stdout       io.Writer
	logURLFormat string
	// summaryPath is the job summary file, if any.
	summaryPath string
	retryDelay  time.Duration
}

func (w *websitePreview) run(ctx context.Context, event *github.CheckRunEvent) error {
	checkRun := event.GetCheckRun()
	owner := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	sha := checkRun.GetHeadSHA()

	buildID := buildIDRegexp.FindString(checkRun.GetOutput().GetSummary())
	if buildID == "" {
		log.Printf("No build id in check run summary %q, skipping", checkRun.GetOutput().GetSummary())
		return nil
	}
	log.Printf("Build %s, commit %s, conclusion %s", buildID, sha, checkRun.GetConclusion())

	buildLog, err := w.fetchLog(ctx, fmt.Sprintf(w.logURLFormat, buildID))
	if err != nil {
		return err
	}

	url, ok := previewURL(buildLog)
	if !ok {
		return w.notice("Website preview not deployed",
			fmt.Sprintf("Build %s for commit %s did not deploy the website preview.", buildID, sha),
			fmt.Sprintf("## :warning: Website preview not deployed\n\n[Build %s](%s) for commit [%s](https://github.com/%s/%s/commit/%s) "+
				"did not deploy the website preview, so no deployment was created.\n", buildID, checkRun.GetDetailsURL(), sha[:min(7, len(sha))], owner, repo, sha))
	}

	existing, _, err := w.gh.Repositories.ListDeployments(ctx, owner, repo, &github.DeploymentsListOptions{SHA: sha, Environment: previewEnvironment})
	if err != nil {
		return fmt.Errorf("could not list deployments: %w", err)
	}
	if len(existing) > 0 {
		log.Printf("A %s deployment already exists for %s", previewEnvironment, sha)
		return nil
	}

	deployment, _, err := w.gh.Repositories.CreateDeployment(ctx, owner, repo, &github.DeploymentRequest{
		Ref:         github.Ptr(sha),
		Environment: github.Ptr(previewEnvironment),
		Description: github.Ptr("Website preview"),
		// Don't merge the default branch into the ref, or wait on the commit's other checks.
		AutoMerge:        github.Ptr(false),
		RequiredContexts: &[]string{},
		// Transient deployments aren't marked inactive by later deployments to the same
		// environment, so each PR keeps its own preview.
		TransientEnvironment:  github.Ptr(true),
		ProductionEnvironment: github.Ptr(false),
	})
	if err != nil {
		return fmt.Errorf("could not create deployment: %w", err)
	}

	if _, _, err := w.gh.Repositories.CreateDeploymentStatus(ctx, owner, repo, deployment.GetID(), &github.DeploymentStatusRequest{
		State:          github.Ptr("success"),
		EnvironmentURL: github.Ptr(url),
		LogURL:         github.Ptr(checkRun.GetDetailsURL()),
		AutoInactive:   github.Ptr(false),
	}); err != nil {
		return fmt.Errorf("could not create deployment status: %w", err)
	}

	log.Printf("Created deployment %d for %s: %s", deployment.GetID(), sha, url)
	return nil
}

// notice shows message as a notice annotation on the workflow run's summary page, and appends
// markdown to the job summary.
func (w *websitePreview) notice(title, message, markdown string) error {
	log.Print(message)
	if _, err := fmt.Fprintf(w.stdout, "::notice title=%s::%s\n", title, message); err != nil {
		return fmt.Errorf("could not write notice: %w", err)
	}
	if w.summaryPath == "" {
		return nil
	}
	f, err := os.OpenFile(w.summaryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("could not open job summary: %w", err)
	}
	if _, err := f.WriteString(markdown); err != nil {
		_ = f.Close()
		return fmt.Errorf("could not write job summary: %w", err)
	}
	return f.Close()
}

// fetchLog downloads the public build log, retrying in case it is not fully available yet.
func (w *websitePreview) fetchLog(ctx context.Context, url string) ([]byte, error) {
	const attempts = 5
	var err error
	for i := range attempts {
		if i > 0 {
			log.Printf("Retrying in %s: %v", w.retryDelay, err)
			time.Sleep(w.retryDelay)
		}
		var body []byte
		if body, err = w.get(ctx, url); err == nil {
			return body, nil
		}
	}
	return nil, fmt.Errorf("could not fetch build log after %d attempts: %w", attempts, err)
}

func (w *websitePreview) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := w.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read only
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// previewURL returns the website preview URL from the build log, if it was deployed.
func previewURL(buildLog []byte) (string, bool) {
	m := previewRegexp.FindSubmatch(buildLog)
	if m == nil {
		return "", false
	}
	return string(m[1]) + "/", true
}
