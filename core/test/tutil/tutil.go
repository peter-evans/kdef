package tutil

import (
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/peter-evans/kdef/config"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/ctl/apply/docparse"
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

// A wrapper around docparse.FromFile to simplify test usage
func FileToYamlDocs(t *testing.T, path string) []string {
	yamlDocs, err := docparse.FromFile(path, docparse.Format(opt.YamlFormat))
	if err != nil {
		t.Errorf("failed to load test fixture %q: %v", path, err)
		t.FailNow()
	}
	return yamlDocs
}

// Determine if two strings are equal JSON
func EqualJSON(t *testing.T, s1 string, s2 string) bool {
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

// Produces random bytes of length n
func RandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// A wrapper around NewClient to simplify test usage
func CreateClient(t *testing.T, configOpts []string) *client.Client {
	cl, err := config.NewClient(&config.ConfigOptions{
		ConfigPath: "does-not-exist",
		ConfigOpts: configOpts,
	})
	if err != nil {
		t.Errorf("failed to create client: %v", err)
		t.FailNow()
	}
	return cl
}
