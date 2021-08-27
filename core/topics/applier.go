package topics

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/req"
	"github.com/peter-evans/kdef/util"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Flags to configure an applier
type ApplierFlags struct {
	DeleteMissingConfigs bool
	DryRun               bool
	ExitCode             bool
	NonIncremental       bool
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	cl      *client.Client
	yamlDoc string
	flags   ApplierFlags

	// internal
	configOps         req.ConfigOperations
	partitionsOp      bool
	createOp          bool
	topicDef          def.TopicDefinition
	existingTopicDef  def.TopicDefinition
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

	// Determine if the topic exists
	if err := a.tryFetchExistingTopic(); err != nil {
		return err
	}

	if !a.createOp {
		// Build topic update operations and determine if updates are required
		if err := a.buildOperations(); err != nil {
			return err
		}
	}

	if a.pendingOpsExist() {
		// Display pending operations
		if !log.Quiet {
			a.displayPendingOps()
		}

		// Execute operations
		if err := a.executeOperations(); err != nil {
			return err
		}

		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Completed apply for topic %q", a.topicDef.Metadata.Name)
		if a.flags.DryRun && a.flags.ExitCode {
			return fmt.Errorf("unapplied changes exist for topic %q", a.topicDef.Metadata.Name)
		}
	} else {
		log.Info("No changes to apply for topic %q", a.topicDef.Metadata.Name)
	}

	return nil
}

// Fetch metadata and config for a topic if it exists
func (a *applier) tryFetchExistingTopic() error {
	log.Info("Checking if topic %q exists...", a.topicDef.Metadata.Name)
	topicMetadataResp, err := req.RequestTopicMetadata(a.cl, []string{a.topicDef.Metadata.Name}, false)
	if err != nil {
		return err
	}
	a.topicMetadataResp = topicMetadataResp[0]

	a.createOp = a.topicMetadataResp.ErrorCode == kerr.UnknownTopicOrPartition.Code
	if a.createOp {
		log.Debug("Topic %q does not exist", a.topicDef.Metadata.Name)
		return nil
	}

	log.Info("Fetching configs for existing topic %q...", a.topicDef.Metadata.Name)
	topicConfigsResp, err := req.RequestDescribeTopicConfigs(a.cl, []string{a.topicDef.Metadata.Name})
	if err != nil {
		return err
	}
	a.topicConfigsResp = topicConfigsResp[0]

	a.existingTopicDef = def.TopicDefinitionFromMetadata(
		a.topicMetadataResp,
		a.topicConfigsResp,
	)

	return nil
}

// Build topic update operations
func (a *applier) buildOperations() error {
	a.buildConfigOps()

	if err := a.buildPartitionsOperation(); err != nil {
		return err
	}

	return nil
}

// Determine if pending operations exist
func (a *applier) pendingOpsExist() bool {
	if a.createOp {
		return true
	}

	if len(a.configOps) > 0 {
		return true
	}

	if a.partitionsOp {
		return true
	}

	return false
}

// Display pending operations
func (a *applier) displayPendingOps() {
	if a.createOp {
		log.Info("Topic %q does not exist and will be created", a.topicDef.Metadata.Name)
	}

	if len(a.configOps) > 0 {
		a.displayConfigOps()
	}

	if a.partitionsOp {
		log.Info(
			"The number of partitions for topic %q will be increased from %d to %d",
			a.topicDef.Metadata.Name,
			a.existingTopicDef.Spec.Partitions,
			a.topicDef.Spec.Partitions,
		)
	}
}

// Build topic update operations
func (a *applier) executeOperations() error {
	if a.createOp {
		if err := a.createTopic(); err != nil {
			return err
		}
	}

	if len(a.configOps) > 0 {
		if err := a.updateConfigs(); err != nil {
			return err
		}
	}

	if a.partitionsOp {
		if err := a.updatePartitions(); err != nil {
			return err
		}
	}

	return nil
}

// Execute a request to create a topic
func (a *applier) createTopic() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Creating topic %q...", a.topicDef.Metadata.Name)

	if err := req.RequestCreateTopic(a.cl, a.topicDef, a.flags.DryRun); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Created topic %q", a.topicDef.Metadata.Name)
	}

	return nil
}

