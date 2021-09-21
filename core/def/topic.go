package def

import (
	"encoding/json"
	"fmt"

	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/util/diff"
	"github.com/peter-evans/kdef/util/i32"
	"github.com/peter-evans/kdef/util/str"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Top-level topic definition
type TopicDefinition struct {
	ApiVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   TopicMetadataDefinition `json:"metadata"`
	Spec       TopicSpecDefinition     `json:"spec"`
}

// Topic metadata definition
type TopicMetadataDefinition struct {
	Labels TopicMetadataLabels `json:"labels,omitempty"`
	Name   string              `json:"name"`
}

// Topic metadata labels
type TopicMetadataLabels map[string]string

// Topic spec definition
type TopicSpecDefinition struct {
	Configs           TopicConfigs                `json:"configs,omitempty"`
	Partitions        int                         `json:"partitions"`
	ReplicationFactor int                         `json:"replicationFactor"`
	Assignments       PartitionAssignments        `json:"assignments,omitempty"`
	RackAssignments   PartitionRackAssignments    `json:"rackAssignments,omitempty"`
	Reassignment      TopicReassignmentDefinition `json:"reassignment,omitempty"`
}

// Topic configs
type TopicConfigs map[string]*string

// Topic assignments
type PartitionAssignments [][]int32

// Topic rack assignments
type PartitionRackAssignments [][]string

// Topic reassignment definition
type TopicReassignmentDefinition struct {
	AwaitTimeoutSec int `json:"awaitTimeoutSec"`
}

// Determine if a spec has assignments
func (s TopicSpecDefinition) HasAssignments() bool {
	return len(s.Assignments) > 0
}

// Determine if a spec has rack assignments
func (s TopicSpecDefinition) HasRackAssignments() bool {
	return len(s.RackAssignments) > 0
}

// Convert a topic definition to JSON
func (t TopicDefinition) JSON() (string, error) {
	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// Create a copy of this TopicDefinition
func (t TopicDefinition) Copy() TopicDefinition {
	copiers := copy.New()
	copier := copiers.Get(&TopicDefinition{}, &TopicDefinition{})
	topicDefCopy := TopicDefinition{}
	copier.Copy(&topicDefCopy, &t)
	return topicDefCopy
}

// Validate a topic definition
func (t TopicDefinition) Validate() error {
	if len(t.Metadata.Name) == 0 {
		return fmt.Errorf("metadata name must be supplied")
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

			for _, rackId := range replicas {
				if len(rackId) == 0 {
					return fmt.Errorf("rack ids cannot be an empty string")
				}
			}
		}
	}

	if t.Spec.Reassignment.AwaitTimeoutSec < 0 {
		return fmt.Errorf("reassignment await timeout seconds must be greater or equal to 0")
	}

	return nil
}

// Further validate a topic definition using metadata
func (t TopicDefinition) ValidateWithMetadata(brokers Brokers) error {
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
				log.Warn("unable to use broker id %q in rack assignments because it has no rack id", fmt.Sprint(broker.Id))
			}
		}

		// Build a map of brokers by rack ID
		brokersByRack := brokers.BrokersByRack()

		// Check the rack IDs in the rack assignments are valid
		for partition, replicas := range t.Spec.RackAssignments {
			rackIdCounts := make(map[string]int)
			for _, rackId := range replicas {
				if !str.Contains(rackId, brokers.Racks()) {
					return fmt.Errorf("invalid rack id %q in rack assignments", rackId)
				}
				rackIdCounts[rackId]++
			}

			// Check there are enough available brokers for the number of times a rack ID has been used in this partition
			// e.g. if rack id "zone-a" is specified for three replicas in the same partition, but "zone-a" only contains
			// two brokers, then the assignment is not possible.
			for rackId, count := range rackIdCounts {
				rackBrokerCount := len(brokersByRack[rackId])
				if count > rackBrokerCount {
					return fmt.Errorf(
						"rack id %q contains %d brokers, but is specified for %d replicas in partition %d",
						rackId,
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
	metadataResp kmsg.MetadataResponseTopic,
	topicConfigsResp kmsg.DescribeConfigsResponseResource,
	brokers Brokers,
	includeAssignments bool,
	includeRackAssignments bool,
) TopicDefinition {
	topicConfigs := TopicConfigs{}
	for _, config := range topicConfigsResp.Configs {
		topicConfigs[config.Name] = config.Value
	}

	topicDef := TopicDefinition{
		ApiVersion: "v1",
		Kind:       "topic",
		Metadata: TopicMetadataDefinition{
			Name: *metadataResp.Topic,
		},
		Spec: TopicSpecDefinition{
			Partitions:        len(metadataResp.Partitions),
			ReplicationFactor: len(metadataResp.Partitions[0].Replicas),
			Configs:           topicConfigs,
		},
	}

	if includeAssignments {
		topicDef.Spec.Assignments = assignmentsDefinitionFromMetadata(metadataResp.Partitions)
	}
	if includeRackAssignments {
		topicDef.Spec.RackAssignments = rackAssignmentsDefinitionFromMetadata(
			assignmentsDefinitionFromMetadata(metadataResp.Partitions),
			brokers,
		)
	}

	return topicDef
}

// Create an assignments definition from metadata
func assignmentsDefinitionFromMetadata(
	partitions []kmsg.MetadataResponseTopicPartition,
) PartitionAssignments {
	assignments := make(PartitionAssignments, len(partitions))
	for _, p := range partitions {
		assignments[p.Partition] = p.Replicas
	}

	return assignments
}

// Create a rack assignments definition from assignments and broker metadata
func rackAssignmentsDefinitionFromMetadata(
	assignments PartitionAssignments,
	brokers Brokers,
) PartitionRackAssignments {
	racksByBroker := brokers.RacksByBroker()
	rackAssignments := make(PartitionRackAssignments, len(assignments))
	for i, p := range assignments {
		replicas := make([]string, len(p))
		for j, r := range p {
			replicas[j] = racksByBroker[r]
		}
		rackAssignments[i] = replicas
	}

	return rackAssignments
}

// Compute the line-oriented diff between the JSON representation of two definitions
func DiffTopicDefinitions(a *TopicDefinition, b *TopicDefinition) (string, error) {
	// Convert definition to JSON handling null pointers
	toJson := func(t *TopicDefinition) (string, error) {
		j := "null"
		if t != nil {
			var err error
			j, err = t.JSON()
			if err != nil {
				return "", err
			}
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

	diff := diff.LineOriented(aJson, bJson)

	return diff, nil
}
