// Package assignments implements helper functions for partition assignment operations.
package assignments

import (
	"sort"

	"github.com/peter-evans/kdef/core/util/i32"
)

// AlterReplicationFactor alters the replication factor and returns the new assignments.
func AlterReplicationFactor(
	assignments [][]int32,
	targetRepFactor int,
	clusterReplicaCounts map[int32]int,
	brokers []int32,
) [][]int32 {
	currentRepFactor := len(assignments[0])
	leaderCounts := leaderCounts(assignments)
	replicaCounts := replicaCounts(assignments)

	switch {
	case targetRepFactor < currentRepFactor:
		return decreaseReplicationFactor(
			assignments,
			targetRepFactor,
			replicaCounts,
			clusterReplicaCounts,
		)
	case targetRepFactor > currentRepFactor:
		return increaseReplicationFactor(
			assignments,
			targetRepFactor,
			leaderCounts,
			replicaCounts,
			clusterReplicaCounts,
			brokers,
		)
	default:
		return assignments
	}
}

func decreaseReplicationFactor(
	assignments [][]int32,
	targetRepFactor int,
	replicaCounts map[int32]int,
	clusterReplicaCounts map[int32]int,
) [][]int32 {
	// Find the broker with the most replicas of any partition.
	selectBrokerToRemove := func(
		replicas []int32,
		brokerCounts map[int32]int,
		clusterReplicaCounts map[int32]int,
	) int32 {
		// Create a copy to prevent the original slice being sorted.
		sortedBrokers := append([]int32{}, replicas...)
		if clusterReplicaCounts != nil {
			// Sort based on broker frequency in the topic, break ties with broker frequency
			// in the cluster, and finally break ties with index.
			sort.Slice(sortedBrokers, func(i, j int) bool {
				return brokerCounts[sortedBrokers[i]] < brokerCounts[sortedBrokers[j]] ||
					(brokerCounts[sortedBrokers[i]] == brokerCounts[sortedBrokers[j]] &&
						clusterReplicaCounts[sortedBrokers[i]] < clusterReplicaCounts[sortedBrokers[j]]) ||
					(brokerCounts[sortedBrokers[i]] == brokerCounts[sortedBrokers[j]] &&
						clusterReplicaCounts[sortedBrokers[i]] == clusterReplicaCounts[sortedBrokers[j]] && i < j)
			})
		} else {
			// Sort based on broker frequency in the topic, breaking ties with index.
			sort.Slice(sortedBrokers, func(i, j int) bool {
				// Sort by increasing broker count, and then by index.
				return brokerCounts[sortedBrokers[i]] < brokerCounts[sortedBrokers[j]] ||
					(brokerCounts[sortedBrokers[i]] == brokerCounts[sortedBrokers[j]] && i < j)
			})
		}
		return sortedBrokers[len(sortedBrokers)-1]
	}

	newAssignments := Copy(assignments)
	for len(newAssignments[0]) > targetRepFactor {
		for partition, replicas := range newAssignments {
			// Select the broker ID to remove.
			selectedBrokerID := selectBrokerToRemove(replicas, replicaCounts, clusterReplicaCounts)

			// Create the modified replica set of broker IDs.
			modifiedReplicas := make([]int32, len(replicas)-1)
			j := 0
			for _, brokerID := range replicas {
				if brokerID == selectedBrokerID {
					continue
				}
				modifiedReplicas[j] = brokerID
				j++
			}

			newAssignments[partition] = modifiedReplicas
			replicaCounts[selectedBrokerID]--
			if clusterReplicaCounts != nil {
				clusterReplicaCounts[selectedBrokerID]--
			}
		}
	}

	return newAssignments
}

