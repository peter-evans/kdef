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

// Determine if two slices are equal regardless of element order
func UnorderedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, av := range a {
		m[av]++
	}
	for _, bv := range b {
		if _, ok := m[bv]; !ok {
			return false
		}
		m[bv] -= 1
		if m[bv] == 0 {
			delete(m, bv)
		}
	}
	return len(m) == 0
}

func Deduplicate(s []string) []string {
	k := make(map[string]bool)
	list := []string{}
	for _, ss := range s {
		if _, ok := k[ss]; !ok {
			k[ss] = true
			list = append(list, ss)
		}
	}
	return list
}
