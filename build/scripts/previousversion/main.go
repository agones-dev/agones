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
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	semver "github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

// gcsLister defines an interface for listing GCS objects, allowing for mocking in tests.
type gcsLister interface {
	List(prefix string) (string, error)
}

// gsutilLister is the real implementation of gcsLister that calls the gsutil command.
type gsutilLister struct{}

func (g gsutilLister) List(prefix string) (string, error) {
	cmd := exec.Command("gsutil", "ls", prefix+"*.tgz")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", errors.Errorf("gsutil command failed: %v: %s", err, stderr.String())
	}
	return out.String(), nil
}

func main() {
	baseVersion := flag.String("version", "", "the base version to be taken in as input for release")
	flag.Parse()

	if *baseVersion == "" {
		log.Fatal("version flag must be provided")
	}

	v, err := semver.Parse(*baseVersion)
	if err != nil {
		log.Fatalf("Error parsing version: %v", err)
	}

	lister := gsutilLister{}
	prevVersion, err := getPreviousVersion(v, lister)
	if err != nil {
		log.Fatalf("Error getting previous version: %v", err)
	}

	// The SERVICE name for the site deployment uses dashes
	releaseVersion := fmt.Sprintf("%d-%d-%d", prevVersion.Major, prevVersion.Minor, prevVersion.Patch)
	fmt.Println(releaseVersion)
}

func getPreviousVersion(v semver.Version, lister gcsLister) (semver.Version, error) {
	var prevVersion semver.Version

	if v.Patch == 0 {
		// This is a MINOR release (e.g., 1.53.0). Find the latest patch of the previous minor (e.g., 1.52.x).
		if v.Minor == 0 {
			return semver.Version{}, fmt.Errorf("cannot determine previous minor version for a major release: %s", v.String())
		}
		prevMinor := v.Minor - 1

		prefix := fmt.Sprintf("gs://agones-chart/agones-%d.%d.", v.Major, prevMinor)
		output, err := lister.List(prefix)
		if err != nil {
			// If we can't find any, fall back to assuming .0 patch.
			log.Printf("gsutil command failed (%v), falling back to .0 patch for version %d.%d", err, v.Major, prevMinor)
			prevVersion = semver.Version{Major: v.Major, Minor: prevMinor, Patch: 0}
		} else {
			prevVersion = getLatestVersionFromGsutil(output, v.Major, prevMinor)
		}

	} else {
		// This is a PATCH release (e.g., 1.52.2 -> 1.52.1)
		prevVersion = semver.Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch - 1}
	}

	return prevVersion, nil
}

// getLatestVersionFromGsutil parses the output of `gsutil ls` to find the highest semver version.
func getLatestVersionFromGsutil(gsutilOutput string, major, minor uint64) semver.Version {
	lines := strings.Split(strings.TrimSpace(gsutilOutput), "\n")
	versions := []semver.Version{}

	// Example line: gs://agones-chart/agones-1.52.2.tgz
	re := regexp.MustCompile(fmt.Sprintf(`agones-%d\.%d\.(\d+)\.tgz$`, major, minor))

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			patch, _ := strconv.ParseUint(matches[1], 10, 64)
			versions = append(versions, semver.Version{Major: major, Minor: minor, Patch: patch})
		}
	}

	if len(versions) == 0 {
		// Fallback if no matching versions are found
		return semver.Version{Major: major, Minor: minor, Patch: 0}
	}

	// Sort versions to find the latest
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LT(versions[j])
	})

	return versions[len(versions)-1]
}
