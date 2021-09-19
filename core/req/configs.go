package req

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/client"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
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

// Determine if the specified config key name exists in the collection
func (c ConfigOperations) Contains(name string) bool {
	for _, op := range c {
		if op.Name == name {
			return true
		}
	}
	return false
}

// Determine if the specified operation type exists in the collection
func (c ConfigOperations) ContainsOp(operation int8) bool {
	for _, op := range c {
		if op.Op == operation {
			return true
		}
	}
	return false
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

	if len(resp.Resources) != 1 {
		return fmt.Errorf("requested %d resources but received %d", 1, len(resp.Resources))
	}

	for _, resource := range resp.Resources {
		if err := kerr.ErrorForCode(resource.ErrorCode); err != nil {
			return fmt.Errorf(*resource.ErrorMessage)
		}
	}

	return nil
}

// Execute a request to perform an incremental alter configs (Kafka 2.3.0+)
func RequestIncrementalAlterConfigs(
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

	if len(resp.Resources) != 1 {
		return fmt.Errorf("requested %d resources but received %d", 1, len(resp.Resources))
	}

	for _, resource := range resp.Resources {
		if err := kerr.ErrorForCode(resource.ErrorCode); err != nil {
			return fmt.Errorf(*resource.ErrorMessage)
		}
	}

	return nil
}
