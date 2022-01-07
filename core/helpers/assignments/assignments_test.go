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
		clusterReplicaCounts    map[int32]int
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
			name: "Tests decreasing the replication factor when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				targetReplicationFactor: 2,
				clusterReplicaCounts: map[int32]int{
					1: 0,
					2: 1,
					3: 0,
				},
				brokers: []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 3},
				{2, 3},
				{1, 2},
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
		{
			name: "Tests increasing the replication factor when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1},
					{2},
					{3},
				},
				targetReplicationFactor: 2,
				clusterReplicaCounts: map[int32]int{
					1: 0,
					2: 1,
					3: 0,
				},
				brokers: []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 3},
				{2, 1},
				{3, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AlterReplicationFactor(
				tt.args.assignments,
				tt.args.targetReplicationFactor,
				tt.args.clusterReplicaCounts,
				tt.args.brokers,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlterReplicationFactor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddPartitions(t *testing.T) {
	type args struct {
		assignments          [][]int32
		targetPartitions     int
		targetRepFactor      int
		clusterReplicaCounts map[int32]int
		brokers              []int32
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
				targetRepFactor:  3,
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
				targetRepFactor:  3,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests adding partitions with no existing assignments",
			args: args{
				assignments:      [][]int32{},
				targetPartitions: 3,
				targetRepFactor:  3,
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
				targetRepFactor:  3,
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
				targetRepFactor:  3,
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
				targetRepFactor:  2,
				brokers:          []int32{1, 2, 3},
			},
			want: [][]int32{
				{3, 2},
				{3, 1},
			},
		},
		{
			name: "Tests adding a partition when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
				},
				targetPartitions: 2,
				targetRepFactor:  3,
				clusterReplicaCounts: map[int32]int{
					1: 0,
					2: 1,
					3: 0,
				},
				brokers: []int32{1, 2, 3},
			},
			want: [][]int32{
				{3, 1, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddPartitions(
				tt.args.assignments,
				tt.args.targetPartitions,
				tt.args.targetRepFactor,
				tt.args.clusterReplicaCounts,
				tt.args.brokers,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddPartitions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyncRackAssignments(t *testing.T) {
	type args struct {
		assignments          [][]int32
		rackConstraints      [][]string
		brokersByRack        map[string][]int32
		clusterReplicaCounts map[int32]int
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests in-sync rack constraints",
			args: args{
				assignments: [][]int32{
					{1, 2, 4},
					{2, 3, 5},
					{3, 1, 6},
				},
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
				rackConstraints: [][]string{
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
			name: "Tests increasing the replication factor when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1, 4},
					{2, 5},
					{3, 6},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-a"},
					{"zone-a", "zone-b", "zone-a"},
					{"zone-a", "zone-b", "zone-a"},
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 2, 3},
					"zone-b": {4, 5, 6},
				},
				clusterReplicaCounts: map[int32]int{
					1: 1,
					2: 0,
					3: 0,
					4: 0,
					5: 0,
					6: 0,
				},
			},
			want: [][]int32{
				{1, 4, 2},
				{2, 5, 3},
				{3, 6, 1},
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
				rackConstraints: [][]string{
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
			if got := SyncRackConstraints(
				tt.args.assignments,
				tt.args.rackConstraints,
				tt.args.brokersByRack,
				tt.args.clusterReplicaCounts,
			); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SyncRackAssignments() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRebalance(t *testing.T) {
	type args struct {
		assignments          [][]int32
		clusterReplicaCounts map[int32]int
		brokers              []int32
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests no unused brokers",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				clusterReplicaCounts: nil,
				brokers:              []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests no unused brokers with leader rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{2, 1, 3},
				},
				clusterReplicaCounts: nil,
				brokers:              []int32{1, 2, 3},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests follower rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				clusterReplicaCounts: nil,
				brokers:              []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 4, 1},
				{3, 4, 2},
			},
		},
		{
			name: "Tests leader and follower rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{1, 3, 2},
					{3, 1, 2},
				},
				clusterReplicaCounts: nil,
				brokers:              []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 4, 1},
				{3, 4, 2},
			},
		},
		{
			name: "Tests leader and follower rebalance when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{1, 3, 2},
					{3, 1, 2},
				},
				clusterReplicaCounts: map[int32]int{
					1: 0,
					2: 1,
					3: 0,
					4: 0,
				},
				brokers: []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2, 3},
				{3, 4, 2},
				{2, 1, 4},
			},
		},
		{
			name: "Tests no changes",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 4, 1},
					{3, 4, 2},
				},
				clusterReplicaCounts: nil,
				brokers:              []int32{1, 2, 3, 4},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 4, 1},
				{3, 4, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Rebalance(tt.args.assignments, tt.args.clusterReplicaCounts, tt.args.brokers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Rebalance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRebalanceWithRackConstraints(t *testing.T) {
	type args struct {
		assignments          [][]int32
		rackConstraints      [][]string
		clusterReplicaCounts map[int32]int
		brokersByRack        map[string][]int32
	}
	tests := []struct {
		name string
		args args
		want [][]int32
	}{
		{
			name: "Tests no unused brokers",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-b"},
					{"zone-b", "zone-b", "zone-a"},
					{"zone-b", "zone-a", "zone-b"},
				},
				clusterReplicaCounts: nil,
				brokersByRack: map[string][]int32{
					"zone-a": {1},
					"zone-b": {2, 3},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests no unused brokers with leader rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{2, 1, 3},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-b"},
					{"zone-b", "zone-b", "zone-a"},
					{"zone-b", "zone-a", "zone-b"},
				},
				clusterReplicaCounts: nil,
				brokersByRack: map[string][]int32{
					"zone-a": {1},
					"zone-b": {2, 3},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 3, 1},
				{3, 1, 2},
			},
		},
		{
			name: "Tests follower rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
				},
				clusterReplicaCounts: nil,
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 6, 4},
				{3, 1, 5},
			},
		},
		{
			name: "Tests leader and follower rebalance",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
				},
				clusterReplicaCounts: nil,
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 6, 4},
				{3, 1, 5},
				{4, 5, 6},
				{5, 3, 1},
				{6, 4, 2},
			},
		},
		{
			name: "Tests leader and follower rebalance when there are cluster replica counts",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 3, 1},
					{3, 1, 2},
					{1, 2, 3},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
					{"zone-a", "zone-b", "zone-c"},
				},
				clusterReplicaCounts: map[int32]int{
					1: 0,
					2: 0,
					3: 0,
					4: 1,
					5: 0,
					6: 0,
					7: 0,
					8: 0,
					9: 0,
				},
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4, 7},
					"zone-b": {2, 5, 8},
					"zone-c": {3, 6, 9},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 6, 7},
				{3, 4, 5},
				{7, 8, 9},
			},
		},
		{
			name: "Tests no changes",
			args: args{
				assignments: [][]int32{
					{1, 2, 3},
					{2, 6, 4},
					{3, 1, 5},
					{4, 5, 6},
					{5, 3, 1},
					{6, 4, 2},
				},
				rackConstraints: [][]string{
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
					{"zone-a", "zone-b", "zone-c"},
					{"zone-b", "zone-c", "zone-a"},
					{"zone-c", "zone-a", "zone-b"},
				},
				clusterReplicaCounts: nil,
				brokersByRack: map[string][]int32{
					"zone-a": {1, 4},
					"zone-b": {2, 5},
					"zone-c": {3, 6},
				},
			},
			want: [][]int32{
				{1, 2, 3},
				{2, 6, 4},
				{3, 1, 5},
				{4, 5, 6},
				{5, 3, 1},
				{6, 4, 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RebalanceWithRackConstraints(tt.args.assignments, tt.args.rackConstraints, tt.args.clusterReplicaCounts, tt.args.brokersByRack); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RebalanceWithRackConstraints() = %v, want %v", got, tt.want)
			}
		})
	}
}
