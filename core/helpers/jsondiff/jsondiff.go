package jsondiff

import (
	"encoding/json"

	"github.com/peter-evans/kdef/util/diff"
)

// Compute the line-oriented diff between the JSON representation of two structs
func Diff(a interface{}, b interface{}) (string, error) {
	// Convert interface to JSON handling null pointers
	toJson := func(d interface{}) (string, error) {
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

	aJson, err := toJson(a)
	if err != nil {
		return "", err
	}

	bJson, err := toJson(b)
	if err != nil {
		return "", err
	}

	if aJson != bJson {
		return diff.LineOriented(aJson, bJson), nil
	} else {
		return "", nil
	}
}