func increaseReplicationFactor(
	assignments [][]int32,
	targetRepFactor int,
	leaderCounts map[int32]int,
	replicaCounts map[int32]int,
	clusterReplicaCounts map[int32]int,
	brokers []int32,
) [][]int32 {
	newAssignments := Copy(assignments)
	for len(newAssignments[0]) < targetRepFactor {
		for partition, replicas := range newAssignments {
			// Find unused broker IDs for this partition.
			unusedBrokers := i32.Diff(brokers, replicas)

			// If the replicas are empty then a new partition is being populated.
			// In that case the next broker we add will be the preferred leader.
			isLeader := len(replicas) == 0

			// Determine the last used broker ID for this partition.
			var lastUsedBroker int32
			if !isLeader {
				lastUsedBroker = replicas[len(replicas)-1]
			}
			// If there are no unused brokers with an ID greater than the last used then reset to zero.
			// This will cause round-robin placement to begin a new cycle.
			if i32.Max(unusedBrokers) <= lastUsedBroker {
				lastUsedBroker = 0
			}

			// If the chosen broker will be the preferred leader we use leader counts to make sure
			// partition leaders are balanced across brokers.
			brokerCounts := replicaCounts
			if isLeader {
				brokerCounts = leaderCounts
			}

			// Select the broker ID to add.
			selectedBrokerID := selectBroker(unusedBrokers, brokerCounts, clusterReplicaCounts, lastUsedBroker)

			// Create the modified replica set of broker IDs.
			modifiedReplicas := make([]int32, len(replicas)+1)
			copy(modifiedReplicas, replicas)
			modifiedReplicas[len(modifiedReplicas)-1] = selectedBrokerID

			newAssignments[partition] = modifiedReplicas
			replicaCounts[selectedBrokerID]++
			if isLeader {
				leaderCounts[selectedBrokerID]++
			}
			if clusterReplicaCounts != nil {
				clusterReplicaCounts[selectedBrokerID]++
			}
		}
	}

	return newAssignments
}

// AddPartitions adds partitions and returns assignments for the new partitions.
func AddPartitions(
	assignments [][]int32,
	targetPartitions int,
	targetRepFactor int,
	clusterReplicaCounts map[int32]int,
	brokers []int32,
) [][]int32 {
	partitionsToAdd := targetPartitions - len(assignments)
	leaderCounts := leaderCounts(assignments)
	replicaCounts := replicaCounts(assignments)

	// Populate assignments for the new partitions.
	newPartitionAssignments := increaseReplicationFactor(
		make([][]int32, partitionsToAdd),
		targetRepFactor,
		leaderCounts,
		replicaCounts,
		clusterReplicaCounts,
		brokers,
	)

	return newPartitionAssignments
}

// SyncRackConstraints checks partition assignments are in sync with rack constraints, updating if necessary.
func SyncRackConstraints(
	assignments [][]int32,
	rackConstraints [][]string,
	brokersByRack map[string][]int32,
	clusterReplicaCounts map[int32]int,
) [][]int32 {
	// Modify assignments by the target replication factor.
	targetRepFactor := len(rackConstraints[0])
	newAssignments := make([][]int32, len(assignments))
	for partition, replicas := range assignments {
		newReplicas := make([]int32, targetRepFactor)
		copy(newReplicas, replicas)
		newAssignments[partition] = newReplicas
	}

	// The counts start from zero to make the assignments more predictable.
	// If there are a lot of out-of-sync replicas then the actual counts are meaningless.
	leaderCounts := make(map[int32]int)
	replicaCounts := make(map[int32]int)

	// Loop through the assignments and update replicas that are out of sync with the assigned rack.
	for replica := 0; replica < len(newAssignments[0]); replica++ {
		for partition, replicas := range newAssignments {
			rack := rackConstraints[partition][replica]

			// Build a list of used brokers for this rack on this partition.
			var usedRackBrokers []int32
			for r, b := range replicas {
				if rackConstraints[partition][r] == rack && i32.Contains(b, brokersByRack[rack]) {
					usedRackBrokers = append(usedRackBrokers, b)
				}
			}

			// Check if the assigned broker is out of sync with the defined rack.
			isLeader := replica == 0
			currentBrokerID := replicas[replica]
			if !i32.Contains(currentBrokerID, brokersByRack[rack]) {
				// Find unused broker IDs for this rack.
				unusedRackBrokers := i32.Diff(brokersByRack[rack], usedRackBrokers)

				// Last used broker is not used with rack constraints.
				var lastUsedBroker int32

				// If the chosen broker will be the preferred leader we use leader counts to make sure
				// partition leaders are balanced across brokers.
				brokerCounts := replicaCounts
				if isLeader {
					brokerCounts = leaderCounts
				}

				// Select the broker ID to add.
				selectedBrokerID := selectBroker(unusedRackBrokers, brokerCounts, clusterReplicaCounts, lastUsedBroker)

				// Replace the broker.
				newAssignments[partition][replica] = selectedBrokerID
			}
			// Update counts.
			replicaCounts[newAssignments[partition][replica]]++
			if isLeader {
				leaderCounts[newAssignments[partition][replica]]++
			}
			if clusterReplicaCounts != nil {
				clusterReplicaCounts[currentBrokerID]--
				clusterReplicaCounts[newAssignments[partition][replica]]++
			}
		}
	}

	return newAssignments
}

