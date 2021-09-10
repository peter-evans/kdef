package str

import (
	"encoding/json"
	"strings"

	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

// Dereference a string pointer handling nil
func Deref(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Normalise a string
func Norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// Determine if a list contains a string
func Contains(str string, list []string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

// Return the diff of two JSON strings (as []byte)
func JsonDiff(a []byte, b []byte) (string, error) {
	differ := diff.New()
	diff, err := differ.Compare(a, b)
	if err != nil {
		return "", err
	}

	if !diff.Modified() {
		return "", nil
	}

	var aJson map[string]interface{}
	if err := json.Unmarshal(a, &aJson); err != nil {
		return "", err
	}

	formatter := formatter.NewAsciiFormatter(aJson, formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
		Coloring:       false,
	})
	diffStr, err := formatter.Format(diff)
	if err != nil {
		return "", err
	}

	return diffStr, nil
}
