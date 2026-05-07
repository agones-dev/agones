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

// build/report/report.go generates a flake report for the last N weeks
// on the configured build trigger. It creates files in `tmp/report`, a
// dated YYYY-MM-DD.html report and an index.html with a redirect to the
// new date, intended to upload to Google Cloud Storage.
package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	cloudbuildpb "cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"google.golang.org/api/iterator"
)

const (
	window             = time.Hour * 24 * 7 * 4 // 4 weeks
	wantBuildTriggerID = "da003bb8-e9bb-4983-a556-e77fb92f17ca"
	outPath            = "tmp/report"

	reportTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Flake Report From {{ .WindowStart }} to {{ .WindowEnd }}</title>
    <link rel="stylesheet" href="https://cdn.simplecss.org/simple.min.css">
</head>
<body>
	<header>
	Flake Report {{ .WindowStart }} to {{ .WindowEnd }}
	</header>
	<p><b>Flake Ratio:</b> {{ printf "%.3f" .FlakeRatio }} ({{ .FlakeCount }} flakes / {{ .BuildCount }} successful builds)</p>

	<table>
		<tr>
			<th>Time</th>
			<th>Flaky Build</th>
		</tr>
{{- range .Flakes -}}
		<tr>
			<td>{{ .CreateTime }}</td>
			<td><a href="https://console.cloud.google.com/cloud-build/builds;region=global/{{ .ID }}?project=agones-images">{{ .ID }}</a></td>
		</tr>
{{- end -}}
	</table>

	<p><b>Methodology:</b> For every successful build of a given commit hash, we count the number of failed
	builds on the same commit hash. The <em>Flake Ratio</em> is the ratio of flakes to successes, giving an
	expected value for how many times a build has to be retried before succeeding, on the same SHA.
	This methodology only covers manual re-runs in Cloud Build - builds retried via rebase are not counted,
	as in general it's difficult to attribute flakes between commit hashes.	
</body>
</html>
`

	redirectTemplate = `
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Latest build</title>
    <meta http-equiv="refresh" content="0;URL='https://agones-build-reports.storage.googleapis.com/{{ .Date }}.html'" />
  </head>
  <body>
    <p><a href="https://agones-build-reports.storage.googleapis.com/{{ .Date }}.html">Latest build report (redirecting now).</a></p>
  </body>
</html>  
`
)

type report struct {
	WindowStart string
	WindowEnd   string
	Flakes      []flake
	FlakeCount  int
	BuildCount  int
	FlakeRatio  float32
}

type flake struct {
	ID         string
	CreateTime string
}

type redirect struct {
	Date string
}

func main() {
	ctx := context.Background()
	reportTmpl := newReportTemplate()
	redirTmpl := newRedirectTemplate()

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-window)
	date := windowEnd.Format("2006-01-02")

	err := os.MkdirAll(outPath, 0o755)
	if err != nil {
		log.Fatalf("failed to create output path %v: %v", outPath, err)
	}

	datePath := fmt.Sprintf("%s/%s.html", outPath, date)
	reportFile, err := os.OpenFile(datePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatalf("failed to open report %v: %v", datePath, err)
	}

	redirPath := fmt.Sprintf("%s/index.html", outPath)
	redirFile, err := os.OpenFile(redirPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		log.Fatalf("failed to open redirect %v: %v", redirPath, err)
	}

	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to initialize cloudbuild client: %v", err)
	}

	success := make(map[string]bool)     // build SHA -> bool
	failure := make(map[string][]string) // build SHA -> slice of build IDs that failed
	idTime := make(map[string]time.Time) // build ID -> create time

	// See https://pkg.go.dev/cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb#ListBuildsRequest.
	req := &cloudbuildpb.ListBuildsRequest{
		ProjectId: "agones-images",
		// TODO(zmerlynn): No idea why this is failing.
		// Filter:    `build_trigger_id = "da003bb8-e9bb-4983-a556-e77fb92f17ca"`,
	}
	it := c.ListBuilds(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("error listing builds: %v", err)
		}
		createTime := resp.CreateTime.AsTime()
		if createTime.Before(windowStart) {
			break
		}
		// We only care about Agones builds.
		if resp.BuildTriggerId != wantBuildTriggerID {
			continue
		}
		// Ignore if it's still running.
		if resp.FinishTime == nil {
			continue
		}

		id := resp.Id
		sha := resp.Substitutions["COMMIT_SHA"]
		status := resp.Status
		idTime[id] = createTime
		log.Printf("id = %v, sha = %v, status = %v", id, sha, status)

		// Record clear cut success/failure, not timeout, cancelled, etc.
		switch status {
		case cloudbuildpb.Build_SUCCESS:
			success[sha] = true
		case cloudbuildpb.Build_FAILURE:
			failure[sha] = append(failure[sha], id)
		default:
			continue
		}
	}

	buildCount := len(success)
	flakeCount := 0
	var flakes []flake
	for sha := range success {
		flakeCount += len(failure[sha])
		for _, id := range failure[sha] {
			flakes = append(flakes, flake{
				ID:         id,
				CreateTime: idTime[id].Format(time.RFC3339),
			})
		}
	}
	sort.Slice(flakes, func(i, j int) bool { return flakes[i].CreateTime > flakes[j].CreateTime })

	if err := reportTmpl.Execute(reportFile, report{
		WindowStart: windowStart.Format("2006-01-02"),
		WindowEnd:   windowEnd.Format("2006-01-02"),
		BuildCount:  buildCount,
		FlakeCount:  flakeCount,
		FlakeRatio:  float32(flakeCount) / float32(buildCount),
		Flakes:      flakes,
	}); err != nil {
		log.Fatalf("failure rendering report: %v", err)
	}

	if err := redirTmpl.Execute(redirFile, redirect{Date: date}); err != nil {
		log.Fatalf("failure rendering redirect: %v", err)
	}
}

func newReportTemplate() *template.Template {
	return template.Must(template.New("report").Parse(reportTemplate)).Option("missingkey=error")
}

func newRedirectTemplate() *template.Template {
	return template.Must(template.New("redir").Parse(redirectTemplate)).Option("missingkey=error")
}
