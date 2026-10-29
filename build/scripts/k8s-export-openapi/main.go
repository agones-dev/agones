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

// Package main implements a program to convert from export-openapi.sh to Go script.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/itchyny/json2yaml"
)

const proxyURL = "http://127.0.0.1:8001"

func main() {
	ctx := context.Background()
	tmpDir := "../../tmp"

	if _, err := os.Stat(tmpDir); err == nil {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := os.MkdirAll(tmpDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	cmd := startKubectlProxy(ctx)

	// Fetch the OpenAPI JSON into the `tmp` directory.
	openapiFile := filepath.Join(tmpDir, "openapi.json")
	if err := downloadOpenAPI(ctx, openapiFile); err != nil {
		log.Fatal(err)
	}

	// Remove specified fields from the OpenAPI JSON.
	err = removeFields(openapiFile)
	if err != nil {
		log.Fatal(err)
	}

	// Read the modified OpenAPI JSON file.
	openAPI, err := readOpenAPI(openapiFile)
	if err != nil {
		log.Fatal(err)
	}

	doExpand("io.k8s.api.core.v1.PodTemplateSpec", tmpDir, openAPI)

	objectMetaPath := filepath.Join(tmpDir, "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.json")
	if err := makeCreationTimestampNullable(objectMetaPath); err != nil {
		log.Fatal(err)
	}

	intOrStringPath := filepath.Join(tmpDir, "io.k8s.apimachinery.pkg.util.intstr.IntOrString.json")
	if err := modifyIntOrString(intOrStringPath); err != nil {
		log.Fatal(err)
	}

	podSpecPath := filepath.Join(tmpDir, "io.k8s.api.core.v1.PodSpec.json")
	if err := escapeDoubleBackslashes(podSpecPath); err != nil {
		log.Fatal(err)
	}

	origDir := filepath.Join(tmpDir, "orig")
	err = os.Mkdir(origDir, 0o755)
	if err != nil {
		log.Fatal(err)
	}

	jsonFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range jsonFiles {
		log.Printf("Processing JSONfile: %s", f)
		src := f
		dest := filepath.Join(origDir, filepath.Base(f))
		input, err := os.ReadFile(src)
		if err != nil {
			log.Fatalf("Error reading %s: %v", src, err)
		}

		err = os.WriteFile(dest, input, 0o644)
		if err != nil {
			log.Fatalf("Error writing to %s: %v", dest, err)
		}
	}

	// Remove openapi.json from the current directory
	err = os.Remove(openapiFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	err = modifyJSONFiles(tmpDir)
	if err != nil {
		log.Fatal(err)
	}

	filenames := []string{
		"io.k8s.api.core.v1.PodTemplateSpec",
		"io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
	}

	for _, filename := range filenames {
		jsonToHelmYaml(tmpDir, filename)
	}

	stopKubectlProxy(cmd)
}

// startKubectlProxy stops any kubectl proxy that is already running, then starts
// a new one and waits for it to serve requests.
func startKubectlProxy(ctx context.Context) *exec.Cmd {
	// Check if kubectl proxy is running
	if _, err := exec.CommandContext(ctx, "pgrep", "-f", "kubectl proxy").Output(); err == nil {
		// If running, terminate it
		if err := exec.CommandContext(ctx, "pkill", "-f", "kubectl proxy").Run(); err != nil {
			log.Fatalf("Failed to terminate existing kubectl proxy: %v", err)
		}
		time.Sleep(3 * time.Second)
	}

	// Start the kubectl proxy.
	cmd := exec.CommandContext(ctx, "kubectl", "proxy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start kubectl proxy: %v", err)
	}

	maxRetries := 10
	ready := false
	for range maxRetries {
		time.Sleep(1 * time.Second)
		if proxyReady(ctx) {
			ready = true
			break
		}
	}

	if !ready {
		log.Fatalf("kubectl proxy not ready after waiting for %d seconds", maxRetries)
	}

	// Sleep for 5 seconds to ensure proxy is ready
	time.Sleep(5 * time.Second)

	return cmd
}

// proxyReady reports whether the kubectl proxy answers with 200 OK.
func proxyReady(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL+"/", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode == http.StatusOK
}

// downloadOpenAPI saves the cluster's OpenAPI v2 spec, fetched through the
// kubectl proxy, to path.
func downloadOpenAPI(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL+"/openapi/v2", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return errors.Join(err, file.Close())
}

// readOpenAPI decodes the OpenAPI JSON at path.
func readOpenAPI(path string) (map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	var openAPI map[string]any
	err = json.NewDecoder(file).Decode(&openAPI)
	return openAPI, err
}

// stopKubectlProxy interrupts the kubectl proxy and waits for it to exit.
func stopKubectlProxy(cmd *exec.Cmd) {
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to interrupt kubectl proxy: %v", err)
	}
	time.Sleep(1 * time.Second)
	err := cmd.Wait()
	if err == nil {
		return
	}
	// Check if the error is due to the process being interrupted
	exitError := &exec.ExitError{}
	if errors.As(err, &exitError) {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() && status.Signal() == syscall.SIGINT {
				log.Println("kubectl proxy has been gracefully shutdown")
				return
			}
		}
	}
	log.Fatalf("Failed to gracefully shutdown kubectl proxy: %v", err)
}

