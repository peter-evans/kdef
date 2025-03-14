// Package kafka implements the Kafka service handling requests and responses.
package kafka

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// tryRequestTopic executes a request for the metadata of a topic that may or may not exist (Kafka 0.11.0+).
func tryRequestTopic(
	ctx context.Context,
	cl *client.Client,
	defMetadata def.ResourceMetadataDefinition,
) (
	*def.TopicDefinition,
	def.Configs,
	def.PartitionAssignments,
	meta.Brokers,
	error,
) {
	metadata, err := describeMetadata(ctx, cl, []string{defMetadata.Name}, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Check if the topic exists
	topicMetadata := metadata.Topics[0]
	if !topicMetadata.Exists {
		return nil, nil, nil, metadata.Brokers, nil
	}
	partitionLeaders := metadata.Topics[0].PartitionLeaders
	partitionISR := metadata.Topics[0].PartitionISR

	// Fetch topic configs
	resourceConfigs, err := describeTopicConfigs(ctx, cl, []string{defMetadata.Name})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	topicConfigs := resourceConfigs[0].Configs

	// Build topic definition
	topicDef := def.NewTopicDefinition(
		defMetadata,
		topicMetadata.PartitionAssignments,
		topicMetadata.PartitionRacks,
		partitionLeaders,
		topicConfigs.ToMap(),
		true,
		true,
		true,
	)

	return &topicDef, topicConfigs, partitionISR, metadata.Brokers, nil
}

// createTopic executes a request to create a topic (Kafka 0.10.1+).
func createTopic(
	ctx context.Context,
	cl *client.Client,
	topicDef def.TopicDefinition,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	configs := make([]kmsg.CreateTopicsRequestTopicConfig, len(topicDef.Spec.Configs))
	i := 0
	for k, v := range topicDef.Spec.Configs {
		configs[i] = kmsg.CreateTopicsRequestTopicConfig{
			Name:  k,
			Value: v,
		}
		i++
	}

	reqT := kmsg.NewCreateTopicsRequestTopic()
	reqT.Topic = topicDef.Metadata.Name
	reqT.Configs = configs

	assignment := make([]kmsg.CreateTopicsRequestTopicReplicaAssignment, len(assignments))
	for i, replicas := range assignments {
		assignment[i] = kmsg.CreateTopicsRequestTopicReplicaAssignment{
			Partition: int32(i),
			Replicas:  replicas,
		}
	}
	reqT.ReplicaAssignment = assignment
	reqT.ReplicationFactor = -1
	reqT.NumPartitions = -1

	req := kmsg.NewCreateTopicsRequest()
	req.Topics = append(req.Topics, reqT)
	req.TimeoutMillis = cl.TimeoutMs()
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreateTopicsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topic(s) but received %d", 1, len(resp.Topics))
	}

	for _, topic := range resp.Topics {
		if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
			errMsg := err.Error()
			if topic.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *topic.ErrorMessage)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}

	return nil
}

// createPartitions executes a request to create partitions (Kafka 0.10.0+).
func createPartitions(
	ctx context.Context,
	cl *client.Client,
	topic string,
	partitions int,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	assignment := make([]kmsg.CreatePartitionsRequestTopicAssignment, len(assignments))
	for i, replicas := range assignments {
		assignment[i] = kmsg.CreatePartitionsRequestTopicAssignment{
			Replicas: replicas,
		}
	}

	t := kmsg.NewCreatePartitionsRequestTopic()
	t.Topic = topic
	t.Count = int32(partitions)
	t.Assignment = assignment

	req := kmsg.NewCreatePartitionsRequest()
	req.Topics = append(req.Topics, t)
	req.TimeoutMillis = cl.TimeoutMs()
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreatePartitionsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topic(s) but received %d", 1, len(resp.Topics))
	}

	for _, topic := range resp.Topics {
		if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
			errMsg := err.Error()
			if topic.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *topic.ErrorMessage)
			}
			return fmt.Errorf("%s", errMsg)
		}
	}

	return nil
}

