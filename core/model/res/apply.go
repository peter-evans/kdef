package res

import (
	"encoding/json"
	"fmt"

	"github.com/peter-evans/kdef/core/model/meta"
)

// An apply result
type ApplyResult struct {
	LocalDef  interface{} `json:"local"`
	RemoteDef interface{} `json:"remote"`
	Data      interface{} `json:"data"`
	Diff      string      `json:"diff"`
	Err       string      `json:"error"`
	Applied   bool        `json:"applied"`
}

// Return the error of an apply
func (a ApplyResult) GetErr() error {
	if len(a.Err) > 0 {
		return fmt.Errorf(a.Err)
	} else {
		return nil
	}
}

// Determine if the apply has unapplied changes
func (a ApplyResult) HasUnappliedChanges() bool {
	return len(a.Diff) > 0 && !a.Applied
}

// A slice of ApplyResult pointers
type ApplyResults []*ApplyResult

// Determine if any apply result has an error
func (a ApplyResults) ContainsErr() bool {
	for _, res := range a {
		err := res.GetErr()
		if err != nil {
			return true
		}
	}
	return false
}

// Determine if any apply result has unapplied changes
func (a ApplyResults) ContainsUnappliedChanges() bool {
	for _, res := range a {
		if res.HasUnappliedChanges() {
			return true
		}
	}
	return false
}

// Convert apply results to JSON
func (a ApplyResults) JSON() (string, error) {
	j, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// *** Topic apply specific ***

// Misc data for a topic apply result
type TopicApplyResultData struct {
	PartitionReassignments []meta.PartitionReassignment `json:"partitionReassignments"`
}