func removeFields(filename string) error {
	openAPI, err := readOpenAPI(filename)
	if err != nil {
		return err
	}

	keysToRemove := []string{
		"x-kubernetes-list-type",
		"x-kubernetes-group-version-kind",
		"x-kubernetes-list-map-keys",
		"x-kubernetes-unions",
	}

	removeKeysRecursively(openAPI, keysToRemove)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(openAPI)
	return errors.Join(err, file.Close())
}

func removeKeysRecursively(obj any, keysToRemove []string) {
	switch v := obj.(type) {
	case map[string]any:
		for _, key := range keysToRemove {
			delete(v, key)
		}
		for _, value := range v {
			removeKeysRecursively(value, keysToRemove)
		}
	case []any:
		for _, item := range v {
			removeKeysRecursively(item, keysToRemove)
		}
	}
}

func doExpand(key, tmpDir string, openapi map[string]any) {
	log.Println("Processing", key)

	// Extract and save the definitions
	definitions, ok := openapi["definitions"].(map[string]any)
	if !ok {
		log.Println("Error: 'definitions' is not a map[string]interface{} type")
		return
	}

	section, exists := definitions[key]
	if !exists {
		log.Printf("Key %s not found in definitions\n", key)
		return
	}

	sectionMap, ok := section.(map[string]any)
	if !ok {
		log.Printf("Error: section for key '%s' is not a map[string]interface{} type\n", key)
		return
	}

	sectionJSON, err := json.MarshalIndent(sectionMap, "", "  ")
	if err != nil {
		log.Println("Error marshaling section:", err)
		return
	}

	err = os.WriteFile(filepath.Join(tmpDir, key+".json"), sectionJSON, 0o644)
	if err != nil {
		log.Println("Error writing to file:", err)
		return
	}

	// Recursively search for $ref values
	var children []string
	err = searchForRefs(section, &children)
	if err != nil {
		log.Println("Error searching for references:", err)
		return
	}

	for _, child := range children {
		doExpand(child, tmpDir, openapi)
	}
}

