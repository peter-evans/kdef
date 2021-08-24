package req

import (
	"context"
	"fmt"
	"time"

	"github.com/bradfitz/slice"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/kversion"
)

const (
	SetConfigOperation    int8 = 0
	DeleteConfigOperation int8 = 1
)

// An alter config operation
type ConfigOperation struct {
	Name         string
	Value        *string
	CurrentValue *string
	Op           int8 // 0: SET, 1: DELETE, 2: APPEND, 3: SUBTRACT
}

// A collection of alter config operations
type ConfigOperations []ConfigOperation

// Determine if the specified operation type exists in the collection
func (c ConfigOperations) ContainsOp(operation int8) bool {
	for _, op := range c {
		if op.Op == operation {
			return true
		}
	}
	return false
}

// Sort the collection
func (c ConfigOperations) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8
	slice.Sort(c[:], func(i, j int) bool {
		return c[i].Name < c[j].Name
	})
}

// Execute a request for topic metadata (Kafka 0.8.0+)
func RequestTopicMetadata(cl *client.Client, topics []string, validateResponse bool) ([]kmsg.MetadataResponseTopic, error) {
	req := kmsg.NewMetadataRequest()

	for _, topic := range topics {
		t := kmsg.NewMetadataRequestTopic()
		t.Topic = kmsg.StringPtr(topic)
		req.Topics = append(req.Topics, t)
	}

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.MetadataResponse)

	if len(topics) != 0 && len(resp.Topics) != len(topics) {
		return nil, fmt.Errorf("requested %d topics but received %d", len(topics), len(resp.Topics))
	}

	if validateResponse {
		for _, topic := range resp.Topics {
			if err := kerr.ErrorForCode(topic.ErrorCode); err != nil {
				return nil, err
			}
		}
	}

	return resp.Topics, nil
}

// Execute a request to describe topic configs (Kafka 0.11.0+)
func RequestDescribeTopicConfigs(cl *client.Client, topics []string) ([]kmsg.DescribeConfigsResponseResource, error) {
	req := kmsg.NewDescribeConfigsRequest()

	for _, topic := range topics {
		res := kmsg.NewDescribeConfigsRequestResource()
		res.ResourceType = kmsg.ConfigResourceTypeTopic
		res.ResourceName = topic
		req.Resources = append(req.Resources, res)
	}

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeConfigsResponse)

	if len(resp.Resources) != len(topics) {
		return nil, fmt.Errorf("requested %d resources but received %d", len(topics), len(resp.Resources))
	}

	for _, resource := range resp.Resources {
		if err := kerr.ErrorForCode(resource.ErrorCode); err != nil {
			return nil, fmt.Errorf(*resource.ErrorMessage)
		}
	}

	return resp.Resources, nil
}

// Execute a request to determine if a request key is supported by the cluster (Kafka 0.10.0+)
func RequestSupported(cl *client.Client, requestKey int16) (bool, error) {
	req := kmsg.NewApiVersionsRequest()
	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return false, err
	}
	resp := kresp.(*kmsg.ApiVersionsResponse)
	return kversion.FromApiVersionsResponse(resp).HasKey(requestKey), nil
}

// Execute a request to describe the cluster (Kafka 2.8.0+)
func RequestDescribeCluster(cl *client.Client) (*kmsg.DescribeClusterResponse, error) {
	kresp, err := cl.Client().Request(context.Background(), kmsg.NewPtrDescribeClusterRequest())
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeClusterResponse)

	if err := kerr.ErrorForCode(resp.ErrorCode); err != nil {
		return nil, fmt.Errorf(*resp.ErrorMessage)
	}

	return resp, nil
}

// Execute a request to create a topic (Kafka 0.10.1+)
func RequestCreateTopic(
	cl *client.Client,
	topicDef def.TopicDefinition,
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
	reqT.ReplicationFactor = int16(topicDef.Spec.ReplicationFactor)
	reqT.NumPartitions = int32(topicDef.Spec.Partitions)
	reqT.Configs = configs

	req := kmsg.NewCreateTopicsRequest()
	req.Topics = append(req.Topics, reqT)
	req.TimeoutMillis = cl.TimeoutMs()
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreateTopicsResponse)

	if err := kerr.ErrorForCode(resp.Topics[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Topics[0].ErrorMessage)
	}

	return nil
}

// Execute a request to perform a non-incremental alter configs (Kafka 0.11.0+)
func RequestAlterConfigs(
	cl *client.Client,
	topic string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	var configs []kmsg.AlterConfigsRequestResourceConfig
	for _, co := range configOps {
		if co.Op != DeleteConfigOperation {
			configs = append(configs, kmsg.AlterConfigsRequestResourceConfig{
				Name:  co.Name,
				Value: co.Value,
			})
		}
	}

	reqR := kmsg.NewAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = topic
	reqR.Configs = configs

	req := kmsg.NewAlterConfigsRequest()
	req.Resources = append(req.Resources, reqR)
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.AlterConfigsResponse)

	if err := kerr.ErrorForCode(resp.Resources[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Resources[0].ErrorMessage)
	}

	return nil
}

// Execute a request to perform an incremental alter configs (Kafka 2.3.0+)
func incrementalAlterConfigs(
	cl *client.Client,
	topic string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	var configs []kmsg.IncrementalAlterConfigsRequestResourceConfig
	for _, co := range configOps {
		configs = append(configs, kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  co.Name,
			Value: co.Value,
			Op:    co.Op,
		})
	}

	reqR := kmsg.NewIncrementalAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = topic
	reqR.Configs = configs

	req := kmsg.NewIncrementalAlterConfigsRequest()
	req.Resources = append(req.Resources, reqR)
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.IncrementalAlterConfigsResponse)

	if err := kerr.ErrorForCode(resp.Resources[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Resources[0].ErrorMessage)
	}

	return nil
}

// Execute a request to create partitions (Kafka 0.10.0+)
func RequestCreatePartitions(
	cl *client.Client,
	topic string,
	partitions int,
	validateOnly bool,
) error {
	t := kmsg.NewCreatePartitionsRequestTopic()
	t.Topic = topic
	t.Count = int32(partitions)

	req := kmsg.NewCreatePartitionsRequest()
	req.Topics = append(req.Topics, t)
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

// Execute describe cluster requests until a minimum number of brokers are alive (Kafka 2.8.0+)
func IsKafkaReady(cl *client.Client, minBrokers int, timeoutSec int) bool {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	for {
		select {
		case <-timeout:
			return false
		default:
			resp, err := RequestDescribeCluster(cl)
			if err == nil {
				if len(resp.Brokers) >= minBrokers {
					return true
				}
			}
			time.Sleep(2 * time.Second)
			continue
		}
	}
}
