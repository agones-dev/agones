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

// The main package implements a program that removes the old version of the link in `navbarDropdownMenuLink` found in the `site/layouts/partials/navbar.html` file
package main

import (
	"flag"
	"log"
	"os"
	"regexp"
	"strings"
)

func removeLastDropdownItem(input string) string {
	re := regexp.MustCompile(`(?s)(<div class="dropdown-menu" aria-labelledby="navbarDropdownMenuLink">)(.*?)(</div>)`)
	matches := re.FindStringSubmatch(input)

	// Determine the indentation before the <div>
	divStartPos := strings.Index(input, matches[1])
	divIndentation := ""
	for i := divStartPos - 1; i >= 0 && (input[i] == ' ' || input[i] == '\t'); i-- {
		divIndentation = string(input[i]) + divIndentation
	}

	// Split links inside the dropdown
	links := strings.Split(matches[2], "</a>")

	updatedLinks := strings.Join(links[:len(links)-2], "</a>") + "</a>\n" + divIndentation

	return strings.Replace(input, matches[2], updatedLinks, 1)
}

func main() {
	filePath := flag.String("file", "", "HTML File Path")
	flag.Parse()

	if *filePath == "" {
		log.Println("Please provide the path to the HTML file using the FILENAME flag.")
		return
	}

	content, err := os.ReadFile(*filePath)
	if err != nil {
		log.Println("Error reading the file:", err)
		return
	}

	updatedContent := removeLastDropdownItem(string(content))

	err = os.WriteFile(*filePath, []byte(updatedContent), 0o644)
	if err != nil {
		log.Println("Error writing to the file:", err)
		return
	}

	log.Println("Successfully updated the file.")
}