func Rebalance(
	assignments [][]int32,
	clusterReplicaCounts map[int32]int,
	brokers []int32,
) [][]int32 {
	leaderCounts := make(map[int32]int)
	replicaCounts := make(map[int32]int)

	newAssignments := Copy(assignments)
	for partition, replicas := range newAssignments {
		for replica, currentBrokerID := range replicas {
			isLeader := replica == 0

			// Find unused broker IDs for this partition.
			unusedBrokers := i32.Diff(brokers, newAssignments[partition])

			// Skip if no replacements are possible.
			if !isLeader && len(unusedBrokers) == 0 {
				continue
			}

			// Determine the last used broker ID for this partition.
			var lastUsedBroker int32
			if !isLeader {
				lastUsedBroker = replicas[replica-1]
			}
			// If there are no unused brokers with an ID greater than the last used then reset to zero.
			// This will cause round-robin placement to begin a new cycle.
			if len(unusedBrokers) > 0 && i32.Max(unusedBrokers) <= lastUsedBroker {
				lastUsedBroker = 0
			}

			// If the chosen broker will be the preferred leader we use leader counts to make sure
			// partition leaders are balanced across brokers.
			brokerCounts := replicaCounts
			brokerPool := unusedBrokers
			if isLeader {
				brokerCounts = leaderCounts
				brokerPool = brokers
			}

			// Select the broker ID to add.
			selectedBrokerID := selectBroker(brokerPool, brokerCounts, clusterReplicaCounts, lastUsedBroker)

			// Replace if the selected broker's count is at least 1 less than the current broker's count.
			// OR if the current follower replica is now the same as the leader of the partition.
			// This second case could occur if the leader is replaced by a broker already in use.
			if brokerCounts[currentBrokerID]-brokerCounts[selectedBrokerID] >= 1 ||
				(!isLeader && currentBrokerID == newAssignments[partition][0]) {
				newAssignments[partition][replica] = selectedBrokerID
			}

			// Update counts.
			replicaCounts[newAssignments[partition][replica]]++
			if isLeader {
				leaderCounts[newAssignments[partition][replica]]++
			}
			if clusterReplicaCounts != nil {
				clusterReplicaCounts[currentBrokerID]--
				clusterReplicaCounts[newAssignments[partition][replica]]++
			}
		}
	}

	return newAssignments
}

func RebalanceWithRackConstraints(
	assignments [][]int32,
	rackConstraints [][]string,
	clusterReplicaCounts map[int32]int,
	brokersByRack map[string][]int32,
) [][]int32 {
	leaderCounts := make(map[int32]int)
	replicaCounts := make(map[int32]int)

	newAssignments := Copy(assignments)
	for partition, replicas := range newAssignments {
		for replica, currentBrokerID := range replicas {
			isLeader := replica == 0
			rack := rackConstraints[partition][replica]

			// Build a list of used brokers for this rack on this partition.
			var usedRackBrokers []int32
			for r, b := range newAssignments[partition] {
				if rackConstraints[partition][r] == rack && i32.Contains(b, brokersByRack[rack]) {
					usedRackBrokers = append(usedRackBrokers, b)
				}
			}

			// Find unused broker IDs for this rack.
			unusedBrokers := i32.Diff(brokersByRack[rack], usedRackBrokers)

			// Skip if no replacements are possible.
			if !isLeader && len(unusedBrokers) == 0 {
				continue
			}

			// Last used broker is not used with rack constraints.
			var lastUsedBroker int32

			// If the chosen broker will be the preferred leader we use leader counts to make sure
			// partition leaders are balanced across brokers.
			brokerCounts := replicaCounts
			brokerPool := unusedBrokers
			if isLeader {
				brokerCounts = leaderCounts
				brokerPool = brokersByRack[rack]
			}

			// Select the broker ID to add.
			selectedBrokerID := selectBroker(brokerPool, brokerCounts, clusterReplicaCounts, lastUsedBroker)

			// Replace if the selected broker's count is at least 1 less than the current broker's count.
			// OR if the current follower replica is now the same as the leader of the partition.
			// This second case could occur if the leader is replaced by a broker already in use.
			if brokerCounts[currentBrokerID]-brokerCounts[selectedBrokerID] >= 1 ||
				(!isLeader && currentBrokerID == newAssignments[partition][0]) {
				newAssignments[partition][replica] = selectedBrokerID
			}

			// Update counts.
			replicaCounts[newAssignments[partition][replica]]++
			if isLeader {
				leaderCounts[newAssignments[partition][replica]]++
			}
			if clusterReplicaCounts != nil {
				clusterReplicaCounts[currentBrokerID]--
				clusterReplicaCounts[newAssignments[partition][replica]]++
			}
		}
	}

	return newAssignments
}

