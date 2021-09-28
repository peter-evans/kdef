package service

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	SetConfigOperation    int8 = 0
	DeleteConfigOperation int8 = 1
)

// An alter config operation
type ConfigOperation struct {
	Name  string
	Value *string
	Op    int8 // 0: SET, 1: DELETE, 2: APPEND, 3: SUBTRACT
}

// An array of ConfigOperation
type ConfigOperations []ConfigOperation

// Determine if the specified config key name exists
func (c ConfigOperations) Contains(name string) bool {
	for _, op := range c {
		if op.Name == name {
			return true
		}
	}
	return false
}

// Determine if the specified operation type exists
func (c ConfigOperations) ContainsOp(operation int8) bool {
	for _, op := range c {
		if op.Op == operation {
			return true
		}
	}
	return false
}

// Create alter configs operations
func NewConfigOps(
	localConfigs def.ConfigsMap,
	remoteConfigsMap def.ConfigsMap,
	remoteConfigs def.Configs,
	deleteMissingConfigs bool,
	nonIncremental bool,
) ConfigOperations {
	var configOps ConfigOperations

	for k, v := range localConfigs {
		if cv, ok := remoteConfigsMap[k]; ok {
			// Config exists
			// TODO: handle nil for sensitive values (applies to single broker target)
			if *v != *cv {
				// Config value has changed
				log.Debug("Value of config key %q has changed from %q to %q and will be updated", k, *cv, *v)
				configOps = append(configOps, ConfigOperation{
					Name:  k,
					Value: v,
					Op:    SetConfigOperation,
				})
			}
		} else {
			// Config does not exist
			log.Debug("Config key %q is missing from remote config and will be added", k)
			configOps = append(configOps, ConfigOperation{
				Name:  k,
				Value: v,
				Op:    SetConfigOperation,
			})
		}
	}

	// Mark missing configs for deletion
	if deleteMissingConfigs || nonIncremental {
		for _, config := range remoteConfigs {
			// Ignore static and default config keys that cannot be deleted
			if config.Source == def.ConfigSourceStaticBrokerConfig || config.Source == def.ConfigSourceDefaultConfig {
				continue
			}
			if _, ok := localConfigs[config.Name]; !ok {
				log.Debug("Config key %q is missing from local definition and will be deleted", config.Name)
				configOps = append(configOps, ConfigOperation{
					Name: config.Name,
					Op:   DeleteConfigOperation,
				})
			} else {
				// For non-incremental, make sure all dynamic keys that exist in local are added
				if nonIncremental && !configOps.Contains(config.Name) {
					log.Debug("Config key %q is unchanged and will be preserved", config.Name)
					configOps = append(configOps, ConfigOperation{
						Name:  config.Name,
						Value: config.Value,
						Op:    SetConfigOperation,
					})
				}
			}
		}
	}

	return configOps
}

// Execute a request to describe all broker configs (Kafka 0.11.0+)
func DescribeAllBrokerConfigs(cl *client.Client) (def.Configs, error) {
	req := kmsg.NewDescribeConfigsRequest()

	res := kmsg.NewDescribeConfigsRequestResource()
	res.ResourceType = kmsg.ConfigResourceTypeBroker
	req.Resources = append(req.Resources, res)

	resp, err := describeConfigs(cl, req)
	if err != nil {
		return nil, err
	}

	return newConfigs(resp[0].Configs), nil
}

// Configs for a named resource
type ResourceConfigs struct {
	ResourceName string
	Configs      def.Configs
}

// Execute a request to describe topic configs (Kafka 0.11.0+)
func DescribeTopicConfigs(cl *client.Client, topics []string) ([]ResourceConfigs, error) {
	req := kmsg.NewDescribeConfigsRequest()

	for _, topic := range topics {
		res := kmsg.NewDescribeConfigsRequestResource()
		res.ResourceType = kmsg.ConfigResourceTypeTopic
		res.ResourceName = topic
		req.Resources = append(req.Resources, res)
	}

	resp, err := describeConfigs(cl, req)
	if err != nil {
		return nil, err
	}

	var resourceConfigs []ResourceConfigs
	for _, resource := range resp {
		resourceConfigs = append(resourceConfigs, ResourceConfigs{
			ResourceName: resource.ResourceName,
			Configs:      newConfigs(resource.Configs),
		})
	}

	return resourceConfigs, nil
}

// Create new configs from API response
func newConfigs(configsResp []kmsg.DescribeConfigsResponseResourceConfig) def.Configs {
	var configs def.Configs
	for _, c := range configsResp {
		configs = append(configs, def.ConfigItem{
			Name:        c.Name,
			Value:       c.Value,
			IsSensitive: c.IsSensitive,
			ReadOnly:    c.ReadOnly,
			Source:      def.ConfigSource(c.Source),
		})
	}
	return configs
}