// Build alter configs operations
func (a *applier) buildConfigOps() {
	log.Debug("Comparing definition configs with existing topic %q", a.topicDef.Metadata.Name)

	for k, v := range a.topicDef.Spec.Configs {
		if cv, ok := a.existingTopicDef.Spec.Configs[k]; ok {
			// Config exists
			if *v != *cv {
				// Config value has changed
				log.Debug("Value of config key %q has changed from %q to %q", k, *cv, *v)
				a.configOps = append(a.configOps, req.ConfigOperation{
					Name:         k,
					Value:        v,
					CurrentValue: cv,
					Op:           req.SetConfigOperation,
				})
			}
		} else {
			// Config does not exist
			log.Debug("Config key %q is missing from existing config", k)
			a.configOps = append(a.configOps, req.ConfigOperation{
				Name:  k,
				Value: v,
				Op:    req.SetConfigOperation,
			})
		}
	}

	// Mark missing configs for deletion
	if a.flags.DeleteMissingConfigs || a.flags.NonIncremental {
		for _, config := range a.topicConfigsResp.Configs {
			// Ignore static and default config keys that cannot be deleted
			if config.Source == kmsg.ConfigSourceStaticBrokerConfig || config.Source == kmsg.ConfigSourceDefaultConfig {
				continue
			}
			if _, ok := a.topicDef.Spec.Configs[config.Name]; !ok {
				log.Debug("Config key %q is missing from definition", config.Name)
				a.configOps = append(a.configOps, req.ConfigOperation{
					Name:         config.Name,
					CurrentValue: config.Value,
					Op:           req.DeleteConfigOperation,
				})
			}
		}
	}

	// Sort the config operations
	a.configOps.Sort()
}

// Display alter configs operations
func (a *applier) displayConfigOps() {
	log.Info("The following changes will be applied to topic %q configs:", a.topicDef.Metadata.Name)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Key", "Current Value", "New Value", "Operation"})
	for _, op := range a.configOps {
		if op.Op == req.DeleteConfigOperation {
			t.AppendRow([]interface{}{op.Name, util.DerefStr(op.CurrentValue), util.DerefStr(op.Value), color.RedString("DELETE")})
		} else {
			t.AppendRow([]interface{}{op.Name, util.DerefStr(op.CurrentValue), util.DerefStr(op.Value), color.CyanString("SET")})
		}
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

// Update topic configs
func (a *applier) updateConfigs() error {
	if a.flags.NonIncremental {
		if a.configOps.ContainsOp(req.DeleteConfigOperation) && !a.flags.DeleteMissingConfigs {
			return errors.New("cannot apply operations containing deletions because flag --delete-missing-configs is not set")
		}

		if err := a.alterConfigs(); err != nil {
			return err
		}
	} else {
		log.Debug("Checking if incremental alter configs is supported by the target cluster...")
		r := kmsg.NewIncrementalAlterConfigsRequest()
		supported, err := req.RequestSupported(a.cl, r.Key())
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

// Execute a request to perform a non-incremental alter configs
func (a *applier) alterConfigs() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altering configs (non-incremental)...")

	if err := req.RequestAlterConfigs(a.cl, a.topicDef.Metadata.Name, a.configOps, a.flags.DryRun); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for topic %q", a.topicDef.Metadata.Name)
	}

	return nil
}

// Execute a request to perform an incremental alter configs
func (a *applier) incrementalAlterConfigs() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altering configs...")

	if err := req.RequestIncrementalAlterConfigs(a.cl, a.topicDef.Metadata.Name, a.configOps, a.flags.DryRun); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for topic %q", a.topicDef.Metadata.Name)
	}

	return nil
}

// Build partitions operation
func (a *applier) buildPartitionsOperation() error {
	a.partitionsOp = false

	if a.topicDef.Spec.Partitions == a.existingTopicDef.Spec.Partitions {
		// No change
		return nil
	}
	if a.topicDef.Spec.Partitions < a.existingTopicDef.Spec.Partitions {
		return fmt.Errorf("decreasing the number of partitions is not supported")
	}

	// Flag for operation
	a.partitionsOp = true

	return nil
}

// Execute a request to create partitions
func (a *applier) updatePartitions() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Creating partitions...")

	if err := req.RequestCreatePartitions(a.cl, a.topicDef.Metadata.Name, a.topicDef.Spec.Partitions, a.flags.DryRun); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Created partitions for topic %q", a.topicDef.Metadata.Name)
	}

	return nil
}
