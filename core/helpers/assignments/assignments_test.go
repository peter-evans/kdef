// Package assignments implements helper functions for partition assignment operations.
package assignments

import (
	"reflect"
	"testing"
)

func TestAlterReplicationFactor(t *testing.T) {
	type args struct {
		assignments             [][]int32
		targetReplicationFactor int
		brokers                 []int32
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests decreasing the replication factor by 1",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetReplicationFactor: 2,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2},
				{2, 3},
				{3, 1},
			},
		},
		{
			name: "Tests decreasing the replication factor by 2",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetReplicationFactor: 1,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1},
				{2},
				{3},
			},
		},
		{
			name: "Tests decreasing the replication factor with unbalanced preferred leaders",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 1, 3},
					{2, 3, 1},
				},
				targetReplicationFactor: 2,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2},
				{2, 3},
				{3, 1},
			},
		},
		{
			name: "Tests decreasing the replication factor with unbalanced replicas",
			args: args{
				assignments: [][]int32{
					{1, 3},
					{1, 2},
					{2, 1},
					{1, 2},
					{2, 1},
				},
				targetReplicationFactor: 1,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{3},
				{1},
				{2},
				{1},
				{2},
			},
		},
		{
			name: "Tests decreasing the replication factor with unused brokers",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetReplicationFactor: 2,
				brokers:                 []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2},
				{2, 3},
				{3, 1},
			},
		},
		{
			name: "Tests increasing the replication factor by 1",
			args: args{
				assignments: [][]int32{
					{1, 2},
					{2, 3},
					{3, 1},
				},
				targetReplicationFactor: 3,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests increasing the replication factor by 2",
			args: args{
				assignments: [][]int32{
					{1},
					{2},
					{3},
				},
				targetReplicationFactor: 3,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests increasing the replication factor with unbalanced preferred leaders",
			args: args{
				assignments: [][]int32{
					{1},
					{2},
					{2},
				},
				targetReplicationFactor: 3,
				brokers:                 []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 3, 2},
				{2, 3, 1},
				{2, 1, 3},
			},
		},
		{
			name: "Tests increasing the replication factor with unused brokers",
			args: args{
				assignments: [][]int32{
					{1, 2},
					{2, 3},
					{3, 1},
				},
				targetReplicationFactor: 4,
				brokers:                 []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2, 4, 3},
				{2, 3, 4, 1},
				{3, 1, 2, 4},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AlterReplicationFactor(
				tt.args.assignments,
				tt.args.targetReplicationFactor,
				tt.args.brokers,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlterReplicationFactor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddPartitions(t *testing.T) {
	type args struct {
		assignments      [][]int32
		targetPartitions int
		brokers          []int32
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests adding a single partition",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
				},
				targetPartitions: 3,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{3, 1, 2},
			},
		},
		{
			name: "Tests adding multiple partitions",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetPartitions: 6,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests adding multiple partitions with unused brokers",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetPartitions: 6,
				brokers:          []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{4, 3, 1},
				{1, 4, 2},
				{2, 4, 3},
			},
		},
		{
			name: "Tests adding partitions with unbalanced preferred leaders",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 1, 3},
					{2, 3, 1},
				},
				targetPartitions: 5,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{3, 2, 1},
				{1, 2, 3},
			},
		},
		{
			name: "Tests adding partitions with unbalanced replicas",
			args: args{
				assignments: [][]int32{
					{1, 3},
					{1, 2},
					{2, 1},
					{1, 2},
					{2, 1},
				},
				targetPartitions: 7,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{3, 2},
				{3, 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddPartitions(tt.args.assignments, tt.args.targetPartitions, tt.args.brokers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddPartitions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncRackAssignments(t *testing.T) {
	type args struct {
		assignments     [][]int32
		rackAssignments [][]string
		brokersByRack   map[string][]int32
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests in-sync rack assignments",
			args: args{
				assignments: [][]int32{
					{1, 2, 4},
					{2, 3, 5},
					{3, 1, 6},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-b", "zone-c", "zone-b"},
					{"zone-c", "zone-a", "zone-c"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 4},
				{2, 3, 5},
				{3, 1, 6},
			},
		},
		{
			name: "Tests an out-of-sync replica",
			args: args{
				assignments: [][]int32{
					{1, 2, 4},
					{2, 4, 5},
					{3, 1, 6},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-b", "zone-c", "zone-b"},
					{"zone-c", "zone-a", "zone-c"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 4},
				{2, 6, 5},
				{3, 1, 6},
			},
		},
		{
			name: "Tests an out-of-sync preferred leader",
			args: args{
				assignments: [][]int32{
					{1, 2, 4},
					{2, 3, 5},
					{4, 1, 6},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-b", "zone-c", "zone-b"},
					{"zone-c", "zone-a", "zone-c"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 4},
				{2, 3, 5},
				{3, 1, 6},
			},
		},
		{
			name: "Tests many out-of-sync replicas",
			args: args{
				assignments: [][]int32{
					{3, 2, 4},
					{2, 4, 1},
					{3, 1, 5},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-b", "zone-c", "zone-b"},
					{"zone-c", "zone-a", "zone-c"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 4},
				{2, 6, 5},
				{3, 1, 6},
			},
		},
		{
			name: "Tests completely out-of-sync replicas",
			args: args{
				assignments: [][]int32{
					{2, 6},
					{3, 1},
					{4, 5},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b"},
					{"zone-b", "zone-c"},
					{"zone-c", "zone-a"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 5},
				{2, 6},
				{3, 4},
			},
		},
		{
			name: "Tests decreasing the replication factor",
			args: args{
				assignments: [][]int32{
					{1, 2, 4},
					{2, 3, 5},
					{3, 1, 6},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b"},
					{"zone-b", "zone-c"},
					{"zone-c", "zone-a"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2},
				{2, 3},
				{3, 1},
			},
		},
		{
			name: "Tests increasing the replication factor",
			args: args{
				assignments: [][]int32{
					{1, 2},
					{2, 3},
					{3, 1},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a", "zone-c"},
					{"zone-b", "zone-c", "zone-b", "zone-a"},
					{"zone-c", "zone-a", "zone-c", "zone-b"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 4, 6},
				{2, 3, 5, 4},
				{3, 1, 6, 5},
			},
		},
		{
			name: "Tests increasing the replication factor with out-of-sync replicas",
			args: args{
				assignments: [][]int32{
					{1, 3},
					{2, 3},
					{2, 1},
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a", "zone-c"},
					{"zone-b", "zone-c", "zone-b", "zone-a"},
					{"zone-c", "zone-a", "zone-c", "zone-b"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 5, 4, 6},
				{2, 3, 5, 4},
				{3, 1, 6, 2},
			},
		},
		{
			name: "Tests populating empty assignments",
			args: args{
				assignments: [][]int32{
					make([]int32, 3),
					make([]int32, 3),
					make([]int32, 3),
				},
				rackAssignments: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-b", "zone-c", "zone-b"},
					{"zone-c", "zone-a", "zone-c"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 5, 4},
				{2, 6, 5},
				{3, 4, 6},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SyncRackAssignments(tt.args.assignments, tt.args.rackAssignments, tt.args.brokersByRack); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SyncRackAssignments() = %v, want %v", got, tt.want)
			}
		})
	}
}
