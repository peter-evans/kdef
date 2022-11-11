// Package tutil implements testing utility functions.
package tutil

import (
	"os"
	"strings"
	"testing"
)

// ErrorContains checks if an error message contains a string.
func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

// Fixture returns the byte slice of a test fixture.
func Fixture(t *testing.T, path string) []byte {
	t.Helper()
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to load test fixture %q: %v", path, err)
		t.FailNow()
	}
	return fileBytes
}
