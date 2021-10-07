package kafka

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Execute a request for the metadata of a topic that may or may not exist (Kafka 0.11.0+)
func tryRequestTopic(cl *client.Client, topic string) (
	*def.TopicDefinition,
	def.Configs,
	meta.Brokers,
	error,
) {
	metadata, err := describeMetadata(cl, []string{topic}, false)
	if err != nil {
		return nil, nil, nil, err
	}

	// Check if the topic exists
	topicMetadata := metadata.Topics[0]
	if !topicMetadata.Exists {
		return nil, nil, metadata.Brokers, nil
	}

	// Fetch topic configs
	resourceConfigs, err := describeTopicConfigs(cl, []string{topic})
	if err != nil {
		return nil, nil, nil, err
	}
	topicConfigs := resourceConfigs[0].Configs

	// Build topic definition
	topicDef := def.NewTopicDefinition(
		topic,
		topicMetadata.PartitionAssignments,
		topicMetadata.PartitionRackAssignments,
		topicConfigs.ToMap(),
		metadata.Brokers,
		true,
		true,
	)

	return &topicDef, topicConfigs, metadata.Brokers, nil
}

// Execute a request to create a topic (Kafka 0.10.1+)
func createTopic(
	cl *client.Client,
	topicDef def.TopicDefinition,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	var configs []kmsg.CreateTopicsRequestTopicConfig
	for k, v := range topicDef.Spec.Configs {
		configs = append(configs, kmsg.CreateTopicsRequestTopicConfig{
			Name:  k,
			Value: v,
		})
	}

	reqT := kmsg.NewCreateTopicsRequestTopic()
	reqT.Topic = topicDef.Metadata.Name
	reqT.Configs = configs

	if len(assignments) > 0 {
		var assignment []kmsg.CreateTopicsRequestTopicReplicaAssignment
		for i, replicas := range assignments {
			assignment = append(assignment, kmsg.CreateTopicsRequestTopicReplicaAssignment{
				Partition: int32(i),
				Replicas:  replicas,
			})
		}
		reqT.ReplicaAssignment = assignment
		reqT.ReplicationFactor = -1
		reqT.NumPartitions = -1
	} else {
		reqT.ReplicationFactor = int16(topicDef.Spec.ReplicationFactor)
		reqT.NumPartitions = int32(topicDef.Spec.Partitions)
	}

	req := kmsg.NewCreateTopicsRequest()
	req.Topics = append(req.Topics, reqT)
	req.TimeoutMillis = cl.TimeoutMs()
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreateTopicsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topics but received %d", 1, len(resp.Topics))
	}

	for _, topic := range resp.Topics {
		if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
			errMsg := err.Error()
			if topic.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *topic.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}

// Execute a request to create partitions (Kafka 0.10.0+)
func createPartitions(
	cl *client.Client,
	topic string,
	partitions int,
	assignments def.PartitionAssignments,
	validateOnly bool,
) error {
	var assignment []kmsg.CreatePartitionsRequestTopicAssignment
	for _, replicas := range assignments {
		assignment = append(assignment, kmsg.CreatePartitionsRequestTopicAssignment{
			Replicas: replicas,
		})
	}

	t := kmsg.NewCreatePartitionsRequestTopic()
	t.Topic = topic
	t.Count = int32(partitions)
	t.Assignment = assignment

	req := kmsg.NewCreatePartitionsRequest()
	req.Topics = append(req.Topics, t)
	req.TimeoutMillis = cl.TimeoutMs()
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreatePartitionsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topics but received %d", 1, len(resp.Topics))
	}

	for _, topic := range resp.Topics {
		if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
			errMsg := err.Error()
			if topic.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *topic.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}

// Execute a request to alter partition assignments (Kafka 2.4.0+)
func alterPartitionAssignments(
	cl *client.Client,
	topic string,
	assignments def.PartitionAssignments,
) error {
	var partitions []kmsg.AlterPartitionAssignmentsRequestTopicPartition
	for i, replicas := range assignments {
		partitions = append(partitions, kmsg.AlterPartitionAssignmentsRequestTopicPartition{
			Partition: int32(i),
			Replicas:  replicas,
		})
	}

	t := kmsg.NewAlterPartitionAssignmentsRequestTopic()
	t.Topic = topic
	t.Partitions = partitions

	req := kmsg.NewAlterPartitionAssignmentsRequest()
	req.Topics = append(req.Topics, t)
	req.TimeoutMillis = cl.TimeoutMs()

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.AlterPartitionAssignmentsResponse)

	if len(resp.Topics) != 1 {
		return fmt.Errorf("requested %d topics but received %d", 1, len(resp.Topics))
	}

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return fmt.Errorf(errMsg)
	}

	for _, topic := range resp.Topics {
		for _, partition := range topic.Partitions {
			if err := kerr.ErrorForCode(partition.ErrorCode); err != nil {
				errMsg := err.Error()
				if partition.ErrorMessage != nil {
					errMsg = fmt.Sprintf("%s: %s", errMsg, *partition.ErrorMessage)
				}
				return fmt.Errorf(errMsg)
			}
		}
	}

	return nil
}

// Execute a request to list partition reassignments (Kafka 2.4.0+)
func listPartitionReassignments(
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

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.ListPartitionReassignmentsResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		errMsg := err.Error()
		if resp.ErrorMessage != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, *resp.ErrorMessage)
		}
		return nil, fmt.Errorf(errMsg)
	}

	reassignments := meta.PartitionReassignments{}
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
