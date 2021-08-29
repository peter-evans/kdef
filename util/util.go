package util

import "strings"

// Dereference a string pointer handling nil
func DerefStr(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Normalise a string
func NormStr(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// Determine if a list contains a string
func StringInList(str string, list []string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

func DuplicateInSlice(s []int32) bool {
	k := make(map[int32]bool, len(s))
	for _, ss := range s {
		if k[ss] {
			return true
		} else {
			k[ss] = true
		}
	}
	return false
}
