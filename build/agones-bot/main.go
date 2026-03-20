/*
 * Copyright 2024 Google LLC
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

package main

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"text/template"

	"github.com/GoogleCloudPlatform/cloud-build-notifiers/lib/notifiers"
	"github.com/Masterminds/sprig/v3"
	log "github.com/golang/glog"
	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
	"google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

const (
	tokenSecretName = "agones-bot-pr-commenter"
	owner           = "googleforgames"
	repo            = "agones"

	// If testPR is non-zero, we ONLY respond on builds for testPR.
	// (The logs will show what we would comment for other PRs.)
	testPR = 0
)

func main() {
	if err := notifiers.Main(new(githubNotifier)); err != nil {
		log.Fatalf("fatal error: %v", err)
	}
}

type githubNotifier struct {
	client          *github.Client
	successTemplate *template.Template
	failureTemplate *template.Template
}

func (g *githubNotifier) SetUp(ctx context.Context, config *notifiers.Config, _ string, sg notifiers.SecretGetter, resolver notifiers.BindingResolver) error {
	tokenResource, err := notifiers.FindSecretResourceName(config.Spec.Secrets, tokenSecretName)
	if err != nil {
		return fmt.Errorf("failed to find Secret for ref %q: %w", tokenSecretName, err)
	}
	token, err := sg.GetSecret(ctx, tokenResource)
	if err != nil {
		return fmt.Errorf("failed to get token secret: %w", err)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	g.client = github.NewClient(tc)

	g.successTemplate, err = template.New("success").Funcs(sprig.FuncMap()).Parse(successTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse success template: %w", err)
	}

	g.failureTemplate, err = template.New("failure").Funcs(sprig.FuncMap()).Parse(failureTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse failure template: %w", err)
	}

	return nil
}

func (g *githubNotifier) SendNotification(ctx context.Context, build *cloudbuild.Build) error {
	statusName := cloudbuild.Build_Status_name[int32(build.Status)]
	log.Infof("Build Status: %s, Detail: %s", statusName, build.StatusDetail)
	if build.Status == cloudbuild.Build_QUEUED || build.Status == cloudbuild.Build_WORKING || build.Status == cloudbuild.Build_CANCELLED {
		log.Info("Skipping notification")
		return nil
	}

	if build.StatusDetail == "" {
		build.StatusDetail = statusName
	}

	// if this isn't a pull request, ignore it.
	prNumber, ok := build.Substitutions["_PR_NUMBER"]
	if !ok {
		log.Errorf("Could not find PR number in build %s", build.Id)
		return nil
	}
	log.Infof("Serving Pull Request #%s: %v", prNumber, build)

	// if we get weird data, ignore it.
	pr, err := strconv.Atoi(prNumber)
	if err != nil {
		log.Errorf("Could not convert the PR number (%s) to an integer in build %s", prNumber, build.Id)
		return nil
	}

	dryRun := false
	if testPR > 0 && pr != testPR {
		log.Infof("Not test PR #%d, dry run", testPR)
		dryRun = true
	}

	// if builds fail, exit, as it's our fault, and there is no point in retrying.
	body := new(bytes.Buffer)

	// hack to allow backticks in multiline strings
	build.Substitutions["_DELIM"] = "`"

	if build.Status == cloudbuild.Build_SUCCESS {
		if err := g.successTemplate.Execute(body, build); err != nil {
			log.Errorf("Error executing success template: %v, in build %s", err, build.Id)
			return nil
		}
	} else {
		if err := g.failureTemplate.Execute(body, build); err != nil {
			log.Errorf("Error executing failure template: %v, in build %s", err, build.Id)
			return nil
		}
	}

	commentBody := body.String()
	comment := &github.IssueComment{
		Body: &commentBody,
	}
	log.Infof("Comment: %q", commentBody)
	if !dryRun {
		if _, _, err := g.client.Issues.CreateComment(ctx, owner, repo, pr, comment); err != nil {
			log.Errorf("Error creating a comment: %v on PR %d", err, pr)
			return err
		}
	}

	return nil
}

const successTemplate = `
{{ $sha := .Substitutions.SHORT_SHA }}
{{ $pr := .Substitutions._PR_NUMBER }}
{{ $tag := (trimSuffix "-amd64" (split ":" (index .Results.Images 0).Name)._1) }}

**Build Succeeded :partying_face:**

_Build Id: {{ .Id }}_

The following development artifacts have been built, and will exist for the next 30 days:
{{ range .Results.Images }}
* image: [{{ trimSuffix "-amd64" .Name }}](https://{{ trimSuffix "-amd64" .Name }})
{{- end}}
* Linux C++ SDK (build): [agonessdk-{{ $tag }}-linux-arch_64.tar.gz](https://storage.googleapis.com/agones-artifacts/cpp-sdk/agonessdk-{{ $tag }}-linux-arch_64.tar.gz)
* SDK Server: [agonessdk-server-{{ $tag }}.zip](https://storage.googleapis.com/agones-artifacts/sdk-server/agonessdk-server-{{ $tag }}.zip)

A preview of the website (the last 30 builds are retained):
* https://{{ $sha }}-dot-preview-dot-agones-images.appspot.com/

To install this version:
` + "```" + `
git fetch https://github.com/agones-dev/agones.git pull/{{ $pr }}/head:pr_{{ $pr }} && git checkout pr_{{ $pr }}
helm install agones ./install/helm/agones --namespace agones-system --set agones.image.registry=us-docker.pkg.dev/agones-images/ci --set agones.image.tag={{ $tag }}
` + "```"

const failureTemplate = `
{{ $sha := .Substitutions.SHORT_SHA }}
{{ $delim := .Substitutions._DELIM }}
{{ $repo := .Substitutions._REPOSITORY }}

**Build Failed :sob:**

_Build Id: {{ .Id }}_

Status: {{ .StatusDetail }}

 - [Cloud Build view]({{.LogUrl}})
 - [Cloud Build log download](https://storage.googleapis.com/agones-build-logs/log-{{ .Id }}.txt)

To get permission to view the Cloud Build view, join the [agones-discuss](https://groups.google.com/forum/#!forum/agones-discuss) Google Group.
`
