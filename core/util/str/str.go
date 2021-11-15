// Package str implements functions for various string operations.
package str

import (
	"strings"
)

// Deref dereferences a string pointer while handling nil.
func Deref(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Norm normalises a string.
func Norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// Contains determines if a value is contained in a slice.
func Contains(str string, s []string) bool {
	for _, item := range s {
		if item == str {
			return true
		}
	}
	return false
}

// UnorderedEqual determines if two slices are equal regardless of element order.
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
		m[bv]--
		if m[bv] == 0 {
			delete(m, bv)
		}
	}
	return len(m) == 0
}

// Deduplicate removes duplicate values from a slice.
func Deduplicate(s []string) []string {
	k := make(map[string]bool)
	l := []string{}
	for _, ss := range s {
		if _, ok := k[ss]; !ok {
			k[ss] = true
			l = append(l, ss)
		}
	}
	return l
}
