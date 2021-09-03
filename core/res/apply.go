package res

import (
	"encoding/json"
	"fmt"
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

// A collection of apply results
type ApplyResults []*ApplyResult

// Convert apply results to JSON
func (a ApplyResults) JSON() (string, error) {
	j, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

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

// Topic apply specific

// A partition reassignment
type PartitionReassignment struct {
	Partition        int32   `json:"partition"`
	Replicas         []int32 `json:"replicas"`
	AddingReplicas   []int32 `json:"addingReplicas"`
	RemovingReplicas []int32 `json:"removingReplicas"`
}

// A collection of partition reassignments
type PartitionReassignments []PartitionReassignment

// Misc data for a topic apply result
type TopicApplyResultData struct {
	PartitionReassignments []PartitionReassignment `json:"partitionReassignments"`
}
