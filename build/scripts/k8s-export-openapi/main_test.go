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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFieldsKeepsPatchKeys(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"definitions": map[string]any{
			"io.k8s.api.core.v1.Container": map[string]any{
				"properties": map[string]any{
					"env": map[string]any{
						"type":                         "array",
						"x-kubernetes-patch-strategy":  "merge",
						"x-kubernetes-patch-merge-key": "name",
						"x-kubernetes-list-type":       "map",
						"x-kubernetes-list-map-keys":   []any{"name"},
					},
				},
				"x-kubernetes-group-version-kind": []any{
					map[string]any{"group": "", "version": "v1", "kind": "Pod"},
				},
				"x-kubernetes-unions": []any{},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.json")
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	if err := removeFields(path); err != nil {
		t.Fatalf("removeFields: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	defs := output["definitions"].(map[string]any)
	container := defs["io.k8s.api.core.v1.Container"].(map[string]any)
	props := container["properties"].(map[string]any)
	env := props["env"].(map[string]any)

	if got := env["x-kubernetes-patch-strategy"]; got != "merge" {
		t.Errorf("x-kubernetes-patch-strategy = %v, want merge", got)
	}
	if got := env["x-kubernetes-patch-merge-key"]; got != "name" {
		t.Errorf("x-kubernetes-patch-merge-key = %v, want name", got)
	}
	if _, ok := env["x-kubernetes-list-type"]; ok {
		t.Errorf("x-kubernetes-list-type should be removed")
	}
	if _, ok := env["x-kubernetes-list-map-keys"]; ok {
		t.Errorf("x-kubernetes-list-map-keys should be removed")
	}
	if _, ok := container["x-kubernetes-group-version-kind"]; ok {
		t.Errorf("x-kubernetes-group-version-kind should be removed")
	}
	if _, ok := container["x-kubernetes-unions"]; ok {
		t.Errorf("x-kubernetes-unions should be removed")
	}
}

func TestWrapPatchKeys(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		in   string
		want string
	}{
		"adjacent keys share a conditional": {
			in: "    type: array\n" +
				"    x-kubernetes-patch-merge-key: uid\n" +
				"    x-kubernetes-patch-strategy: merge\n" +
				"  resourceVersion:\n",
			want: "    type: array\n" +
				"{{- if .includeKubernetesPatchKeys }}\n" +
				"    x-kubernetes-patch-merge-key: uid\n" +
				"    x-kubernetes-patch-strategy: merge\n" +
				"{{- end }}\n" +
				"  resourceVersion:\n",
		},
		"standalone key": {
			in: "    type: array\n" +
				"    x-kubernetes-patch-strategy: merge\n",
			want: "    type: array\n" +
				"{{- if .includeKubernetesPatchKeys }}\n" +
				"    x-kubernetes-patch-strategy: merge\n" +
				"{{- end }}\n",
		},
		"runs at differing indentation are not merged": {
			in: "        x-kubernetes-patch-strategy: merge\n" +
				"    x-kubernetes-patch-strategy: merge,retainKeys\n",
			want: "{{- if .includeKubernetesPatchKeys }}\n" +
				"        x-kubernetes-patch-strategy: merge\n" +
				"{{- end }}\n" +
				"{{- if .includeKubernetesPatchKeys }}\n" +
				"    x-kubernetes-patch-strategy: merge,retainKeys\n" +
				"{{- end }}\n",
		},
		"other extensions are left alone": {
			in: "      type: object\n" +
				"      x-kubernetes-map-type: atomic\n" +
				"      x-kubernetes-int-or-string: true\n",
			want: "      type: object\n" +
				"      x-kubernetes-map-type: atomic\n" +
				"      x-kubernetes-int-or-string: true\n",
		},
		"no patch keys": {
			in:   "description: a description\ntype: object\n",
			want: "description: a description\ntype: object\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := wrapPatchKeys(tc.in); got != tc.want {
				t.Errorf("wrapPatchKeys() =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
