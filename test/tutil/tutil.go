package tutil

import (
	"io/ioutil"
	"strings"
	"testing"
)

// Check if an error message contains a string
func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

// Return the byte slice of a test fixture
func Fixture(t *testing.T, path string) []byte {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Errorf("failed to load test fixture %q: %v", path, err)
		t.FailNow()
	}
	return fileBytes
}
