package main

import (
	"testing"
)

func TestFilenameIsIrrelevant(t *testing.T) {
	exampleNames := []string{"inner", "first"}

	irrelevantMap := map[string]bool{
		// in deny-list
		"README.md":       true,
		"cloudbuild.yaml": true,
		// in deny-list and inside an example
		"examples/inner/outer/EXAMPLE.md":      true,
		"examples/first/second/third/test.yml": true,
		// not in deny list, but outside examples
		"outside/test.txt": true,
		"1/2/3/4/5.go":     true,
		// in examples and relevant
		"examples/inner/main.go": false,
	}

	for filename, expected := range irrelevantMap {
		observed := filenameIsIrrelevant(filename, exampleNames)
		if observed != expected {
			t.Errorf("%s was expected to be: %t", filename, expected)
		}
	}
}
