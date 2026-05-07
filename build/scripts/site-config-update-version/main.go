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

// Package main implements a program that updates the release version and sync data between dev and prod in site/config.toml file.
package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml"
)

// The Config struct holds two nested structs, DevConfig and ProdConfig, which represent the current and previous version of Kubernetes.
type Config struct {
	DevConfig struct {
		DevGKEExampleClusterVersion      string   `toml:"dev_gke_example_cluster_version"`
		DevAKSExampleClusterVersion      string   `toml:"dev_aks_example_cluster_version"`
		DevEKSExampleClusterVersion      string   `toml:"dev_eks_example_cluster_version"`
		DevMinikubeExampleClusterVersion string   `toml:"dev_minikube_example_cluster_version"`
		DevK8sAPIVersion                 string   `toml:"dev_k8s_api_version"`
		DevSupportedK8s                  []string `toml:"dev_supported_k8s"`
	} `toml:"params"`

	ProdConfig struct {
		K8sAPIVersion                 string   `toml:"k8s_api_version"`
		GKEExampleClusterVersion      string   `toml:"gke_example_cluster_version"`
		AKSExampleClusterVersion      string   `toml:"aks_example_cluster_version"`
		EKSExampleClusterVersion      string   `toml:"eks_example_cluster_version"`
		MinikubeExampleClusterVersion string   `toml:"minikube_example_cluster_version"`
		SupportedK8s                  []string `toml:"supported_k8s"`
	} `toml:"params"`
}

func main() {
	var releaseStage, configFile string
	flag.StringVar(&releaseStage, "release-stage", "minor", "Specify the release stage ('minor' or 'patch')")
	flag.StringVar(&configFile, "config-file", "site/config.toml", "Path to the config.toml file")
	flag.Parse()

	if err := updateSiteConfig(configFile, releaseStage); err != nil {
		log.Fatalf("Error: %v", err)
	}
	log.Println("Values updated and saved successfully!")
}

func updateSiteConfig(filePath, stage string) error {
	if stage != "minor" && stage != "patch" {
		log.Fatalf("invalid release stage: must be 'minor' or 'patch'")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var updatedLines []string
	if stage == "minor" {
		var config Config
		err = toml.Unmarshal(content, &config)
		if err != nil {
			return err
		}

		// Copy values from dev to prod
		config.ProdConfig.SupportedK8s = config.DevConfig.DevSupportedK8s
		config.ProdConfig.K8sAPIVersion = config.DevConfig.DevK8sAPIVersion
		config.ProdConfig.GKEExampleClusterVersion = config.DevConfig.DevGKEExampleClusterVersion
		config.ProdConfig.AKSExampleClusterVersion = config.DevConfig.DevAKSExampleClusterVersion
		config.ProdConfig.EKSExampleClusterVersion = config.DevConfig.DevEKSExampleClusterVersion
		config.ProdConfig.MinikubeExampleClusterVersion = config.DevConfig.DevMinikubeExampleClusterVersion

		var lines []string
		for _, line := range strings.Split(string(content), "\n") {
			switch {
			case strings.HasPrefix(line, "supported_k8s"):
				line = "supported_k8s = " + tomlArray(config.ProdConfig.SupportedK8s)
			case strings.HasPrefix(line, "k8s_api_version"):
				line = "k8s_api_version = \"" + config.ProdConfig.K8sAPIVersion + "\""
			case strings.HasPrefix(line, "gke_example_cluster_version"):
				line = "gke_example_cluster_version = \"" + config.ProdConfig.GKEExampleClusterVersion + "\""
			case strings.HasPrefix(line, "aks_example_cluster_version"):
				line = "aks_example_cluster_version = \"" + config.ProdConfig.AKSExampleClusterVersion + "\""
			case strings.HasPrefix(line, "eks_example_cluster_version"):
				line = "eks_example_cluster_version = \"" + config.ProdConfig.EKSExampleClusterVersion + "\""
			case strings.HasPrefix(line, "minikube_example_cluster_version"):
				line = "minikube_example_cluster_version = \"" + config.ProdConfig.MinikubeExampleClusterVersion + "\""
			}
			lines = append(lines, line)
		}
		updatedLines = updateReleaseValues(lines, stage)
	} else {
		updatedLines = updateReleaseValues(strings.Split(string(content), "\n"), stage)
	}

	return writeLinesToFile(filePath, updatedLines)
}

// Helper function to convert an array of strings to a TOML array representation
func tomlArray(values []string) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i, value := range values {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("\"")
		builder.WriteString(value)
		builder.WriteString("\"")
	}
	builder.WriteString("]")
	return builder.String()
}

func updateReleaseValues(lines []string, stage string) []string {
	var updatedLines []string

	for _, line := range lines {
		updatedLine := line
		trimmedLine := strings.TrimSpace(line) // Trim leading/trailing spaces

		if strings.HasPrefix(trimmedLine, "release_branch") || strings.HasPrefix(trimmedLine, "release_version") {
			if stage == "minor" {
				updatedLine = incrementMinorVersion(line)
			} else {
				updatedLine = incrementPatchVersion(line)
			}
		}

		updatedLines = append(updatedLines, updatedLine)
	}

	return updatedLines
}

// Increments the minor version, and resets the patch version (if any) to 0
// ex: 1.52.0 -> 1.53.0 or 1.52.1 -> 1.53.0
func incrementMinorVersion(line string) string {
	re := regexp.MustCompile(`"([^"]+)"`)
	match := re.FindStringSubmatch(line)
	if len(match) > 1 {
		version := match[1]
		segments := strings.Split(version, ".")
		if len(segments) == 3 {
			secondSegment, err := strconv.Atoi(segments[1])
			if err != nil {
				return line
			}
			secondSegment++
			segments[1] = strconv.Itoa(secondSegment)
			segments[2] = "0" // Reset patch version for minor release
			updatedVersion := strings.Join(segments, ".")
			return strings.Replace(line, version, updatedVersion, 1)
		}
	}
	return line
}

func incrementPatchVersion(line string) string {
	re := regexp.MustCompile(`"([^"]+)"`)
	match := re.FindStringSubmatch(line)
	if len(match) > 1 {
		version := match[1]
		segments := strings.Split(version, ".")
		if len(segments) == 3 {
			patchSegment, err := strconv.Atoi(segments[2])
			if err != nil {
				return line
			}
			patchSegment++
			segments[2] = strconv.Itoa(patchSegment)
			updatedVersion := strings.Join(segments, ".")
			return strings.Replace(line, version, updatedVersion, 1)
		}
	}
	return line
}

func writeLinesToFile(filePath string, lines []string) error {
	// Open the file in write mode
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			log.Println(cerr)
		}
	}()

	// Create a writer to write to the file
	writer := bufio.NewWriter(file)

	// Write the lines to the file
	for i, line := range lines {
		// Avoid adding a new line at the end of the file
		if i < len(lines)-1 {
			_, err := writer.WriteString(line + "\n")
			if err != nil {
				return err
			}
		} else {
			_, err := writer.WriteString(line)
			if err != nil {
				return err
			}
		}
	}

	// Flush the writer to ensure all data is written to the file
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}