func searchForRefs(obj any, refs *[]string) error {
	switch v := obj.(type) {
	case map[string]any:
		if ref, ok := v["$ref"]; ok {
			refStr := ref.(string)
			refStr = strings.TrimPrefix(refStr, "#/definitions/")
			*refs = append(*refs, refStr)
		}

		for _, value := range v {
			err := searchForRefs(value, refs)
			if err != nil {
				return err
			}
		}
	case []any:
		for _, item := range v {
			err := searchForRefs(item, refs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func makeCreationTimestampNullable(filePath string) error {
	var obj map[string]any

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if properties, ok := obj["properties"].(map[string]any); ok {
		if timestamp, ok := properties["creationTimestamp"].(map[string]any); ok {
			timestamp["nullable"] = true
		}
	}

	modifiedData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, modifiedData, 0o644)
}

func modifyIntOrString(filePath string) error {
	var obj map[string]any

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	obj["x-kubernetes-int-or-string"] = true
	delete(obj, "type")

	modifiedData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, modifiedData, 0o644)
}

func escapeDoubleBackslashes(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	modifiedData := strings.ReplaceAll(string(data), "\\\\", "\\\\\\\\")
	return os.WriteFile(filePath, []byte(modifiedData), 0o644)
}

// patchKeys are OpenAPI extensions that are not valid CustomResourceDefinition schema fields, so
// including them results in "unknown field" warnings on install, and errors on Kubernetes
// distributions that decode strictly. They are wrapped in a Helm conditional so they can be opted
// into by tools that make use of them, such as Kustomize.
var patchKeys = []string{
	"x-kubernetes-patch-merge-key:",
	"x-kubernetes-patch-strategy:",
}

// wrapPatchKeys wraps each run of consecutive, equally indented patchKeys lines in a Helm
// conditional. The conditional is emitted at column zero with whitespace trimming, so it renders
// cleanly at any indentation depth.
func wrapPatchKeys(yamlContent string) string {
	lines := strings.Split(yamlContent, "\n")
	out := make([]string, 0, len(lines))

	for i := 0; i < len(lines); i++ {
		if !isPatchKeyLine(lines[i]) {
			out = append(out, lines[i])
			continue
		}

		// Extend the run over following lines that are patch keys at the same indentation.
		indent := leadingSpaces(lines[i])
		j := i + 1
		for j < len(lines) && isPatchKeyLine(lines[j]) && leadingSpaces(lines[j]) == indent {
			j++
		}

		out = append(out, "{{- if .includeKubernetesPatchKeys }}")
		out = append(out, lines[i:j]...)
		out = append(out, "{{- end }}")
		// We'll continue from j, since this is where we ended up after at the end of the j loop
		// which looked for the closing line for the template condition.
		i = j - 1
	}

	return strings.Join(out, "\n")
}

func isPatchKeyLine(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	for _, key := range patchKeys {
		if strings.HasPrefix(trimmed, key) {
			return true
		}
	}
	return false
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func jsonToHelmYaml(tmpDir, filename string) {
	jsonFilePath := filepath.Join(tmpDir, filename+".json")
	log.Println("JSONFILEPATH: ", jsonFilePath)
	jsonContent, err := os.ReadFile(jsonFilePath)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %s", err)
	}

	// Convert JSON content to YAML
	jsonReader := bytes.NewReader(jsonContent)
	var yamlBuffer bytes.Buffer
	if err := json2yaml.Convert(&yamlBuffer, jsonReader); err != nil {
		log.Fatalf("Failed to convert JSON to YAML: %s", err)
	}
	yamlContent := wrapPatchKeys(yamlBuffer.String())

	// Read boilerplate
	boilerplateContent, err := os.ReadFile("../../boilerplate.yaml.txt")
	if err != nil {
		log.Fatalf("Failed to read boilerplate.yaml.txt: %s", err)
	}

	// Concatenate content to form the final YAML
	finalContent := `---
` + string(boilerplateContent) + `{{- define "` + filename + `" }}
` + yamlContent + `
{{- end }}
`

	outputPath := filepath.Join("..", "..", "..", "install", "helm", "agones", "templates", "crds", "k8s", "_"+filename+".yaml")
	if err := os.WriteFile(outputPath, []byte(finalContent), 0o644); err != nil {
		log.Fatalf("Failed to write to file %s: %s", outputPath, err)
	}
}

func modifyJSONFiles(tmpDir string) error {
	jsonFiles, err := filepath.Glob(filepath.Join(tmpDir, "*.json"))
	if err != nil {
		return err
	}

	// Map to cache the content of loaded JSON files.
	fileCache := make(map[string][]byte)

	for _, f := range jsonFiles {
		log.Printf("Expanding %s", f)

		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}

		var jsonData map[string]any
		err = json.Unmarshal(content, &jsonData)
		if err != nil {
			log.Printf("Error parsing JSON from %s. Content:\n%s", f, string(content))
			return err
		}

		// Recursive function to resolve references.
		var resolveRef func(data map[string]any) error
		resolveRef = func(data map[string]any) error {
			for _, value := range data {
				v, isMap := value.(map[string]any)
				if !isMap {
					continue
				}
				ref, ok := v["$ref"].(string)
				if ok {
					parts := strings.Split(ref, "/")
					refName := parts[len(parts)-1]
					refContent, cached := fileCache[refName]
					if !cached {
						for _, file := range jsonFiles {
							if strings.HasSuffix(file, refName+".json") {
								refContent, err = os.ReadFile(file)
								if err != nil {
									return err
								}
								fileCache[refName] = refContent
								break
							}
						}
					}
					var refData map[string]any
					err := json.Unmarshal(refContent, &refData)
					if err != nil {
						return err
					}

					delete(refData, "description")

					// Merge refData into v rather than replacing v.
					maps.Copy(v, refData)

					delete(v, "$ref") // remove $ref key after merging

					// Recursive call to resolve references within this reference.
					err = resolveRef(refData)
					if err != nil {
						return err
					}
				} else {
					// Recursive call to check nested maps.
					err := resolveRef(v)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}

		err = resolveRef(jsonData)
		if err != nil {
			return err
		}

		// Marshal the modified JSON and write back to the file.
		modifiedContent, err := json.MarshalIndent(jsonData, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(f, modifiedContent, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}
