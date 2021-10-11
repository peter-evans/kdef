package kafka

import (
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Create a new service
func NewService(
	cl *client.Client,
) *Service {
	srv := &Service{
		cl: cl,
	}
	return srv
}

// A Kafka service
type Service struct {
	// constructor params
	cl *client.Client

	// internal
	incrementalAlter *bool
}

// Determine if incremental alter configs should/can be used
func (s *Service) getIncrementalAlter() (bool, error) {
	if s.incrementalAlter == nil {
		var ia bool
		switch s.cl.AlterConfigsMethod() {
		case "incremental":
			ia = true
		case "non-incremental":
			ia = false
		case "auto":
			log.Debug("Checking if incremental alter configs is supported by the target cluster...")
			r := kmsg.NewIncrementalAlterConfigsRequest()
			var err error
			ia, err = requestIsSupported(s.cl, r.Key())
			if err != nil {
				return false, err
			}
		default:
			// Should never reach here due to client config validation
			return false, fmt.Errorf("invalid alter configs method")
		}
		s.incrementalAlter = &ia
		log.Debug("Incremental alter configs enabled: %v", *s.incrementalAlter)
	}

	return *s.incrementalAlter, nil
}

// ========================= Metadata =========================

// Execute a request for metadata (Kafka 0.8.0+)
func (s *Service) DescribeMetadata(topics []string, errorOnNonExistence bool) (*Metadata, error) {
	return describeMetadata(s.cl, topics, errorOnNonExistence)
}

// Execute describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+)
func (s *Service) IsKafkaReady(minBrokers int, timeoutSec int) bool {
	return isKafkaReady(s.cl, minBrokers, timeoutSec)
}

// ========================= Configs ==========================

// Create alter configs operations
func (s *Service) NewConfigOps(
	localConfigs def.ConfigsMap,
	remoteConfigsMap def.ConfigsMap,
	remoteConfigs def.Configs,
	deleteMissingConfigs bool,
) (ConfigOperations, error) {
	incrementalAlter, err := s.getIncrementalAlter()
	if err != nil {
		return nil, err
	}
	return newConfigOps(
		localConfigs,
		remoteConfigsMap,
		remoteConfigs,
		deleteMissingConfigs,
		!incrementalAlter,
	), nil
}

// Execute a request to describe broker configs (Kafka 0.11.0+)
func (s *Service) DescribeBrokerConfigs(brokerId string) (def.Configs, error) {
	return describeBrokerConfigs(s.cl, brokerId)
}

// Execute a request to describe all broker configs (Kafka 0.11.0+)
func (s *Service) DescribeAllBrokerConfigs() (def.Configs, error) {
	// Empty brokerId returns dynamic config for all brokers (cluster-wide)
	return describeBrokerConfigs(s.cl, "")
}

// Execute a request to alter broker configs (Kafka 0.11.0+/2.3.0+)
func (s *Service) AlterBrokerConfigs(brokerId string, configOps ConfigOperations, validateOnly bool) error {
	incrementalAlter, err := s.getIncrementalAlter()
	if err != nil {
		return err
	}
	if incrementalAlter {
		return incrementalAlterBrokerConfigs(s.cl, brokerId, configOps, validateOnly)
	} else {
		return alterBrokerConfigs(s.cl, brokerId, configOps, validateOnly)
	}
}

// Execute a request to alter cluster-wide broker configs (Kafka 0.11.0+/2.3.0+)
func (s *Service) AlterAllBrokerConfigs(configOps ConfigOperations, validateOnly bool) error {
	return s.AlterBrokerConfigs("", configOps, validateOnly)
}

// Execute a request to describe topic configs (Kafka 0.11.0+)
func (s *Service) DescribeTopicConfigs(topics []string) ([]ResourceConfigs, error) {
	return describeTopicConfigs(s.cl, topics)
}

// Execute a request to alter topic configs (Kafka 0.11.0+/2.3.0+)
func (s *Service) AlterTopicConfigs(topic string, configOps ConfigOperations, validateOnly bool) error {
	incrementalAlter, err := s.getIncrementalAlter()
	if err != nil {
		return err
	}
	if incrementalAlter {
		return incrementalAlterTopicConfigs(s.cl, topic, configOps, validateOnly)
	} else {
		return alterTopicConfigs(s.cl, topic, configOps, validateOnly)
	}
}

// ========================= Topic ============================

// Execute a request for the metadata of a topic that may or may not exist (Kafka 0.11.0+)
func (s *Service) TryRequestTopic(topic string) (
	*def.TopicDefinition,
	def.Configs,
	meta.Brokers,
	error,
) {
	return tryRequestTopic(s.cl, topic)
}

// Execute a request to create a topic (Kafka 0.10.1+)
func (s *Service) CreateTopic(
	topicDef def.TopicDefinition,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	return createTopic(s.cl, topicDef, assignments, validateOnly)
}

// Execute a request to create partitions (Kafka 0.10.0+)
func (s *Service) CreatePartitions(
	topic string,
	partitions int,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	return createPartitions(s.cl, topic, partitions, assignments, validateOnly)
}

// Execute a request to list partition reassignments (Kafka 2.4.0+)
func (s *Service) ListPartitionReassignments(
	topic string,
	partitions []int32,
) (meta.PartitionReassignments, error) {
	return listPartitionReassignments(s.cl, topic, partitions)
}

// Execute a request to alter partition assignments (Kafka 2.4.0+)
func (s *Service) AlterPartitionAssignments(
	topic string,
	assignments def.PartitionAssignments,
) error {
	return alterPartitionAssignments(s.cl, topic, assignments)
}
