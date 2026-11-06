package main

import (
	"log"
	"strings"
)

func logReport(failedFileNames []string) {
	if len(failedFileNames) != 0 {
		grouped := groupByExampleName(failedFileNames)
		reportFailedFiles(grouped)
	}

	log.Println()
	log.Println("The check for version changes succeeded.")
}

func reportFailedFiles(groupedPaths map[string][]string) {
	log.Println()
	log.Println("The below examples were modified, without increasing the version in its Makefile.")
	log.Println()
	log.Println("examples/")

	for exampleName, filePaths := range groupedPaths {
		log.Printf("\t%s/\n", exampleName)
		for _, filePath := range filePaths {
			log.Printf("\t\t%s\n", filePath)
		}
	}

	log.Println()
	log.Fatal("The check for version changes failed. Please increase the above versions.")
}

func groupByExampleName(filePaths []string) map[string][]string {
	grouped := make(map[string][]string, 0)

	for _, filePath := range filePaths {
		grouped = appendToGroup(grouped, filePath)
	}

	return grouped
}

func splitPath(filePath string) (string, string) {
	trimmed, _ := strings.CutPrefix(filePath, examplesDir+"/")
	split := strings.SplitN(trimmed, "/", 2)

	exampleName := split[0]
	relative := split[1]

	return exampleName, relative
}

func appendToGroup(grouped map[string][]string, filePath string) map[string][]string {
	exampleName, relative := splitPath(filePath)

	val, ok := grouped[exampleName]
	if ok {
		val = append(val, relative)
	} else {
		val = []string{relative}
	}

	grouped[exampleName] = val

	return grouped
}
