// Package kafka implements the Kafka service handling requests and responses.
package kafka

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// NewService creates a new Kafka service.
func NewService(
	cl *client.Client,
) *Service {
	return &Service{
		cl: cl,
	}
}

// Service represents a Kafka service.
type Service struct {
	cl               *client.Client
	incrementalAlter *bool
}

func (s *Service) getIncrementalAlter(ctx context.Context) (bool, error) {
	if s.incrementalAlter == nil {
		var ia bool
		switch s.cl.AlterConfigsMethod() {
		case "incremental":
			ia = true
		case "non-incremental":
			ia = false
		case "auto":
			log.Debugf("Checking if incremental alter configs is supported by the target cluster...")
			r := kmsg.NewIncrementalAlterConfigsRequest()
			var err error
			ia, err = requestIsSupported(ctx, s.cl, r.Key())
			if err != nil {
				return false, err
			}
		default:
			// Should never reach here due to client config validation.
			return false, fmt.Errorf("invalid alter configs method")
		}
		s.incrementalAlter = &ia
		log.Debugf("Incremental alter configs enabled: %v", *s.incrementalAlter)
	}

	return *s.incrementalAlter, nil
}

// ========================= Metadata =========================

// DescribeMetadata executes a request for metadata (Kafka 0.8.0+).
func (s *Service) DescribeMetadata(
	ctx context.Context,
	topics []string,
	errorOnNonExistence bool,
) (*Metadata, error) {
	return describeMetadata(ctx, s.cl, topics, errorOnNonExistence)
}

// IsKafkaReady executes describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+).
func (s *Service) IsKafkaReady(ctx context.Context, minBrokers int, timeoutSec int) bool {
	return isKafkaReady(ctx, s.cl, minBrokers, timeoutSec)
}

// ========================= Configs ==========================

// NewConfigOps creates alter configs operations.
func (s *Service) NewConfigOps(
	ctx context.Context,
	localConfigs def.ConfigsMap,
	remoteConfigsMap def.ConfigsMap,
	remoteConfigs def.Configs,
	deleteUndefinedConfigs bool,
) (ConfigOperations, error) {
	incrementalAlter, err := s.getIncrementalAlter(ctx)
	if err != nil {
		return nil, err
	}
	return newConfigOps(
		localConfigs,
		remoteConfigsMap,
		remoteConfigs,
		deleteUndefinedConfigs,
		!incrementalAlter,
	), nil
}

// DescribeBrokerConfigs executes a request to describe broker configs (Kafka 0.11.0+).
func (s *Service) DescribeBrokerConfigs(ctx context.Context, brokerID string) (def.Configs, error) {
	return describeBrokerConfigs(ctx, s.cl, brokerID)
}

// DescribeAllBrokerConfigs executes a request to describe all broker configs (Kafka 0.11.0+).
func (s *Service) DescribeAllBrokerConfigs(ctx context.Context) (def.Configs, error) {
	// Empty brokerID returns dynamic config for all brokers (cluster-wide).
	return describeBrokerConfigs(ctx, s.cl, "")
}

// AlterBrokerConfigs executes a request to alter broker configs (Kafka 0.11.0+/2.3.0+).
func (s *Service) AlterBrokerConfigs(
	ctx context.Context,
	brokerID string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	incrementalAlter, err := s.getIncrementalAlter(ctx)
	if err != nil {
		return err
	}
	if incrementalAlter {
		return incrementalAlterBrokerConfigs(ctx, s.cl, brokerID, configOps, validateOnly)
	}
	return alterBrokerConfigs(ctx, s.cl, brokerID, configOps, validateOnly)
}

// AlterAllBrokerConfigs executes a request to alter cluster-wide broker configs (Kafka 0.11.0+/2.3.0+).
func (s *Service) AlterAllBrokerConfigs(
	ctx context.Context,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	return s.AlterBrokerConfigs(ctx, "", configOps, validateOnly)
}

// DescribeTopicConfigs executes a request to describe topic configs (Kafka 0.11.0+).
func (s *Service) DescribeTopicConfigs(ctx context.Context, topics []string) ([]ResourceConfigs, error) {
	return describeTopicConfigs(ctx, s.cl, topics)
}

// AlterTopicConfigs executes a request to alter topic configs (Kafka 0.11.0+/2.3.0+).
func (s *Service) AlterTopicConfigs(
	ctx context.Context,
	topic string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	incrementalAlter, err := s.getIncrementalAlter(ctx)
	if err != nil {
		return err
	}
	if incrementalAlter {
		return incrementalAlterTopicConfigs(ctx, s.cl, topic, configOps, validateOnly)
	}
	return alterTopicConfigs(ctx, s.cl, topic, configOps, validateOnly)
}

// ========================= Topic ============================

// TryRequestTopic executes a request for the metadata of a topic that may or may not exist (Kafka 0.11.0+).
func (s *Service) TryRequestTopic(ctx context.Context, defMetadata def.ResourceMetadataDefinition) (
	*def.TopicDefinition,
	def.Configs,
	meta.Brokers,
	error,
) {
	return tryRequestTopic(ctx, s.cl, defMetadata)
}

// CreateTopic executes a request to create a topic (Kafka 0.10.1+).
func (s *Service) CreateTopic(
	ctx context.Context,
	topicDef def.TopicDefinition,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	return createTopic(ctx, s.cl, topicDef, assignments, validateOnly)
}

// CreatePartitions executes a request to create partitions (Kafka 0.10.0+).
func (s *Service) CreatePartitions(
	ctx context.Context,
	topic string,
	partitions int,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	return createPartitions(ctx, s.cl, topic, partitions, assignments, validateOnly)
}

// ListPartitionReassignments executes a request to list partition reassignments (Kafka 2.4.0+).
func (s *Service) ListPartitionReassignments(
	ctx context.Context,
	topic string,
	partitions []int32,
) (meta.PartitionReassignments, error) {
	return listPartitionReassignments(ctx, s.cl, topic, partitions)
}

// AlterPartitionAssignments executes a request to alter partition assignments (Kafka 2.4.0+).
func (s *Service) AlterPartitionAssignments(
	ctx context.Context,
	topic string,
	assignments def.PartitionAssignments,
) error {
	return alterPartitionAssignments(ctx, s.cl, topic, assignments)
}

// ========================= ACL =============================

// DescribeResourceACLs executes a request to describe ACLs of a specific resource (Kafka 0.11.0+).
func (s *Service) DescribeResourceACLs(
	ctx context.Context,
	name string,
	resourceType string,
) (def.ACLEntryGroups, error) {
	return describeResourceACLs(ctx, s.cl, name, resourceType)
}

// DescribeAllResourceACLs executes a request to describe ACLs for all resources (Kafka 0.11.0+).
func (s *Service) DescribeAllResourceACLs(
	ctx context.Context,
	resourceType string,
) ([]ResourceACLs, error) {
	return describeAllResourceACLs(ctx, s.cl, resourceType)
}

// CreateACLs executes a request to create ACLs (Kafka 0.11.0+).
func (s *Service) CreateACLs(
	ctx context.Context,
	name string,
	resourceType string,
	acls def.ACLEntryGroups,
) error {
	return createACLs(ctx, s.cl, name, resourceType, acls)
}

// DeleteACLs executes a request to delete acls (Kafka 0.11.0+).
func (s *Service) DeleteACLs(
	ctx context.Context,
	name string,
	resourceType string,
	acls def.ACLEntryGroups,
) error {
	return deleteACLs(ctx, s.cl, name, resourceType, acls)
}
