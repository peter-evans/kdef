// Package tutil implements testing utility functions.
package tutil

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/apply/docparse"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/opt"
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

// FileToYAMLDocs wraps docparse.FromFile to simplify test usage.
func FileToYAMLDocs(t *testing.T, path string) []string {
	t.Helper()
	yamlDocs, err := docparse.FromFile(path, docparse.Format(opt.YAMLFormat))
	if err != nil {
		t.Errorf("failed to load test fixture %q: %v", path, err)
		t.FailNow()
	}
	return yamlDocs
}

// EqualJSON determines if two strings are equal JSON.
func EqualJSON(t *testing.T, s1 string, s2 string) bool {
	t.Helper()
	toInterface := func(s string) interface{} {
		var i interface{}
		if err := json.Unmarshal([]byte(s), &i); err != nil {
			t.Errorf("json unmarshal failed: %v", err)
			t.FailNow()
		}
		return i
	}
	return reflect.DeepEqual(
		toInterface(s1),
		toInterface(s2),
	)
}

// RandomBytes produces random bytes of length n.
func RandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// CreateClient wraps config.NewClient to simplify test usage.
func CreateClient(t *testing.T, configOpts []string) *client.Client {
	t.Helper()
	cl, err := config.NewClient(&config.Options{
		ConfigPath: "does-not-exist",
		ConfigOpts: configOpts,
	})
	if err != nil {
		t.Errorf("failed to create client: %v", err)
		t.FailNow()
	}
	return cl
}