// Execute a request to describe configs (Kafka 0.11.0+)
func describeConfigs(cl *client.Client, req kmsg.DescribeConfigsRequest) ([]kmsg.DescribeConfigsResponseResource, error) {
	kresp, err := cl.Client().Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeConfigsResponse)

	if len(resp.Resources) != len(req.Resources) {
		return nil, fmt.Errorf("requested %d resources but received %d", len(req.Resources), len(resp.Resources))
	}

	for _, resource := range resp.Resources {
		if err := kerr.ErrorForCode(resource.ErrorCode); err != nil {
			errMsg := err.Error()
			if resource.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *resource.ErrorMessage)
			}
			return nil, fmt.Errorf(errMsg)
		}
	}

	return resp.Resources, nil
}

// Execute a request to perform a non-incremental alter all broker configs (Kafka 0.11.0+)
func AlterAllBrokerConfigs(
	cl *client.Client,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeBroker
	reqR.Configs = buildAlterConfigsResourceConfig(configOps)

	if err := alterConfigs(
		cl,
		[]kmsg.AlterConfigsRequestResource{reqR},
		validateOnly,
	); err != nil {
		return err
	}

	return nil
}

// Execute a request to perform a non-incremental alter topic configs (Kafka 0.11.0+)
func AlterTopicConfigs(
	cl *client.Client,
	topic string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = topic
	reqR.Configs = buildAlterConfigsResourceConfig(configOps)

	if err := alterConfigs(
		cl,
		[]kmsg.AlterConfigsRequestResource{reqR},
		validateOnly,
	); err != nil {
		return err
	}

	return nil
}

// Build configs for an alter configs request
func buildAlterConfigsResourceConfig(
	configOps ConfigOperations,
) []kmsg.AlterConfigsRequestResourceConfig {
	var configs []kmsg.AlterConfigsRequestResourceConfig
	for _, co := range configOps {
		if co.Op != DeleteConfigOperation {
			configs = append(configs, kmsg.AlterConfigsRequestResourceConfig{
				Name:  co.Name,
				Value: co.Value,
			})
		}
	}
	return configs
}

// Execute a request to perform a non-incremental alter configs (Kafka 0.11.0+)
func alterConfigs(
	cl *client.Client,
	resources []kmsg.AlterConfigsRequestResource,
	validateOnly bool,
) error {
	req := kmsg.NewAlterConfigsRequest()
	req.Resources = resources
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
			errMsg := err.Error()
			if resource.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *resource.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}

// Execute a request to perform an incremental alter all broker configs (Kafka 2.3.0+)
func IncrementalAlterAllBrokerConfigs(
	cl *client.Client,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewIncrementalAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeBroker
	reqR.Configs = buildIncrementalAlterConfigsResourceConfig(configOps)

	if err := incrementalAlterConfigs(
		cl,
		[]kmsg.IncrementalAlterConfigsRequestResource{reqR},
		validateOnly,
	); err != nil {
		return err
	}

	return nil
}

// Execute a request to perform an incremental alter topic configs (Kafka 2.3.0+)
func IncrementalAlterTopicConfigs(
	cl *client.Client,
	topic string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewIncrementalAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = topic
	reqR.Configs = buildIncrementalAlterConfigsResourceConfig(configOps)

	if err := incrementalAlterConfigs(
		cl,
		[]kmsg.IncrementalAlterConfigsRequestResource{reqR},
		validateOnly,
	); err != nil {
		return err
	}

	return nil
}

// Build configs for an incremental alter configs request
func buildIncrementalAlterConfigsResourceConfig(
	configOps ConfigOperations,
) []kmsg.IncrementalAlterConfigsRequestResourceConfig {
	var configs []kmsg.IncrementalAlterConfigsRequestResourceConfig
	for _, co := range configOps {
		configs = append(configs, kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  co.Name,
			Value: co.Value,
			Op:    co.Op,
		})
	}
	return configs
}

// Execute a request to perform an incremental alter configs (Kafka 2.3.0+)
func incrementalAlterConfigs(
	cl *client.Client,
	resources []kmsg.IncrementalAlterConfigsRequestResource,
	validateOnly bool,
) error {
	req := kmsg.NewIncrementalAlterConfigsRequest()
	req.Resources = resources
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
			errMsg := err.Error()
			if resource.ErrorMessage != nil {
				errMsg = fmt.Sprintf("%s: %s", errMsg, *resource.ErrorMessage)
			}
			return fmt.Errorf(errMsg)
		}
	}

	return nil
}
