package req

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/util/str"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Execute a request for the metadata of a topic that may or may not exist (Kafka 0.11.0+)
func TryRequestTopic(cl *client.Client, topic string) (
	*def.TopicDefinition,
	*kmsg.DescribeConfigsResponseResource, // TODO: Can I refactor this out?
	def.Brokers,
	error,
) {
	metadataResp, err := RequestMetadata(cl, []string{topic}, false)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build broker/rack metadata
	var brokers def.Brokers
	for _, broker := range metadataResp.Brokers {
		brokers = append(brokers, def.Broker{
			Id:   broker.NodeID,
			Rack: str.Deref(broker.Rack),
		})
	}

	// Check if the topic exists
	topicMetadata := metadataResp.Topics[0]
	exists := topicMetadata.ErrorCode != kerr.UnknownTopicOrPartition.Code
	if !exists {
		return nil, nil, brokers, nil
	}
	if err := kerr.ErrorForCode(topicMetadata.ErrorCode); err != nil {
		return nil, nil, nil, err
	}

	// Fetch topic configs
	topicConfigsResp, err := RequestDescribeTopicConfigs(cl, []string{topic})
	if err != nil {
		return nil, nil, nil, err
	}
	topicConfigs := topicConfigsResp[0]

	// Build topic definition
	topicDef := def.NewTopicDefinition(
		topicMetadata,
		topicConfigs,
		brokers,
		true,
		true,
	)

	return &topicDef, &topicConfigs, brokers, nil
}

// Execute a request to create a topic (Kafka 0.10.1+)
func RequestCreateTopic(
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
			return fmt.Errorf(*topic.ErrorMessage)
		}
	}

	return nil
}

// Execute a request to create partitions (Kafka 0.10.0+)
func RequestCreatePartitions(
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
			return fmt.Errorf(*topic.ErrorMessage)
		}
	}

	return nil
}

// Execute a request to alter partition assignments (Kafka 2.4.0+)
func RequestAlterPartitionAssignments(
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
		return fmt.Errorf(*resp.ErrorMessage)
	}

	for _, topic := range resp.Topics {
		for _, partition := range topic.Partitions {
			if err := kerr.ErrorForCode(partition.ErrorCode); err != nil {
				return fmt.Errorf(*partition.ErrorMessage)
			}
		}
	}

	return nil
}

// Execute a request to list partition reassignments (Kafka 2.4.0+)
func RequestListPartitionReassignments(
	cl *client.Client,
	topic string,
	partitions []int32,
) ([]kmsg.ListPartitionReassignmentsResponseTopicPartition, error) {
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
		return nil, fmt.Errorf(*resp.ErrorMessage)
	}

	if len(resp.Topics) > 0 {
		return resp.Topics[0].Partitions, nil
	} else {
		return []kmsg.ListPartitionReassignmentsResponseTopicPartition{}, nil
	}
}
