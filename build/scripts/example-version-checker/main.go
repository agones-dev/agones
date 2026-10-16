// Package main checks that every example modified relative to main has a version
// increase in its Makefile.
package main

import (
	"log"
)

func main() {
	log.Println("Checking that modified examples have new versions.")
	log.Println()

	failed := getFailedFiles()

	logReport(failed)
}