// Copy makes a copy of partition assignments.
func Copy(assignments [][]int32) [][]int32 {
	c := make([][]int32, len(assignments))
	for partition, replicas := range assignments {
		c[partition] = append([]int32{}, replicas...)
	}
	return c
}

func replicaCounts(assignments [][]int32) map[int32]int {
	replicaCounts := make(map[int32]int)
	for _, replicas := range assignments {
		for _, brokerID := range replicas {
			replicaCounts[brokerID]++
		}
	}
	return replicaCounts
}

func leaderCounts(assignments [][]int32) map[int32]int {
	leaderCounts := make(map[int32]int)
	for _, replicas := range assignments {
		leaderCounts[replicas[0]]++
	}
	return leaderCounts
}

func selectBroker(
	unusedBrokers []int32,
	brokerCounts map[int32]int,
	clusterReplicaCounts map[int32]int,
	lastUsedBroker int32,
) int32 {
	if clusterReplicaCounts != nil {
		return selectByTopicClusterUse(unusedBrokers, brokerCounts, clusterReplicaCounts, lastUsedBroker)
	}
	return selectByTopicUse(unusedBrokers, brokerCounts, lastUsedBroker)
}

func selectByTopicUse(unusedBrokers []int32, brokerCounts map[int32]int, lastUsedBroker int32) int32 {
	sort.Slice(unusedBrokers, func(i, j int) bool {
		/*
			Sort based on broker frequency in the topic, breaking ties with round-robin broker ID.

			Broker i has less replicas than j
			OR Broker i has the same number of replicas as j
			  AND broker ID i and j are both greater than the last used broker for this partition
			  AND broker ID i is less than j
			OR Broker i has the same number of replicas as j
			  AND either broker ID i or j are less than the last used broker for this partition
			  AND the difference between broker ID i and the last used broker is greater than that of j
		*/
		return brokerCounts[unusedBrokers[i]] < brokerCounts[unusedBrokers[j]] ||
			(brokerCounts[unusedBrokers[i]] == brokerCounts[unusedBrokers[j]] &&
				unusedBrokers[i] > lastUsedBroker && unusedBrokers[j] > lastUsedBroker &&
				unusedBrokers[i] < unusedBrokers[j]) ||
			(brokerCounts[unusedBrokers[i]] == brokerCounts[unusedBrokers[j]] &&
				(unusedBrokers[i] < lastUsedBroker || unusedBrokers[j] < lastUsedBroker) &&
				unusedBrokers[i]-lastUsedBroker > unusedBrokers[j]-lastUsedBroker)
	})
	return unusedBrokers[0]
}

func selectByTopicClusterUse(
	unusedBrokers []int32,
	brokerCounts map[int32]int,
	clusterReplicaCounts map[int32]int,
	lastUsedBroker int32,
) int32 {
	sort.Slice(unusedBrokers, func(i, j int) bool {
		// Sort based on broker frequency in the topic, break ties with broker frequency
		// in the cluster, and finally break ties with round-robin broker ID.
		return brokerCounts[unusedBrokers[i]] < brokerCounts[unusedBrokers[j]] ||
			(brokerCounts[unusedBrokers[i]] == brokerCounts[unusedBrokers[j]] &&
				clusterReplicaCounts[unusedBrokers[i]] < clusterReplicaCounts[unusedBrokers[j]]) ||
			(brokerCounts[unusedBrokers[i]] == brokerCounts[unusedBrokers[j]] &&
				clusterReplicaCounts[unusedBrokers[i]] == clusterReplicaCounts[unusedBrokers[j]] &&
				unusedBrokers[i] > lastUsedBroker && unusedBrokers[j] > lastUsedBroker &&
				unusedBrokers[i] < unusedBrokers[j]) ||
			(brokerCounts[unusedBrokers[i]] == brokerCounts[unusedBrokers[j]] &&
				clusterReplicaCounts[unusedBrokers[i]] == clusterReplicaCounts[unusedBrokers[j]] &&
				(unusedBrokers[i] < lastUsedBroker || unusedBrokers[j] < lastUsedBroker) &&
				unusedBrokers[i]-lastUsedBroker > unusedBrokers[j]-lastUsedBroker)
	})
	return unusedBrokers[0]
}
