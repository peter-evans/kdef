// Package def implements definitions for Kafka resources.
package def

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/util/i32"
	"github.com/peter-evans/kdef/core/util/str"
)

const (
	BalanceNew = "new"
	BalanceAll = "all"
)

var balanceScopes = []string{
	BalanceNew,
	BalanceAll,
}

const (
	SelectionTopicClusterUse = "topic-cluster-use"
	SelectionTopicUse        = "topic-use"
)

var selectionMethods = []string{
	SelectionTopicClusterUse,
	SelectionTopicUse,
}

// PartitionAssignments represents partition assignments by broker ID.
type PartitionAssignments [][]int32

// PartitionRacks represents assigned racks for partitions.
type PartitionRacks [][]string

// PartitionLeaders represents partition leaders by broker ID.
type PartitionLeaders []int32

// ManagedAssignmentsDefinition represents a managed assignments definition.
type ManagedAssignmentsDefinition struct {
	Balance         string         `json:"balance,omitempty"`
	Selection       string         `json:"selection,omitempty"`
	RackConstraints PartitionRacks `json:"rackConstraints,omitempty"`
}

// HasRackConstraints determines if a managed assignments definition has rack constraints.
func (m ManagedAssignmentsDefinition) HasRackConstraints() bool {
	return len(m.RackConstraints) > 0
}

// TopicSpecDefinition represents a topic spec definition.
type TopicSpecDefinition struct {
	Configs                ConfigsMap                    `json:"configs,omitempty"`
	DeleteUndefinedConfigs bool                          `json:"deleteUndefinedConfigs"`
	Partitions             int                           `json:"partitions"`
	ReplicationFactor      int                           `json:"replicationFactor"`
	Assignments            PartitionAssignments          `json:"assignments,omitempty"`
	ManagedAssignments     *ManagedAssignmentsDefinition `json:"managedAssignments,omitempty"`
	MaintainLeaders        bool                          `json:"maintainLeaders"`
}

// HasAssignments determines if a spec has assignments.
func (t TopicSpecDefinition) HasAssignments() bool {
	return len(t.Assignments) > 0
}

// HasManagedAssignments determines if a spec has a managed assignments definition.
func (t TopicSpecDefinition) HasManagedAssignments() bool {
	return t.ManagedAssignments != nil
}

// TopicStateDefinition represents a topic state definition.
type TopicStateDefinition struct {
	Assignments PartitionAssignments `json:"assignments,omitempty"`
	Leaders     PartitionLeaders     `json:"leaders,omitempty"`
}

// TopicDefinition represents a topic resource definition.
type TopicDefinition struct {
	ResourceDefinition
	Spec  TopicSpecDefinition   `json:"spec"`
	State *TopicStateDefinition `json:"state,omitempty"`
}

// Copy creates a copy of this TopicDefinition.
func (t TopicDefinition) Copy() TopicDefinition {
	copiers := copy.New()
	copier := copiers.Get(&TopicDefinition{}, &TopicDefinition{})
	var topicDefCopy TopicDefinition
	copier.Copy(&topicDefCopy, &t)
	return topicDefCopy
}

// Validate validates the definition.
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

	if t.Spec.HasAssignments() && t.Spec.HasManagedAssignments() {
		return fmt.Errorf("assignments and managed assignments cannot be specified together")
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

	if t.Spec.HasManagedAssignments() {
		if !str.Contains(t.Spec.ManagedAssignments.Balance, balanceScopes) {
			return fmt.Errorf("balance must be one of %q", strings.Join(balanceScopes, "|"))
		}

		if !str.Contains(t.Spec.ManagedAssignments.Selection, selectionMethods) {
			return fmt.Errorf("selection must be one of %q", strings.Join(selectionMethods, "|"))
		}

		if t.Spec.ManagedAssignments.HasRackConstraints() {
			if len(t.Spec.ManagedAssignments.RackConstraints) != t.Spec.Partitions {
				return fmt.Errorf("number of rack constraints must match partitions")
			}

			for _, replicas := range t.Spec.ManagedAssignments.RackConstraints {
				if len(replicas) != t.Spec.ReplicationFactor {
					return fmt.Errorf("number of replicas in a partition's rack constraints must match replication factor")
				}

				for _, rackID := range replicas {
					if len(rackID) == 0 {
						return fmt.Errorf("rack ids cannot be an empty string")
					}
				}
			}
		}
	}

	return nil
}

