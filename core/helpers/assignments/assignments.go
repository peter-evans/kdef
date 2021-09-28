package assignments

import (
	"sort"

	"github.com/peter-evans/kdef/util/i32"
)

// Alter the replication factor and return the new assignments
func AlterReplicationFactor(
	assignments [][]int32,
	targetReplicationFactor int,
	brokers []int32,
) [][]int32 {
	currentReplicationFactor := len(assignments[0])
	leaderCounts := leaderCounts(assignments)
	replicaCounts := replicaCounts(assignments)

	if targetReplicationFactor < currentReplicationFactor {
		return decreaseReplicationFactor(assignments, targetReplicationFactor, replicaCounts)
	} else if targetReplicationFactor > currentReplicationFactor {
		return increaseReplicationFactor(assignments, targetReplicationFactor, leaderCounts, replicaCounts, brokers)
	} else {
		return assignments
	}
}

// Decrease the replication factor and return the new assignments
func decreaseReplicationFactor(
	assignments [][]int32,
	targetReplicationFactor int,
	replicaCounts map[int32]int,
) [][]int32 {
	// Find the broker with the most replicas of any partition
	mostPopulousBroker := func(replicas []int32, brokerCounts map[int32]int) int32 {
		// Create a copy to prevent the original slice being sorted
		sortedBrokers := append([]int32{}, replicas...)
		sort.Slice(sortedBrokers, func(i, j int) bool {
			// Order by increasing broker count, and then by index
			return brokerCounts[sortedBrokers[i]] < brokerCounts[sortedBrokers[j]] ||
				(brokerCounts[sortedBrokers[i]] == brokerCounts[sortedBrokers[j]] && i < j)
		})
		return sortedBrokers[len(sortedBrokers)-1]
	}

	newAssignments := Copy(assignments)
	for len(newAssignments[0]) > int(targetReplicationFactor) {
		for i, replicas := range newAssignments {
			// Find the broker ID to remove
			brokerIdToRemove := mostPopulousBroker(replicas, replicaCounts)

			// Create the modified replica set of broker IDs
			modifiedReplicas := make([]int32, len(replicas)-1)
			j := 0
			for _, brokerId := range replicas {
				if brokerId == brokerIdToRemove {
					continue
				}
				modifiedReplicas[j] = brokerId
				j++
			}

			newAssignments[i] = modifiedReplicas
			replicaCounts[brokerIdToRemove]--
		}
	}

	return newAssignments
}

// Increase the replication factor and return the new assignments
func increaseReplicationFactor(
	assignments [][]int32,
	targetReplicationFactor int,
	leaderCounts map[int32]int,
	replicaCounts map[int32]int,
	brokers []int32,
) [][]int32 {
	newAssignments := Copy(assignments)
	for len(newAssignments[0]) < int(targetReplicationFactor) {
		for i, replicas := range newAssignments {
			// Find unused broker IDs for this partition
			unusedBrokers := i32.Diff(brokers, replicas)

			// If the replicas are empty then a new partition is being populated
			// In that case the next broker we add will be the preferred leader
			isLeader := len(replicas) == 0

			// Determine the last used broker ID for this partition
			var lastUsedBroker int32 = 0
			if !isLeader {
				lastUsedBroker = replicas[len(replicas)-1]
			}
			// If there are no usused brokers with an ID greater than the last used then reset to zero
			// This will cause round-robin placement to begin a new cycle
			if i32.Max(unusedBrokers) <= lastUsedBroker {
				lastUsedBroker = 0
			}

			// If the chosen broker will be the preferred leader we use leader counts to make sure
			// leaders are balanced across partitions in the assignments
			brokerCounts := replicaCounts
			if isLeader {
				brokerCounts = leaderCounts
			}

			// Find the broker ID to add
			brokerIdToAdd := leastPopulousBroker(unusedBrokers, brokerCounts, lastUsedBroker)

			// Create the modified replica set of broker IDs
			modifiedReplicas := make([]int32, len(replicas)+1)
			copy(modifiedReplicas, replicas)
			modifiedReplicas[len(modifiedReplicas)-1] = brokerIdToAdd

			newAssignments[i] = modifiedReplicas
			replicaCounts[brokerIdToAdd]++
			if isLeader {
				leaderCounts[brokerIdToAdd]++
			}
		}
	}

	return newAssignments
}

