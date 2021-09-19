package str

import (
	"strings"
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

// Determine if a value is contained in a list
func Contains(str string, list []string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}
