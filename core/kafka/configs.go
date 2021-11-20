// Package kafka implements the Kafka service handling requests and responses.
package kafka

import (
	"context"
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Config operation types.
const (
	SetConfigOperation    int8 = 0
	DeleteConfigOperation int8 = 1
)

// ConfigOperation represents an alter config operation.
type ConfigOperation struct {
	Name  string
	Value *string
	Op    int8 // 0: SET, 1: DELETE, 2: APPEND, 3: SUBTRACT.
}

// ConfigOperations represents a slice of ConfigOperation.
type ConfigOperations []ConfigOperation

// Contains determines if the specified config key name exists.
func (c ConfigOperations) Contains(name string) bool {
	for _, op := range c {
		if op.Name == name {
			return true
		}
	}
	return false
}

// ContainsOp determines if the specified operation type exists.
func (c ConfigOperations) ContainsOp(operation int8) bool {
	for _, op := range c {
		if op.Op == operation {
			return true
		}
	}
	return false
}

func newConfigOps(
	localConfigs def.ConfigsMap,
	remoteConfigsMap def.ConfigsMap,
	remoteConfigs def.Configs,
	deleteUndefinedConfigs bool,
	nonIncremental bool,
) ConfigOperations {
	var configOps ConfigOperations

	for k, v := range localConfigs {
		if cv, ok := remoteConfigsMap[k]; ok {
			// Config exists.
			vv := "null"
			if v != nil {
				vv = *v
			}
			cvv := "null"
			if cv != nil {
				cvv = *cv
			}
			if vv != cvv {
				// Config value has changed.
				log.Debugf("Value of config key %q has changed from %q to %q and will be updated", k, cvv, vv)
				configOps = append(configOps, ConfigOperation{
					Name:  k,
					Value: v,
					Op:    SetConfigOperation,
				})
			}
		} else {
			// Config does not exist.
			log.Debugf("Config key %q is missing from remote config and will be added", k)
			configOps = append(configOps, ConfigOperation{
				Name:  k,
				Value: v,
				Op:    SetConfigOperation,
			})
		}
	}

	// Mark undefined configs for deletion.
	if deleteUndefinedConfigs || nonIncremental {
		for _, config := range remoteConfigs {
			// Ignore static and default config keys that cannot be deleted.
			if config.Source == def.ConfigSourceStaticBrokerConfig || config.Source == def.ConfigSourceDefaultConfig {
				continue
			}
			if _, ok := localConfigs[config.Name]; !ok {
				log.Debugf("Config key %q is missing from local definition and will be deleted", config.Name)
				configOps = append(configOps, ConfigOperation{
					Name: config.Name,
					Op:   DeleteConfigOperation,
				})
			} else if nonIncremental && !configOps.Contains(config.Name) {
				// For non-incremental, make sure all dynamic keys that exist in local are added.
				log.Debugf("Config key %q is unchanged and will be preserved", config.Name)
				configOps = append(configOps, ConfigOperation{
					Name:  config.Name,
					Value: config.Value,
					Op:    SetConfigOperation,
				})
			}
		}
	}

	return configOps
}

// describeBrokerConfigs executes a request to describe broker configs (Kafka 0.11.0+).
func describeBrokerConfigs(cl *client.Client, brokerID string) (def.Configs, error) {
	req := kmsg.NewDescribeConfigsRequest()

	res := kmsg.NewDescribeConfigsRequestResource()
	res.ResourceType = kmsg.ConfigResourceTypeBroker
	res.ResourceName = brokerID
	req.Resources = append(req.Resources, res)

	resp, err := describeConfigs(cl, req)
	if err != nil {
		return nil, err
	}

	return newConfigs(resp[0].Configs), nil
}

// ResourceConfigs represents configs for a named resource.
type ResourceConfigs struct {
	ResourceName string
	Configs      def.Configs
}

// describeTopicConfigs executes a request to describe topic configs (Kafka 0.11.0+).
func describeTopicConfigs(cl *client.Client, topics []string) ([]ResourceConfigs, error) {
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

	resourceConfigs := make([]ResourceConfigs, len(resp))
	for i, resource := range resp {
		resourceConfigs[i] = ResourceConfigs{
			ResourceName: resource.ResourceName,
			Configs:      newConfigs(resource.Configs),
		}
	}

	return resourceConfigs, nil
}

func newConfigs(configsResp []kmsg.DescribeConfigsResponseResourceConfig) def.Configs {
	var configs def.Configs
	for _, c := range configsResp {
		configKey := def.ConfigKey{
			Name:        c.Name,
			Value:       c.Value,
			IsSensitive: c.IsSensitive,
			ReadOnly:    c.ReadOnly,
			Source:      def.ConfigSource(c.Source),
		}

		// Kafka has a category of dynamic configs that have no physical manifestation in the
		// server.properties file and can only be set dynamically. For some reason Kafka marks
		// these keys as read-only, even though they most certainly are not. For this reason
		// it's necessary to additionally check the config key is not dynamic.
		// See https://github.com/twmb/franz-go/issues/91#issuecomment-929872304
		if configKey.ReadOnly && !configKey.IsDynamic() {
			// Ignore all keys that are read-only and not dynamic.
			continue
		}

		configs = append(configs, configKey)
	}
	return configs
}

// describeConfigs executes a request to describe configs (Kafka 0.11.0+).
func describeConfigs(cl *client.Client, req kmsg.DescribeConfigsRequest) ([]kmsg.DescribeConfigsResponseResource, error) {
	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return nil, err
	}
	resp := kresp.(*kmsg.DescribeConfigsResponse)

	if len(resp.Resources) != len(req.Resources) {
		return nil, fmt.Errorf("requested %d resource(s) but received %d", len(req.Resources), len(resp.Resources))
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

// alterBrokerConfigs executes a request to perform a non-incremental alter broker configs (Kafka 0.11.0+).
func alterBrokerConfigs(
	cl *client.Client,
	brokerID string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeBroker
	reqR.ResourceName = brokerID
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

// alterTopicConfigs executes a request to perform a non-incremental alter topic configs (Kafka 0.11.0+).
func alterTopicConfigs(
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

// alterConfigs executes a request to perform a non-incremental alter configs (Kafka 0.11.0+).
func alterConfigs(
	cl *client.Client,
	resources []kmsg.AlterConfigsRequestResource,
	validateOnly bool,
) error {
	req := kmsg.NewAlterConfigsRequest()
	req.Resources = resources
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.AlterConfigsResponse)

	if len(resp.Resources) != 1 {
		return fmt.Errorf("requested %d resource(s) but received %d", 1, len(resp.Resources))
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

// incrementalAlterBrokerConfigs executes a request to perform an incremental alter broker configs (Kafka 2.3.0+).
func incrementalAlterBrokerConfigs(
	cl *client.Client,
	brokerID string,
	configOps ConfigOperations,
	validateOnly bool,
) error {
	reqR := kmsg.NewIncrementalAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeBroker
	reqR.ResourceName = brokerID
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

// incrementalAlterTopicConfigs executes a request to perform an incremental alter topic configs (Kafka 2.3.0+).
func incrementalAlterTopicConfigs(
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

func buildIncrementalAlterConfigsResourceConfig(
	configOps ConfigOperations,
) []kmsg.IncrementalAlterConfigsRequestResourceConfig {
	configs := make([]kmsg.IncrementalAlterConfigsRequestResourceConfig, len(configOps))
	for i, co := range configOps {
		configs[i] = kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  co.Name,
			Value: co.Value,
			Op:    kmsg.IncrementalAlterConfigOp(co.Op),
		}
	}
	return configs
}

// incrementalAlterConfigs executes a request to perform an incremental alter configs (Kafka 2.3.0+).
func incrementalAlterConfigs(
	cl *client.Client,
	resources []kmsg.IncrementalAlterConfigsRequestResource,
	validateOnly bool,
) error {
	req := kmsg.NewIncrementalAlterConfigsRequest()
	req.Resources = resources
	req.ValidateOnly = validateOnly

	kresp, err := cl.Client.Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.IncrementalAlterConfigsResponse)

	if len(resp.Resources) != 1 {
		return fmt.Errorf("requested %d resource(s) but received %d", 1, len(resp.Resources))
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
