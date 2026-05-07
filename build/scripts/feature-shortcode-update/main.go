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
	"strings"
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

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
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
		modifiedContent := removeBlocks(scanner, *version)

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

func removeBlocks(scanner *bufio.Scanner, version string) string {
	var sb strings.Builder
	expiryBlock := false
	publishBlock := false
	preserveLines := true
	modified := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if the line contains the starting of the expiryVersion shortcode with the specified version
		if strings.Contains(line, "{{% feature expiryVersion=\""+version+"\" %}}") ||
			strings.Contains(line, "{{< feature expiryVersion=\""+version+"\" >}}") {
			expiryBlock = true
			preserveLines = false
			modified = true
			continue
		}

		// Check if the line contains the ending of the expiryVersion shortcode
		if (strings.Contains(line, "{{% /feature %}}") && expiryBlock) ||
			(strings.Contains(line, "{{< /feature >}}") && expiryBlock) {
			expiryBlock = false
			preserveLines = true
			modified = true
			continue
		}

		// Check if the line contains the starting of the publishVersion shortcode with the specified version
		if strings.Contains(line, "{{% feature publishVersion=\""+version+"\" %}}") ||
			strings.Contains(line, "{{< feature publishVersion=\""+version+"\" >}}") {
			publishBlock = true
			modified = true
			continue
		}

		// Check if the line contains the ending of the publishVersion shortcode
		if (strings.Contains(line, "{{% /feature %}}") && publishBlock) ||
			(strings.Contains(line, "{{< /feature >}}") && publishBlock) {
			publishBlock = false
			modified = true
			continue
		}

		// Preserve the line if it is not within an expiryVersion or publishVersion block
		if preserveLines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	// If no modifications were made, return an empty string
	if !modified {
		return ""
	}

	return sb.String()
}
