package def

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/util/diff"
	"github.com/peter-evans/kdef/util/i32"
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
	Reassignment      TopicReassignmentDefinition `json:"reassignment,omitempty"`
}

// Topic configs definition
type TopicConfigs map[string]*string

// Topic assignments definition
type PartitionAssignments [][]int32

// Topic reassignment definition
type TopicReassignmentDefinition struct {
	AwaitTimeoutSec int `json:"awaitTimeoutSec"`
}

// Determines if a spec has assignments
func (s TopicSpecDefinition) HasAssignments() bool {
	return len(s.Assignments) > 0
}

// Converts a topic definition to YAML
func (t TopicDefinition) YAML() (string, error) {
	y, err := yaml.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(y), nil
}

// Converts a topic definition to JSON
func (t TopicDefinition) JSON() (string, error) {
	j, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// Creates a copy of this TopicDefinition
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

	if t.Spec.Reassignment.AwaitTimeoutSec < 0 {
		return fmt.Errorf("reassignment await timeout seconds must be greater or equal to 0")
	}

	return nil
}

// Further topic definition validation using metadata
func (t TopicDefinition) ValidateWithMetadata(brokers []int32) error {
	// Note:
	// These are validations that are applicable regardless of whether it's a create or update operation
	// Validation specific to either create or update can remain in the applier

	if t.Spec.ReplicationFactor > len(brokers) {
		return fmt.Errorf("replication factor cannot exceed the number of available brokers")
	}

	if t.Spec.HasAssignments() {
		// Check the broker IDs in the assignments are valid
		brokerIds := make(map[int32]bool, len(brokers))
		for _, broker := range brokers {
			brokerIds[broker] = true
		}

		for _, replicas := range t.Spec.Assignments {
			for _, id := range replicas {
				if _, exists := brokerIds[id]; !exists {
					return fmt.Errorf("invalid broker id %q in assignments", fmt.Sprint(id))
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
	forExport bool,
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

	if !forExport {
		topicDef.Spec.Assignments = assignmentsDefinitionFromMetadata(metadataResp.Partitions)
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