// alterPartitionAssignments executes a request to alter partition assignments (Kafka 2.4.0+).
func alterPartitionAssignments(
	ctx context.Context,
	cl *client.Client,
	topic string,
	assignments def.PartitionAssignments,
) error {
	partitions := make([]kmsg.AlterPartitionAssignmentsRequestTopicPartition, len(assignments))
	for i, replicas := range assignments {
		partitions[i] = kmsg.AlterPartitionAssignmentsRequestTopicPartition{
			Partition: int32(i),
			Replicas:  replicas,
		}
	}

	t := kmsg.NewAlterPartitionAssignmentsRequestTopic()
	t.Topic = topic
	t.Partitions = partitions

	req := kmsg.NewAlterPartitionAssignmentsRequest()
	req.Topics = append(req.Topics, t)
	req.TimeoutMillis = cl.TimeoutMs()

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.AlterPartitionAssignmentsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topic(s) but received %d", 1, len(resp.Topics))
	}

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return fmt.Errorf("%s", errMsg)
	}

	for _, topic := range resp.Topics {
		for _, partition := range topic.Partitions {
			if err := kerr.ErrorForCode(partition.ErrorCode); err != nil {
				errMsg := err.Error()
				if partition.ErrorMessage != nil {
					errMsg = fmt.Sprintf("%s: %s", errMsg, *partition.ErrorMessage)
				}
				return fmt.Errorf("%s", errMsg)
			}
		}
	}

	return nil
}

// listPartitionReassignments executes a request to list partition reassignments (Kafka 2.4.0+).
func listPartitionReassignments(
	ctx context.Context,
	cl *client.Client,
	topic string,
	partitions []int32,
) (meta.PartitionReassignments, error) {
	t := kmsg.NewListPartitionReassignmentsRequestTopic()
	t.Topic = topic
	t.Partitions = partitions

	req := kmsg.NewListPartitionReassignmentsRequest()
	req.Topics = append(req.Topics, t)
	req.TimeoutMillis = cl.TimeoutMs()

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.ListPartitionReassignmentsResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	var reassignments meta.PartitionReassignments
	if len(resp.Topics) > 0 {
		for _, p := range resp.Topics[0].Partitions {
			reassignments = append(reassignments, meta.PartitionReassignment{
				Partition:        p.Partition,
				Replicas:         p.Replicas,
				AddingReplicas:   p.AddingReplicas,
				RemovingReplicas: p.RemovingReplicas,
			})
		}
	}

	reassignments.Sort()

	return reassignments, nil
}

// electLeaders executes a request to elect preferred partition leaders (Kafka 2.4.0+).
func electLeaders(
	ctx context.Context,
	cl *client.Client,
	topic string,
	partitions []int32,
) error {
	reqT := kmsg.NewElectLeadersRequestTopic()
	reqT.Topic = topic
	reqT.Partitions = partitions

	req := kmsg.NewElectLeadersRequest()
	req.Topics = []kmsg.ElectLeadersRequestTopic{reqT}
	req.ElectionType = 0
	req.TimeoutMillis = cl.TimeoutMs()

	kresp, err := cl.Client.Request(ctx, &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.ElectLeadersResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topic(s) but received %d", 1, len(resp.Topics))
	}

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		return err
	}

	for _, topic := range resp.Topics {
		for _, partition := range topic.Partitions {
			if err := kerr.ErrorForCode(partition.ErrorCode); err != nil {
				if partition.ErrorCode == kerr.ElectionNotNeeded.Code {
					// The preferred leader is already the leader.
					continue
				}
				errMsg := err.Error()
				if partition.ErrorMessage != nil {
					errMsg = fmt.Sprintf("%s: %s", errMsg, *partition.ErrorMessage)
				}
				return fmt.Errorf("%s", errMsg)
			}
		}
	}

	return nil
}
