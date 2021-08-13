package topics

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/util"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

const (
	setConfigOperation    int8 = 0
	deleteConfigOperation int8 = 1
)

// An alter config operation
type configOp struct {
	name         string
	value        *string
	currentValue *string
	op           int8 // 0: SET, 1: DELETE, 2: APPEND, 3: SUBTRACT
}

// A collection of alter config operations
type configOps []configOp

// Determine if the specified operation type exists in the collection
func (c configOps) containsOp(operation int8) bool {
	for _, op := range c {
		if op.op == operation {
			return true
		}
	}
	return false
}

// Flags to configure an applier
type ApplierFlags struct {
	DeleteMissingConfigs bool
	DryRun               bool
	NonIncremental       bool
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	cl      *client.Client
	yamlDoc string
	flags   ApplierFlags

	// internal
	configOperations  configOps
	create            bool
	topicDef          TopicDefinition
	existingTopicDef  TopicDefinition
	topicConfigsResp  kmsg.DescribeConfigsResponseResource
	topicMetadataResp kmsg.MetadataResponseTopic
}

// Creates a new applier
func NewApplier(
	cl *client.Client,
	yamlDoc string,
	flags ApplierFlags,
) *applier {
	return &applier{
		cl:      cl,
		yamlDoc: yamlDoc,
		flags:   flags,
	}
}

// Executes the apply operation
func (a *applier) Execute() error {
	log.Debug("Validating topic definition")
	if err := yaml.Unmarshal([]byte(a.yamlDoc), &a.topicDef); err != nil {
		return err
	}
	if err := a.topicDef.Validate(); err != nil {
		return err
	}

	if err := a.tryFetchExistingTopic(); err != nil {
		return err
	}

	if a.create {
		return a.createTopic()
	} else {
		return a.updateTopic()
	}
}

// Fetch metadata and config for a topic if it exists
func (a *applier) tryFetchExistingTopic() error {
	log.Info("Checking if topic %q exists...", a.topicDef.Metadata.Name)
	topicMetadataResp, err := requestTopicMetadata(a.cl, []string{a.topicDef.Metadata.Name}, false)
	if err != nil {
		return err
	}
	a.topicMetadataResp = topicMetadataResp[0]

	a.create = a.topicMetadataResp.ErrorCode == kerr.UnknownTopicOrPartition.Code
	if a.create {
		log.Debug("Topic %q does not exist", a.topicDef.Metadata.Name)
		return nil
	}

	log.Info("Fetching configs for existing topic %q...", a.topicDef.Metadata.Name)
	topicConfigsResp, err := requestDescribeTopicConfigs(a.cl, []string{a.topicDef.Metadata.Name})
	if err != nil {
		return err
	}
	a.topicConfigsResp = topicConfigsResp[0]

	a.existingTopicDef = topicDefinitionFromMetadata(
		a.topicMetadataResp,
		a.topicConfigsResp,
	)

	return nil
}

// Create a topic (Kafka 0.10.1+)
func (a *applier) createTopic() error {
	if a.flags.DryRun {
		log.InfoWithKey("dry-run", "Creating topic %q...", a.topicDef.Metadata.Name)
	} else {
		log.Info("Creating topic %q...", a.topicDef.Metadata.Name)
	}

	var configs []kmsg.CreateTopicsRequestTopicConfig
	for k, v := range a.topicDef.Spec.Configs {
		configs = append(configs, kmsg.CreateTopicsRequestTopicConfig{
			Name:  k,
			Value: v,
		})
	}

	reqT := kmsg.NewCreateTopicsRequestTopic()
	reqT.Topic = a.topicDef.Metadata.Name
	reqT.ReplicationFactor = int16(a.topicDef.Spec.ReplicationFactor)
	reqT.NumPartitions = int32(a.topicDef.Spec.Partitions)
	reqT.Configs = configs

	req := kmsg.NewCreateTopicsRequest()
	req.Topics = append(req.Topics, reqT)
	req.TimeoutMillis = a.cl.TimeoutMs()
	req.ValidateOnly = a.flags.DryRun

	kresp, err := a.cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.CreateTopicsResponse)

	if err := kerr.ErrorForCode(resp.Topics[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Topics[0].ErrorMessage)
	} else {
		if a.flags.DryRun {
			log.InfoWithKey("dry-run", "Created topic %q", a.topicDef.Metadata.Name)
		} else {
			log.Info("Created topic %q", a.topicDef.Metadata.Name)
		}
	}

	return nil
}

// Update a topic
func (a *applier) updateTopic() error {
	// Build config operations and determine if an update is required
	a.buildConfigOperations()

	if len(a.configOperations) > 0 {
		a.displayConfigOperations()

		if err := a.updateConfigs(); err != nil {
			return err
		}
	}

	// TODO
	// Support update of partitions (addition only)
	// Support update of replication factor(?)

	return nil
}

