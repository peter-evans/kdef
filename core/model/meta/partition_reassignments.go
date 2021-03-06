// Package meta implements metadata structures and related operations.
package meta

import "github.com/bradfitz/slice" //nolint

// PartitionReassignment represents a partition reassignment.
type PartitionReassignment struct {
	Partition        int32   `json:"partition"`
	Replicas         []int32 `json:"replicas"`
	AddingReplicas   []int32 `json:"addingReplicas"`
	RemovingReplicas []int32 `json:"removingReplicas"`
}

// PartitionReassignments represents a slice of PartitionReassignment.
type PartitionReassignments []PartitionReassignment

// Sort sorts by partition ID.
func (p PartitionReassignments) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8.
	//nolint
	slice.Sort(p[:], func(i, j int) bool {
		return p[i].Partition < p[j].Partition
	})
}