// Add partitions and return assignments for the new partitions
func AddPartitions(
	assignments [][]int32,
	targetPartitions int,
	brokers []int32,
) [][]int32 {
	partitionsToAdd := targetPartitions - len(assignments)
	targetReplicationFactor := len(assignments[0])
	leaderCounts := leaderCounts(assignments)
	replicaCounts := replicaCounts(assignments)

	// Populate assignments for the new partitions
	newPartitionAssignments := increaseReplicationFactor(
		make([][]int32, partitionsToAdd),
		targetReplicationFactor,
		leaderCounts,
		replicaCounts,
		brokers,
	)

	return newPartitionAssignments
}

// Check partition assignments are in sync with rack assignments, updating if necessary
func SyncRackAssignments(
	assignments [][]int32,
	rackAssignments [][]string,
	brokersByRack map[string][]int32,
) [][]int32 {
	// Modify assignments by the target replication factor
	targetReplicationFactor := len(rackAssignments[0])
	newAssignments := make([][]int32, len(assignments))
	for i, replicas := range assignments {
		newReplicas := make([]int32, targetReplicationFactor)
		copy(newReplicas, replicas)
		newAssignments[i] = newReplicas
	}

	leaderCounts := make(map[int32]int)
	replicaCounts := make(map[int32]int)

	// Loop through the assignments and update replicas that are out of sync with the assigned rack
	for replica := 0; replica < len(newAssignments[0]); replica++ {
		for partition, replicas := range newAssignments {
			rack := rackAssignments[partition][replica]

			// Build a list of used brokers for this rack on this partition
			var usedRackBrokers []int32
			for r, b := range replicas {
				if rackAssignments[partition][r] == rack && i32.Contains(b, brokersByRack[rack]) {
					usedRackBrokers = append(usedRackBrokers, b)
				}
			}

			// Check if the assigned broker is out of sync with the defined rack
			isLeader := replica == 0
			if !i32.Contains(replicas[replica], brokersByRack[rack]) {
				// Find unused broker IDs for this rack
				unusedRackBrokers := i32.Diff(brokersByRack[rack], usedRackBrokers)

				// Last used broker is not used for rack assignments
				var lastUsedBroker int32 = 0

				// If the chosen broker will be the preferred leader we use leader counts to make sure
				// leaders are balanced across partitions in the assignments
				brokerCounts := replicaCounts
				if isLeader {
					brokerCounts = leaderCounts
				}

				// Find the broker ID to add
				brokerIdToAdd := leastPopulousBroker(unusedRackBrokers, brokerCounts, lastUsedBroker)

				// Replace the broker
				newAssignments[partition][replica] = brokerIdToAdd
			}
			// Update counts
			replicaCounts[newAssignments[partition][replica]]++
			if isLeader {
				leaderCounts[newAssignments[partition][replica]]++
			}
		}
	}

	return newAssignments
}

// Make a copy of partition assignments
func Copy(assignments [][]int32) [][]int32 {
	copy := make([][]int32, len(assignments))
	for i, replicas := range assignments {
		copy[i] = append([]int32{}, replicas...)
	}
	return copy
}

// A map of the broker ID counts for all replicas
func replicaCounts(assignments [][]int32) map[int32]int {
	replicaCounts := make(map[int32]int)
	for _, replicas := range assignments {
		for _, brokerId := range replicas {
			replicaCounts[brokerId]++
		}
	}
	return replicaCounts
}

// A map of the broker ID counts for preferred leaders
func leaderCounts(assignments [][]int32) map[int32]int {
	leaderCounts := make(map[int32]int)
	for _, replicas := range assignments {
		leaderCounts[replicas[0]]++
	}
	return leaderCounts
}

// Find the unused broker with the least replicas of any partition
func leastPopulousBroker(unusedBrokers []int32, brokerCounts map[int32]int, lastUsedBroker int32) int32 {
	sort.Slice(unusedBrokers, func(i, j int) bool {
		// Sort priority:
		// - The least populous broker
		// - Round robin with increasing broker ID
		//
		// To demonstrate why this sort is somewhat complicated, in this example we want the next broker ID to be 2...
		// {1, ?},
		// {2,  },
		// {3,  },
		// ...but in the very next placement cycle we want the next broker ID to be 3, not 1.
		// {1, 2},
		// {2, ?},
		// {3,  },
		//
		// Sort:
		//   Broker i has less replicas than j
		//   OR Broker i has the same number of replicas as j
		//     AND broker ID i and j are both greater than the last used broker for this partition
		//     AND broker ID i is less than j
		//   OR Broker i has the same number of replicas as j
		//     AND either broker ID i or j are less than the last used broker for this partition
		//     AND the difference between broker ID i and the last used broker is greater than that of j
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
