package def

import (
	"fmt"

	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/util/i32"
	"github.com/peter-evans/kdef/core/util/str"
)

// Topic assignments
type PartitionAssignments [][]int32

// Topic rack assignments
type PartitionRackAssignments [][]string

// Topic spec definition
type TopicSpecDefinition struct {
	Configs                ConfigsMap               `json:"configs,omitempty"`
	DeleteUndefinedConfigs bool                     `json:"deleteUndefinedConfigs"`
	Partitions             int                      `json:"partitions"`
	ReplicationFactor      int                      `json:"replicationFactor"`
	Assignments            PartitionAssignments     `json:"assignments,omitempty"`
	RackAssignments        PartitionRackAssignments `json:"rackAssignments,omitempty"`
}

// Determine if a spec has assignments
func (s TopicSpecDefinition) HasAssignments() bool {
	return len(s.Assignments) > 0
}

// Determine if a spec has rack assignments
func (s TopicSpecDefinition) HasRackAssignments() bool {
	return len(s.RackAssignments) > 0
}

// Top-level topic definition
type TopicDefinition struct {
	ResourceDefinition
	Spec TopicSpecDefinition `json:"spec"`
}

// Create a copy of this TopicDefinition
func (t TopicDefinition) Copy() TopicDefinition {
	copiers := copy.New()
	copier := copiers.Get(&TopicDefinition{}, &TopicDefinition{})
	topicDefCopy := TopicDefinition{}
	copier.Copy(&topicDefCopy, &t)
	return topicDefCopy
}

// Validate definition
func (t TopicDefinition) Validate() error {
	if err := t.ValidateResource(); err != nil {
		return err
	}

	if t.Spec.Partitions <= 0 {
		return fmt.Errorf("partitions must be greater than 0")
	}

	if t.Spec.ReplicationFactor <= 0 {
		return fmt.Errorf("replication factor must be greater than 0")
	}

	if t.Spec.HasAssignments() && t.Spec.HasRackAssignments() {
		return fmt.Errorf("assignments and rack assignments cannot be specified together")
	}

	if t.Spec.HasAssignments() {
		if len(t.Spec.Assignments) != t.Spec.Partitions {
			return fmt.Errorf("number of replica assignments must match partitions")
		}

		for _, replicas := range t.Spec.Assignments {
			if len(replicas) != t.Spec.ReplicationFactor {
				return fmt.Errorf("number of replicas in each assignment must match replication factor")
			}

			if i32.ContainsDuplicate(replicas) {
				return fmt.Errorf("a replica assignment cannot contain duplicate brokers")
			}
		}
	}

	if t.Spec.HasRackAssignments() {
		if len(t.Spec.RackAssignments) != t.Spec.Partitions {
			return fmt.Errorf("number of rack assignments must match partitions")
		}

		for _, replicas := range t.Spec.RackAssignments {
			if len(replicas) != t.Spec.ReplicationFactor {
				return fmt.Errorf("number of replicas in each rack assignment must match replication factor")
			}

			for _, rackID := range replicas {
				if len(rackID) == 0 {
					return fmt.Errorf("rack ids cannot be an empty string")
				}
			}
		}
	}

	return nil
}

// Further validate definition using metadata
func (t TopicDefinition) ValidateWithMetadata(brokers meta.Brokers) error {
	// Note:
	// These are validations that are applicable regardless of whether it's a create or update operation
	// Validation specific to either create or update can remain in the applier

	if t.Spec.ReplicationFactor > len(brokers) {
		return fmt.Errorf("replication factor cannot exceed the number of available brokers")
	}

	if t.Spec.HasAssignments() {
		// Check the broker IDs in the assignments are valid
		for _, replicas := range t.Spec.Assignments {
			for _, id := range replicas {
				if !i32.Contains(id, brokers.Ids()) {
					return fmt.Errorf("invalid broker id %q in assignments", fmt.Sprint(id))
				}
			}
		}
	}

	if t.Spec.HasRackAssignments() {
		// Warn if the cluster has no rack ID set on brokers
		for _, broker := range brokers {
			if len(broker.Rack) == 0 {
				log.Warnf("unable to use broker id %q in rack assignments because it has no rack id", fmt.Sprint(broker.ID))
			}
		}

		// Build a map of brokers by rack ID
		brokersByRack := brokers.BrokersByRack()

		// Check the rack IDs in the rack assignments are valid
		for partition, replicas := range t.Spec.RackAssignments {
			rackIDCounts := make(map[string]int)
			for _, rackID := range replicas {
				if !str.Contains(rackID, brokers.Racks()) {
					return fmt.Errorf("invalid rack id %q in rack assignments", rackID)
				}
				rackIDCounts[rackID]++
			}

			// Check there are enough available brokers for the number of times a rack ID has been used in this partition
			// e.g. if rack id "zone-a" is specified for three replicas in the same partition, but "zone-a" only contains
			// two brokers, then the assignment is not possible.
			for rackID, count := range rackIDCounts {
				rackBrokerCount := len(brokersByRack[rackID])
				if count > rackBrokerCount {
					return fmt.Errorf(
						"rack id %q contains %d brokers, but is specified for %d replicas in partition %d",
						rackID,
						rackBrokerCount,
						count,
						partition,
					)
				}
			}
		}
	}

	return nil
}

// Create a topic definition from metadata and config
func NewTopicDefinition(
	name string,
	partitionAssignments PartitionAssignments,
	partitionRackAssignments PartitionRackAssignments,
	configsMap ConfigsMap,
	brokers meta.Brokers,
	includeAssignments bool,
	includeRackAssignments bool,
) TopicDefinition {
	topicDef := TopicDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "topic",
			Metadata: ResourceMetadataDefinition{
				Name: name,
			},
		},
		Spec: TopicSpecDefinition{
			Partitions:        len(partitionAssignments),
			ReplicationFactor: len(partitionAssignments[0]),
			Configs:           configsMap,
		},
	}

	if includeAssignments {
		topicDef.Spec.Assignments = partitionAssignments
	}
	if includeRackAssignments {
		topicDef.Spec.RackAssignments = partitionRackAssignments
	}

	return topicDef
}
