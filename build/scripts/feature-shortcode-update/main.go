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

// Package main implements a program that removes the contents of feature expiry version and publish version shortcodes in .md files within the site/content/en/docs directory.
package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	expiryOpenPercentRe   = regexp.MustCompile(`\{\{% feature expiryVersion="([^"]+)" %\}\}`)
	expiryOpenAngleRe     = regexp.MustCompile(`\{\{< feature expiryVersion="([^"]+)" >\}\}`)
	publishOpenPercentRe  = regexp.MustCompile(`\{\{% feature publishVersion="([^"]+)" %\}\}`)
	publishOpenAngleRe    = regexp.MustCompile(`\{\{< feature publishVersion="([^"]+)" >\}\}`)
	featureClosePercentRe = regexp.MustCompile(`\{\{% /feature %\}\}`)
	featureCloseAngleRe   = regexp.MustCompile(`\{\{< /feature >\}\}`)
)

const (
	mdExt   = ".md"
	htmlExt = ".html"
)

func main() {
	dirPath := "site/content/en/docs"

	version := flag.String("version", "", "Expiry version to remove")
	flag.Parse()

	// Check if the version is provided
	if *version == "" {
		log.Fatal("Version not specified. Please provide a value for the -version flag in CLI.")
	}

	modifiedFiles := 0

	err := filepath.WalkDir(dirPath, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		ext := filepath.Ext(d.Name())
		if d.IsDir() || (ext != mdExt && ext != htmlExt) {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		scanner := bufio.NewScanner(file)
		modifiedContent := removeBlocks(scanner, *version, ext)

		// Only write the modified content back to the .md file if there are changes
		if modifiedContent != "" {
			outputFile, err := os.Create(filePath)
			if err != nil {
				log.Fatal(err)
			}
			defer func() {
				if err := outputFile.Close(); err != nil {
					log.Fatal(err)
				}
			}()

			writer := bufio.NewWriter(outputFile)
			_, err = writer.WriteString(modifiedContent)
			if err != nil {
				log.Fatal(err)
			}

			// Flush the writer to ensure all content is written
			if err := writer.Flush(); err != nil {
				log.Fatal(err)
			}

			log.Printf("Processed file: %s\n", filePath)
			modifiedFiles++
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	if modifiedFiles == 0 {
		log.Println("There are no files with feature expiryVersion or publishVersion shortcodes")
	}
}

// removeBlocks assumes feature shortcodes never nest — a {{%|{{< /feature %}}|>}} always
// closes the single innermost open block, tracked via inExpiryBlock/inPublishBlock below.
func removeBlocks(scanner *bufio.Scanner, targetVersion, ext string) string {
	var sb strings.Builder
	inExpiryBlock := false
	inPublishBlock := false
	modified := false

	for scanner.Scan() {
		line := scanner.Text()

		if inExpiryBlock {
			// Drop every line inside a resolved expiryVersion block, including its own closing tag.
			if matchFeatureClose(line) {
				inExpiryBlock = false
			}
			modified = true
			continue
		}

		if inPublishBlock {
			// Only reached for .md files (see below) - drop just the closing tag, content is kept.
			if matchFeatureClose(line) {
				inPublishBlock = false
				modified = true
				continue
			}
			sb.WriteString(line)
			sb.WriteString("\n")
			continue
		}

		if v, ok := matchOpen(line, expiryOpenPercentRe, expiryOpenAngleRe); ok && versionLTE(v, targetVersion) {
			inExpiryBlock = true
			modified = true
			continue
		}
		if ext == mdExt {
			if v, ok := matchOpen(line, publishOpenPercentRe, publishOpenAngleRe); ok && versionLTE(v, targetVersion) {
				inPublishBlock = true
				modified = true
				continue
			}
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	if !modified {
		return ""
	}

	return sb.String()
}

func matchOpen(line string, percentRe, angleRe *regexp.Regexp) (string, bool) {
	if m := percentRe.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	if m := angleRe.FindStringSubmatch(line); m != nil {
		return m[1], true
	}
	return "", false
}

func matchFeatureClose(line string) bool {
	return featureClosePercentRe.MatchString(line) || featureCloseAngleRe.MatchString(line)
}

// versionLTE reports whether v is the same as, or an earlier release than, target.
func versionLTE(v, target string) bool {
	vParts := strings.Split(v, ".")
	tParts := strings.Split(target, ".")

	toInt := func(s string) (int, error) {
		return strconv.Atoi(strings.TrimSpace(s))
	}

	for i := 0; i < len(vParts) || i < len(tParts); i++ {
		var vn, tn int
		var err error
		if i < len(vParts) {
			if vn, err = toInt(vParts[i]); err != nil {
				log.Printf("warning: could not parse version segment %q in %q, treating as 0", vParts[i], v)
			}
		}
		if i < len(tParts) {
			if tn, err = toInt(tParts[i]); err != nil {
				log.Printf("warning: could not parse version segment %q in %q, treating as 0", tParts[i], target)
			}
		}
		if vn != tn {
			return vn < tn
		}
	}
	return true // equal
}
