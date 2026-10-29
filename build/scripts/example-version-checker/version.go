package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const examplesDir = "examples"

var excludedPatterns = [...]string{"*.md", "*.yaml", "*.yml", "OWNERS", ".gitignore"}
var excludedExamples = []string{"autoscaler-wasm"}

func dirIsExample(dirName string) bool {
	makefileName := dirName + "/Makefile"
	_, err := os.Stat(makefileName)
	return err == nil
}

func getAllExampleNames() []string {
	dirNames := make([]string, 0)

	baseDir := fmt.Sprintf("%s%s", rootDir, examplesDir)

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		log.Fatalf("Could not open directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if slices.Contains(excludedExamples, name) {
			continue
		}
		path := fmt.Sprintf("%s/%s", baseDir, name)
		if dirIsExample(path) {
			dirNames = append(dirNames, name)
		}
	}
	return dirNames
}

func getVersionFromMakefile(contents string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		if lineContainsVersion(line) {
			return getVersionFromLine(line), nil
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Could not get version: %v", err)
	}

	return "", errors.New("no version could be found")
}

func lineContainsVersion(line string) bool {
	return strings.HasPrefix(line, "version :=")
}

func getVersionFromLine(line string) string {
	split := strings.SplitN(line, ":=", 2)
	if len(split) != 2 {
		log.Fatalf("Bad version line: %s", line)
	}
	trimmed := strings.TrimSpace(split[1])
	if trimmed == "" {
		log.Fatalf("Version can not be empty: %s", line)
	}
	return trimmed
}

func filenameIsIrrelevant(filename string, exampleNames []string) bool {
	if !filenameInExamples(filename, exampleNames) {
		return true
	}

	_, splitname := filepath.Split(filename)

	for _, excludedName := range excludedPatterns {
		matches, err := filepath.Match(excludedName, splitname)
		if err != nil {
			log.Fatalf("Unknown error: %s", err)
		}

		if matches {
			return true
		}
	}
	return false
}

func filenameInExamples(filename string, exampleNames []string) bool {
	for _, exampleName := range exampleNames {
		path := fmt.Sprintf("%s/%s", examplesDir, exampleName)
		if strings.HasPrefix(filename, path) {
			return true
		}
	}
	return false
}