// ValidateWithMetadata further validates the definition using metadata.
func (t TopicDefinition) ValidateWithMetadata(brokers meta.Brokers) error {
	// These are validations that are applicable regardless of whether it's a create or update operation.
	// Validation specific to either create or update can remain in the applier.

	if t.Spec.ReplicationFactor > len(brokers) {
		return fmt.Errorf("replication factor cannot exceed the number of available brokers")
	}

	if t.Spec.HasAssignments() {
		// Check the broker IDs in the assignments are valid.
		for _, replicas := range t.Spec.Assignments {
			for _, id := range replicas {
				if !i32.Contains(id, brokers.IDs()) {
					return fmt.Errorf("invalid broker id %q in assignments", fmt.Sprint(id))
				}
			}
		}
	}

	if t.Spec.HasManagedAssignments() && t.Spec.ManagedAssignments.HasRackConstraints() {
		// Warn if the cluster has no rack ID set on brokers.
		for _, broker := range brokers {
			if len(broker.Rack) == 0 {
				log.Warnf("unable to use broker id %q in rack constraints because it has no rack id", fmt.Sprint(broker.ID))
			}
		}

		brokersByRack := brokers.BrokersByRack()

		// Check the rack IDs in the rack constraints are valid.
		for partition, replicas := range t.Spec.ManagedAssignments.RackConstraints {
			rackIDCounts := make(map[string]int)
			for _, rackID := range replicas {
				if !str.Contains(rackID, brokers.Racks()) {
					return fmt.Errorf("invalid rack id %q in rack constraints", rackID)
				}
				rackIDCounts[rackID]++
			}

			// Check there are enough available brokers for the number of times a rack ID has been used in this partition.
			// e.g. if rack id "zone-a" is specified for three replicas in the same partition, but "zone-a" only contains
			// two brokers, then the constraint is not possible.
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

// NewTopicDefinition creates a topic definition from metadata and config.
func NewTopicDefinition(
	metadata ResourceMetadataDefinition,
	partitionAssignments PartitionAssignments,
	rackConstraints PartitionRacks,
	partitionLeaders PartitionLeaders,
	configsMap ConfigsMap,
	includeAssignments bool,
	includeRackConstraints bool,
	includeState bool,
) TopicDefinition {
	topicDef := TopicDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "topic",
			Metadata:   metadata,
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
	if includeRackConstraints {
		topicDef.Spec.ManagedAssignments = &ManagedAssignmentsDefinition{
			RackConstraints: rackConstraints,
		}
	}
	if includeState {
		topicDef.State = &TopicStateDefinition{
			Assignments: partitionAssignments,
			Leaders:     partitionLeaders,
		}
	}

	return topicDef
}

// LoadTopicDefinition loads a topic definition from a document.
func LoadTopicDefinition(
	defDoc string,
	format opt.DefinitionFormat,
	propOverrides []string,
) (TopicDefinition, error) {
	var def TopicDefinition

	switch format {
	case opt.YAMLFormat:
		if err := yaml.Unmarshal([]byte(defDoc), &def); err != nil {
			return def, err
		}
	case opt.JSONFormat:
		if err := json.Unmarshal([]byte(defDoc), &def); err != nil {
			return def, err
		}
	default:
		return def, fmt.Errorf("unsupported format")
	}

	// Set defaults
	if !def.Spec.HasAssignments() {
		if def.Spec.HasManagedAssignments() {
			if len(def.Spec.ManagedAssignments.Balance) == 0 {
				def.Spec.ManagedAssignments.Balance = BalanceNew
			}
			if len(def.Spec.ManagedAssignments.Selection) == 0 {
				def.Spec.ManagedAssignments.Selection = SelectionTopicClusterUse
			}
		} else {
			def.Spec.ManagedAssignments = &ManagedAssignmentsDefinition{
				Balance:   BalanceNew,
				Selection: SelectionTopicClusterUse,
			}
		}
	}

	// Apply property overrides
	for _, po := range propOverrides {
		if !strings.HasPrefix(po, "topic.") {
			continue
		}
		kv := strings.SplitN(po, "=", 2)
		if len(kv) != 2 {
			return def, fmt.Errorf("property override %q not a 'key=value' pair", po)
		}
		log.Debugf("Setting property override %q", po)
		switch kv[0] {
		case "topic.spec.managedAssignments.balance":
			def.Spec.ManagedAssignments.Balance = kv[1]
		case "topic.spec.maintainLeaders":
			maintainLeaders, err := strconv.ParseBool(kv[1])
			if err != nil {
				return def, fmt.Errorf("value %q is not valid for property %q", kv[1], kv[0])
			}
			def.Spec.MaintainLeaders = maintainLeaders
		default:
			return def, fmt.Errorf("property %q is not overridable", kv[0])
		}
	}

	def.State = nil

	return def, nil
}