// Build alter configs operations
func (a *applier) buildConfigOperations() {
	log.Debug("Comparing definition configs with existing topic %q", a.topicDef.Metadata.Name)

	for k, v := range a.topicDef.Spec.Configs {
		if cv, ok := a.existingTopicDef.Spec.Configs[k]; ok {
			// config exists
			if *v != *cv {
				// config value has changed
				log.Debug("Value of config key %q has changed from %q to %q", k, *cv, *v)
				a.configOperations = append(a.configOperations, configOp{
					name:         k,
					value:        v,
					currentValue: cv,
					op:           setConfigOperation,
				})
			}
		} else {
			// config does not exist
			log.Debug("Config key %q is missing from existing config", k)
			a.configOperations = append(a.configOperations, configOp{
				name:  k,
				value: v,
				op:    setConfigOperation,
			})
		}
	}

	// mark missing configs for deletion
	if a.flags.DeleteMissingConfigs || a.flags.NonIncremental {
		for _, config := range a.topicConfigsResp.Configs {
			// ignore static and default config keys that cannot be deleted
			if config.Source == kmsg.ConfigSourceStaticBrokerConfig || config.Source == kmsg.ConfigSourceDefaultConfig {
				continue
			}
			if _, ok := a.topicDef.Spec.Configs[config.Name]; !ok {
				log.Debug("Config key %q is missing from definition", config.Name)
				a.configOperations = append(a.configOperations, configOp{
					name:         config.Name,
					currentValue: config.Value,
					op:           deleteConfigOperation,
				})
			}
		}
	}
}

// Display alter configs operations
func (a *applier) displayConfigOperations() {
	log.Info("The following changes will be applied to topic %q configs:", a.topicDef.Metadata.Name)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Key", "Current Value", "New Value", "Operation"})
	for _, op := range a.configOperations {
		if op.op == deleteConfigOperation {
			t.AppendRow([]interface{}{op.name, util.DerefStr(op.currentValue), util.DerefStr(op.value), color.RedString("DELETE")})
		} else {
			t.AppendRow([]interface{}{op.name, util.DerefStr(op.currentValue), util.DerefStr(op.value), color.CyanString("SET")})
		}
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

// Update topic configs
func (a *applier) updateConfigs() error {
	if a.flags.NonIncremental {
		if a.configOperations.containsOp(deleteConfigOperation) && !a.flags.DeleteMissingConfigs {
			return errors.New("cannot apply operations containing deletions because flag --delete-missing-configs is not set")
		}

		if err := a.alterConfigs(); err != nil {
			return err
		}
	} else {
		log.Debug("Checking if incremental alter configs is supported by the target cluster...")
		req := kmsg.NewIncrementalAlterConfigsRequest()
		supported, err := requestSupported(a.cl, req.Key())
		if err != nil {
			return err
		}

		if supported {
			if err := a.incrementalAlterConfigs(); err != nil {
				return err
			}
		} else {
			log.Info("The target cluster does not support incremental alter configs (Kafka 2.3.0+)")
			log.Info("Set flag --non-inc to use the non-incremental alter configs method")
			return errors.New("api unsupported by the target cluster")
		}
	}

	return nil
}

// Perform a non-incremental alter configs (Kafka 0.11.0+)
func (a *applier) alterConfigs() error {
	if a.flags.DryRun {
		log.InfoWithKey("dry-run", "Altering configs (non-incremental)...")
	} else {
		log.Info("Altering configs (non-incremental)...")
	}

	var configs []kmsg.AlterConfigsRequestResourceConfig
	for _, co := range a.configOperations {
		if co.op != deleteConfigOperation {
			configs = append(configs, kmsg.AlterConfigsRequestResourceConfig{
				Name:  co.name,
				Value: co.value,
			})
		}
	}

	reqR := kmsg.NewAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = a.topicDef.Metadata.Name
	reqR.Configs = configs

	req := kmsg.NewAlterConfigsRequest()
	req.Resources = append(req.Resources, reqR)
	req.ValidateOnly = a.flags.DryRun

	kresp, err := a.cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.AlterConfigsResponse)

	if err := kerr.ErrorForCode(resp.Resources[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Resources[0].ErrorMessage)
	} else {
		if a.flags.DryRun {
			log.InfoWithKey("dry-run", "Altered configs for topic %q", a.topicDef.Metadata.Name)
		} else {
			log.Info("Altered configs for topic %q", a.topicDef.Metadata.Name)
		}
	}

	return nil
}

// Perform an incremental alter configs (Kafka 2.3.0+)
func (a *applier) incrementalAlterConfigs() error {
	if a.flags.DryRun {
		log.InfoWithKey("dry-run", "Altering configs...")
	} else {
		log.Info("Altering configs...")
	}

	var configs []kmsg.IncrementalAlterConfigsRequestResourceConfig
	for _, co := range a.configOperations {
		configs = append(configs, kmsg.IncrementalAlterConfigsRequestResourceConfig{
			Name:  co.name,
			Value: co.value,
			Op:    co.op,
		})
	}

	reqR := kmsg.NewIncrementalAlterConfigsRequestResource()
	reqR.ResourceType = kmsg.ConfigResourceTypeTopic
	reqR.ResourceName = a.topicDef.Metadata.Name
	reqR.Configs = configs

	req := kmsg.NewIncrementalAlterConfigsRequest()
	req.Resources = append(req.Resources, reqR)
	req.ValidateOnly = a.flags.DryRun

	kresp, err := a.cl.Client().Request(context.Background(), &req)
	if err != nil {
		return err
	}
	resp := kresp.(*kmsg.IncrementalAlterConfigsResponse)

	if err := kerr.ErrorForCode(resp.Resources[0].ErrorCode); err != nil {
		return fmt.Errorf(*resp.Resources[0].ErrorMessage)
	} else {
		if a.flags.DryRun {
			log.InfoWithKey("dry-run", "Altered configs for topic %q", a.topicDef.Metadata.Name)
		} else {
			log.Info("Altered configs for topic %q", a.topicDef.Metadata.Name)
		}
	}

	return nil
}
