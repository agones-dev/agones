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
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/itchyny/json2yaml"
)

func main() {
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

	// Check if kubectl proxy is running
	_, err = exec.Command("pgrep", "-f", "kubectl proxy").Output()
	if err == nil {
		// If running, terminate it
		errTerminate := exec.Command("pkill", "-f", "kubectl proxy").Run()
		if errTerminate != nil {
			log.Fatalf("Failed to terminate existing kubectl proxy: %v", errTerminate)
		}
		time.Sleep(3 * time.Second)
	}

	// Start the kubectl proxy.
	cmd := exec.Command("kubectl", "proxy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start kubectl proxy: %v", err)
	}

	maxRetries := 10
	retryCount := 0
	for retryCount < maxRetries {
		time.Sleep(1 * time.Second)
		resp, err := http.Get("http://127.0.0.1:8001/")
		if err == nil && resp.StatusCode == 200 {
			break
		}
		retryCount++
	}

	if retryCount == maxRetries {
		log.Fatalf("kubectl proxy not ready after waiting for %d seconds", maxRetries)
	}

	// Sleep for 5 seconds to ensure proxy is ready
	time.Sleep(5 * time.Second)

	// Make an HTTP request to fetch the OpenAPI JSON.
	resp, err := http.Get("http://127.0.0.1:8001/openapi/v2")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Create a file to store the OpenAPI JSON in the `tmp` directory.
	openapiFile := filepath.Join(tmpDir, "openapi.json")
	file, err := os.Create(openapiFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Copy the HTTP response body to the file.
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Remove specified fields from the OpenAPI JSON.
	err = removeFields(openapiFile)
	if err != nil {
		log.Fatal(err)
	}

	// Read the modified OpenAPI JSON file.
	file, err = os.Open(openapiFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var openAPI map[string]interface{}
	err = json.NewDecoder(file).Decode(&openAPI)
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
	err = os.Mkdir(origDir, 0755)
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

	cmd.Process.Signal(os.Interrupt)
	time.Sleep(1 * time.Second)
	err = cmd.Wait()
	if err != nil {
		// Check if the error is due to the process being interrupted
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() && status.Signal() == syscall.SIGINT {
					log.Println("kubectl proxy has been gracefully shutdown")
					return
				}
			}
		}
		log.Fatalf("Failed to gracefully shutdown kubectl proxy: %v", err)
	}

}

func removeFields(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var openAPI map[string]interface{}
	err = json.NewDecoder(file).Decode(&openAPI)
	if err != nil {
		return err
	}

	keysToRemove := []string{
		"x-kubernetes-patch-strategy",
		"x-kubernetes-patch-merge-key",
		"x-kubernetes-list-type",
		"x-kubernetes-group-version-kind",
		"x-kubernetes-list-map-keys",
		"x-kubernetes-unions",
	}

	removeKeysRecursively(openAPI, keysToRemove)

	file, err = os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(openAPI)
	if err != nil {
		return err
	}

	return nil
}

func removeKeysRecursively(obj interface{}, keysToRemove []string) {
	switch v := obj.(type) {
	case map[string]interface{}:
		for _, key := range keysToRemove {
			delete(v, key)
		}
		for _, value := range v {
			removeKeysRecursively(value, keysToRemove)
		}
	case []interface{}:
		for _, item := range v {
			removeKeysRecursively(item, keysToRemove)
		}
	}
}

func doExpand(key, tmpDir string, openapi map[string]interface{}) {
	log.Println("Processing", key)

	// Extract and save the definitions
	definitions, ok := openapi["definitions"].(map[string]interface{})
	if !ok {
		log.Println("Error: 'definitions' is not a map[string]interface{} type")
		return
	}

	section, exists := definitions[key]
	if !exists {
		log.Printf("Key %s not found in definitions\n", key)
		return
	}

	sectionMap, ok := section.(map[string]interface{})
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

func searchForRefs(obj interface{}, refs *[]string) error {
	switch v := obj.(type) {
	case map[string]interface{}:
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
	case []interface{}:
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
	var obj map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if properties, ok := obj["properties"].(map[string]interface{}); ok {
		if timestamp, ok := properties["creationTimestamp"].(map[string]interface{}); ok {
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
	var obj map[string]interface{}

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
	yamlContent := yamlBuffer.String()

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

	outputPath := filepath.Join("../../../install/helm/agones/templates/crds/k8s", "_"+filename+".yaml")
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

		var jsonData map[string]interface{}
		err = json.Unmarshal(content, &jsonData)
		if err != nil {
			log.Printf("Error parsing JSON from %s. Content:\n%s", f, string(content))
			return err
		}

		// Recursive function to resolve references.
		var resolveRef func(data map[string]interface{}) error
		resolveRef = func(data map[string]interface{}) error {
			for _, value := range data {
				switch v := value.(type) {
				case map[string]interface{}:
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
						var refData map[string]interface{}
						err := json.Unmarshal(refContent, &refData)
						if err != nil {
							return err
						}

						delete(refData, "description")

						// Merge refData into v rather than replacing v.
						for refKey, refValue := range refData {
							v[refKey] = refValue
						}

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
