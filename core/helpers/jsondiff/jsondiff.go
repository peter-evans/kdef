package jsondiff

import (
	"encoding/json"

	"github.com/peter-evans/kdef/core/util/diff"
)

// Compute the line-oriented diff between the JSON representation of two structs
func Diff(a interface{}, b interface{}) (string, error) {
	// Convert interface to JSON handling null pointers
	toJSON := func(d interface{}) (string, error) {
		j := "null"
		if d != nil {
			jBytes, err := json.MarshalIndent(d, "", "  ")
			if err != nil {
				return "", err
			}
			j = string(jBytes)
		}
		return j, nil
	}

	aJSON, err := toJSON(a)
	if err != nil {
		return "", err
	}

	bJSON, err := toJSON(b)
	if err != nil {
		return "", err
	}

	if aJSON != bJSON {
		return diff.LineOriented(aJSON, bJSON), nil
	}

	return "", nil
}
