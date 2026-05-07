// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// govanityurls serves Go vanity URLs.
package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

type handler struct {
	host         string
	cacheControl string
	paths        pathConfigSet
}

type pathConfig struct {
	path    string
	repo    string
	display string
	vcs     string
}

func newHandler(config []byte) (*handler, error) {
	var parsed struct {
		Host     string `yaml:"host,omitempty"`
		CacheAge *int64 `yaml:"cache_max_age,omitempty"`
		Paths    map[string]struct {
			Repo    string `yaml:"repo,omitempty"`
			Display string `yaml:"display,omitempty"`
			VCS     string `yaml:"vcs,omitempty"`
		} `yaml:"paths,omitempty"`
	}
	if err := yaml.Unmarshal(config, &parsed); err != nil {
		return nil, err
	}
	h := &handler{host: parsed.Host}
	cacheAge := int64(86400) // 24 hours (in seconds)
	if parsed.CacheAge != nil {
		cacheAge = *parsed.CacheAge
		if cacheAge < 0 {
			return nil, errors.New("cache_max_age is negative")
		}
	}
	h.cacheControl = fmt.Sprintf("public, max-age=%d", cacheAge)
	for path, e := range parsed.Paths {
		pc := pathConfig{
			path:    strings.TrimSuffix(path, "/"),
			repo:    e.Repo,
			display: e.Display,
			vcs:     e.VCS,
		}
		switch {
		case e.Display != "":
			// Already filled in.
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.display = fmt.Sprintf("%v %v/tree/main{/dir} %v/blob/main{/dir}/{file}#L{line}", e.Repo, e.Repo, e.Repo)
		case strings.HasPrefix(e.Repo, "https://bitbucket.org"):
			pc.display = fmt.Sprintf("%v %v/src/default{/dir} %v/src/default{/dir}/{file}#{file}-{line}", e.Repo, e.Repo, e.Repo)
		}
		switch {
		case e.VCS != "":
			// Already filled in.
			if e.VCS != "bzr" && e.VCS != "git" && e.VCS != "hg" && e.VCS != "svn" {
				return nil, fmt.Errorf("configuration for %v: unknown VCS %s", path, e.VCS)
			}
		case strings.HasPrefix(e.Repo, "https://github.com/"):
			pc.vcs = "git"
		default:
			return nil, fmt.Errorf("configuration for %v: cannot infer VCS from %s", path, e.Repo)
		}
		h.paths = append(h.paths, pc)
	}
	sort.Sort(h.paths)
	return h, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	current := r.URL.Path
	pc, subpath := h.paths.find(current)
	if pc == nil && current == "/" {
		h.serveIndex(w, r)
		return
	}
	if pc == nil && strings.Contains(current, "/chart/stable") {
		path := strings.Replace(current, "/chart/stable", "", 1)
		h.serveChart(w, r, path)
		return
	}
	if pc == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", h.cacheControl)
	if err := vanityTmpl.Execute(w, struct {
		Import  string
		Subpath string
		Repo    string
		Display string
		VCS     string
	}{
		Import:  h.Host(r) + pc.path,
		Subpath: subpath,
		Repo:    pc.repo,
		Display: pc.display,
		VCS:     pc.vcs,
	}); err != nil {
		http.Error(w, "cannot render the page", http.StatusInternalServerError)
	}
}

func (h *handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	// Just redirect to the first one
	// just commenting out, in case we want to soft launch
	// http.Redirect(w, r, h.paths[0].repo, http.StatusTemporaryRedirect)
	http.Redirect(w, r, "/site/", http.StatusTemporaryRedirect)
}

func (h *handler) serveChart(w http.ResponseWriter, r *http.Request, path string) {
	root := "https://storage.googleapis.com/agones-chart"
	http.Redirect(w, r, root+path, http.StatusTemporaryRedirect)
}

func (h *handler) Host(r *http.Request) string {
	host := h.host
	if host == "" {
		host = defaultHost(r)
	}
	return host
}

var vanityTmpl = template.Must(template.New("vanity").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{.Import}} {{.VCS}} {{.Repo}}">
<meta name="go-source" content="{{.Import}} {{.Display}}">
<meta http-equiv="refresh" content="0; url=https://godoc.org/{{.Import}}/{{.Subpath}}">
</head>
<body>
Nothing to see here; <a href="https://godoc.org/{{.Import}}/{{.Subpath}}">see the package on godoc</a>.
</body>
</html>`))

type pathConfigSet []pathConfig

func (pset pathConfigSet) Len() int {
	return len(pset)
}

func (pset pathConfigSet) Less(i, j int) bool {
	return pset[i].path < pset[j].path
}

func (pset pathConfigSet) Swap(i, j int) {
	pset[i], pset[j] = pset[j], pset[i]
}

func (pset pathConfigSet) find(path string) (pc *pathConfig, subpath string) {
	i := sort.Search(len(pset), func(i int) bool {
		return pset[i].path >= path
	})
	if i < len(pset) && pset[i].path == path {
		return &pset[i], ""
	}
	if i > 0 && strings.HasPrefix(path, pset[i-1].path+"/") {
		return &pset[i-1], path[len(pset[i-1].path)+1:]
	}
	return nil, ""
}
