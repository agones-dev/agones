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
	"fmt"
	"io/fs"
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
	mdExt            = ".md"
	htmlExt          = ".html"
	maxScanTokenSize = 10 * 1024 * 1024 // 10MB
)

func main() {
	version := flag.String("version", "", "the version being released, e.g. 1.61.0")
	flag.Parse()

	if *version == "" {
		log.Fatal("must specify -version")
	}

	dirPath := "site/content/en/docs"
	filesProcessed := 0

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ext := filepath.Ext(d.Name())
		if d.IsDir() || (ext != mdExt && ext != htmlExt) {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer func() {
			if cerr := file.Close(); cerr != nil {
				log.Printf("warning: closing %s: %v", path, cerr)
			}
		}()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), maxScanTokenSize)

		modifiedContent, changed, err := removeBlocks(scanner, *version, ext)
		if err != nil {
			// Fix for review comment #1: propagate the scan failure and
			// bail out *before* os.Create below, instead of silently
			// writing back a truncated result.
			return fmt.Errorf("scanning %s: %w", path, err)
		}

		if changed {
			if werr := os.WriteFile(path, []byte(modifiedContent), 0o644); werr != nil {
				return fmt.Errorf("writing %s: %w", path, werr)
			}
			log.Printf("Processed file: %s", path)
			filesProcessed++
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if filesProcessed == 0 {
		log.Println("There are no files with feature expiryVersion or publishVersion shortcodes")
	}
}

// removeBlocks assumes feature shortcodes never nest — a {{%|{{< /feature %}}|>}}
// always closes the single innermost open block, tracked via
// inExpiryBlock/inPublishBlock below.
//
// It returns the (possibly unchanged) file content, whether anything was
// actually changed, and any scanning error encountered.
func removeBlocks(scanner *bufio.Scanner, targetVersion, ext string) (string, bool, error) {
	var sb strings.Builder
	inExpiryBlock := false
	inPublishBlock := false
	modified := false

	for scanner.Scan() {
		line := scanner.Text()

		if inExpiryBlock {
			// Drop every line inside a resolved expiryVersion block,
			// including its own closing tag.
			if matchFeatureClose(line) {
				inExpiryBlock = false
			}
			modified = true
			continue
		}

		if inPublishBlock {
			// Only reached for .md files (see below) - drop just the
			// closing tag, content is kept.
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
			modified = true
			if matchFeatureClose(line) {
				// Whole block resolves on one line: drop it entirely,
				// stay out of any block state.
				continue
			}
			inExpiryBlock = true
			continue
		}

		if ext == mdExt {
			if v, ok := matchOpen(line, publishOpenPercentRe, publishOpenAngleRe); ok && versionLTE(v, targetVersion) {
				modified = true
				unwrapped, closedOnSameLine := stripPublishTagsOnLine(line)
				if closedOnSameLine {
					if strings.TrimSpace(unwrapped) != "" {
						sb.WriteString(unwrapped)
						sb.WriteString("\n")
					}
					continue
				}
				inPublishBlock = true
				if strings.TrimSpace(unwrapped) != "" {
					sb.WriteString(unwrapped)
					sb.WriteString("\n")
				}
				continue
			}
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", false, err
	}

	return sb.String(), modified, nil
}

// stripPublishTagsOnLine removes a publishVersion open tag (already
// confirmed matched by the caller) and, if present on the same line, its
// matching close tag, returning the remaining text and whether the close
// tag was found on this line.
func stripPublishTagsOnLine(line string) (string, bool) {
	out := publishOpenPercentRe.ReplaceAllString(line, "")
	out = publishOpenAngleRe.ReplaceAllString(out, "")

	closedOnSameLine := matchFeatureClose(out)
	if closedOnSameLine {
		out = featureClosePercentRe.ReplaceAllString(out, "")
		out = featureCloseAngleRe.ReplaceAllString(out, "")
	}
	return out, closedOnSameLine
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
