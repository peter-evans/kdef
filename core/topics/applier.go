package topics

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/req"
	"github.com/peter-evans/kdef/core/res"
	"github.com/peter-evans/kdef/util"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

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

	// result
	res res.ApplyResult

	// internal
	configOps          req.ConfigOperations
	createOp           bool
	partitionsOp       bool
	assignmentsOp      bool
	remoteTopicConfigs kmsg.DescribeConfigsResponseResource
	metadata           *kmsg.MetadataResponse
	localDef           def.TopicDefinition
	remoteDef          def.TopicDefinition
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

// Executes the applier
func (a *applier) Execute() *res.ApplyResult {
	if err := a.apply(); err != nil {
		a.res.Err = err.Error()
		log.Error(err)
	}

	if !a.flags.DryRun {
		a.res.Applied = true
	}

	return &a.res
}

// Performs the apply operation sequence
func (a *applier) apply() error {
	log.Debug("Validating topic definition")
	if err := yaml.Unmarshal([]byte(a.yamlDoc), &a.localDef); err != nil {
		return err
	}
	// Set the local definition on the apply result
	a.res.LocalDef = &a.localDef

	if err := a.localDef.Validate(); err != nil {
		return err
	}

	// Determine if the topic exists
	if err := a.tryFetchRemoteTopic(); err != nil {
		return err
	}

	// Perform further validations with metadata
	log.Debug("Validating topic definition using cluster metadata")
	if err := a.localDef.ValidateWithMetadata(a.metadata); err != nil {
		return err
	}

	if !a.createOp {
		// Build topic update operations and determine if updates are required
		if err := a.buildOperations(); err != nil {
			return err
		}
	}

	// Update the apply result with the remote definition and human readable diff
	if err := a.updateApplyResult(); err != nil {
		return err
	}

	if a.pendingOpsExist() {
		// Display diff and pending operations
		if !log.Quiet {
			a.displayPendingOps()
		}

		// Execute operations
		if err := a.executeOperations(); err != nil {
			return err
		}

		// Check for replica migrations
		// TODO: check for partitions ops too (?)
		// if a.assignmentsOp {
		log.Info("Fetching ongoing replica migrations for topic %q", a.localDef.Metadata.Name)

		partitions := make([]int32, a.localDef.Spec.Partitions)
		for i := range partitions {
			partitions[i] = int32(i)
		}
		reassignments, err := req.RequestListPartitionReassignments(
			a.cl,
			a.localDef.Metadata.Name,
			partitions,
		)
		if err != nil {
			return err
		}
		for _, r := range reassignments {
			fmt.Printf("Partition %s: +%d -%d\n", fmt.Sprint(r.Partition), len(r.AddingReplicas), len(r.RemovingReplicas))
		}
		// }

		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Completed apply for topic %q", a.localDef.Metadata.Name)
	} else {
		log.Info("No changes to apply for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Fetch metadata and config for a topic if it exists
func (a *applier) tryFetchRemoteTopic() error {
	log.Info("Checking if topic %q exists...", a.localDef.Metadata.Name)

	// Fetch and store metadata
	metadataResp, err := req.RequestMetadata(a.cl, []string{a.localDef.Metadata.Name}, false)
	if err != nil {
		return err
	}
	a.metadata = metadataResp

	// Check if the topic exists
	topicMetadata := a.metadata.Topics[0]
	a.createOp = topicMetadata.ErrorCode == kerr.UnknownTopicOrPartition.Code
	if a.createOp {
		log.Debug("Topic %q does not exist", a.localDef.Metadata.Name)
		return nil
	}

	// Fetch and store remote topic configs
	log.Info("Fetching configs for remote topic %q...", a.localDef.Metadata.Name)
	topicConfigsResp, err := req.RequestDescribeTopicConfigs(a.cl, []string{a.localDef.Metadata.Name})
	if err != nil {
		return err
	}
	a.remoteTopicConfigs = topicConfigsResp[0]

	// Build remote topic definition
	var remoteDef = def.NewTopicDefinition(
		topicMetadata,
		a.remoteTopicConfigs,
		false,
	)
	a.remoteDef = remoteDef

	return nil
}

// Build topic update operations
func (a *applier) buildOperations() error {
	a.buildConfigOps()

	if err := a.buildPartitionsOperation(); err != nil {
		return err
	}

	a.buildAssignmentsOperation()

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

	if a.assignmentsOp {
		return true
	}

	return false
}

// Update the apply result with the remote definition and human readable diff
func (a *applier) updateApplyResult() error {
	// Copy the remote definition
	var remoteCopy *def.TopicDefinition
	if !a.createOp {
		c := a.remoteDef.Copy()
		remoteCopy = &c
	}

	// Modify the remote definition to remove unnecessary properties
	if remoteCopy != nil {
		// Remove assignments if not specified in local
		if !a.localDef.Spec.HasAssignments() {
			remoteCopy.Spec.Assignments = nil
		}

		// The only configs we want to see are those specified in local and those in configOps
		for k := range remoteCopy.Spec.Configs {
			_, existsInLocal := a.localDef.Spec.Configs[k]
			existsInOps := a.configOps.Contains(k)

			if !existsInLocal && !existsInOps {
				delete(remoteCopy.Spec.Configs, k)
			}
		}
	}

	// Compute diff
	diff, err := def.DiffTopicDefinitions(remoteCopy, &a.localDef)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %v", err)
	}

	// Check the diff against the pending operations
	if diffExists := (len(diff) > 0); diffExists != a.pendingOpsExist() {
		return fmt.Errorf("existence of diff was %v, but expected %v", diffExists, a.pendingOpsExist())
	}

	// Update the apply result
	a.res.RemoteDef = remoteCopy
	a.res.Diff = diff

	return nil
}

// Display pending operations
func (a *applier) displayPendingOps() {
	if a.createOp {
		log.Info("Topic %q does not exist and will be created", a.localDef.Metadata.Name)
	}

	log.Info("Topic %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
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

	if a.assignmentsOp {
		if err := a.updateAssignments(); err != nil {
			return err
		}
	}

	return nil
}

// Execute a request to create a topic
func (a *applier) createTopic() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Creating topic %q...", a.localDef.Metadata.Name)

	if err := req.RequestCreateTopic(a.cl, a.localDef, a.flags.DryRun); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Created topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Build alter configs operations
func (a *applier) buildConfigOps() {
	log.Debug("Comparing local definition configs with remote topic %q", a.localDef.Metadata.Name)

	for k, v := range a.localDef.Spec.Configs {
		if cv, ok := a.remoteDef.Spec.Configs[k]; ok {
			// Config exists
			if *v != *cv {
				// Config value has changed
				log.Debug("Value of config key %q has changed from %q to %q and will be updated", k, *cv, *v)
				a.configOps = append(a.configOps, req.ConfigOperation{
					Name:         k,
					Value:        v,
					CurrentValue: cv,
					Op:           req.SetConfigOperation,
				})
			}
		} else {
			// Config does not exist
			log.Debug("Config key %q is missing from remote config and will be added", k)
			a.configOps = append(a.configOps, req.ConfigOperation{
				Name:  k,
				Value: v,
				Op:    req.SetConfigOperation,
			})
		}
	}

	// Mark missing configs for deletion
	if a.flags.DeleteMissingConfigs || a.flags.NonIncremental {
		for _, config := range a.remoteTopicConfigs.Configs {
			// Ignore static and default config keys that cannot be deleted
			if config.Source == kmsg.ConfigSourceStaticBrokerConfig || config.Source == kmsg.ConfigSourceDefaultConfig {
				continue
			}
			if _, ok := a.localDef.Spec.Configs[config.Name]; !ok {
				log.Debug("Config key %q is missing from local definition and will be deleted", config.Name)
				a.configOps = append(a.configOps, req.ConfigOperation{
					Name:         config.Name,
					CurrentValue: config.Value,
					Op:           req.DeleteConfigOperation,
				})
			}
		}
	}

	// TODO: remove
	// Sort the config operations
	a.configOps.Sort()
}

// TODO: remove
// Display alter configs operations
func (a *applier) displayConfigOps() {
	log.Info("The following changes will be applied to topic %q configs:", a.localDef.Metadata.Name)
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
			return errors.New("cannot apply delete config operations because flag --delete-missing-configs is not set")
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

	if err := req.RequestAlterConfigs(
		a.cl,
		a.localDef.Metadata.Name,
		a.configOps,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Execute a request to perform an incremental alter configs
func (a *applier) incrementalAlterConfigs() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altering configs...")

	if err := req.RequestIncrementalAlterConfigs(
		a.cl,
		a.localDef.Metadata.Name,
		a.configOps,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Build partitions operation
func (a *applier) buildPartitionsOperation() error {
	a.partitionsOp = a.localDef.Spec.Partitions > a.remoteDef.Spec.Partitions
	if a.partitionsOp {
		log.Debug(
			"The number of partitions has changed and will be increased from %d to %d",
			a.remoteDef.Spec.Partitions,
			a.localDef.Spec.Partitions,
		)
	}

	if a.localDef.Spec.Partitions < a.remoteDef.Spec.Partitions {
		return fmt.Errorf("decreasing the number of partitions is not supported")
	}

	return nil
}

// Execute a request to create partitions
func (a *applier) updatePartitions() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Creating partitions...")

	if err := req.RequestCreatePartitions(
		a.cl,
		a.localDef.Metadata.Name,
		a.localDef.Spec.Partitions,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Created partitions for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Build partitions operation
func (a *applier) buildAssignmentsOperation() {
	a.assignmentsOp = a.localDef.Spec.HasAssignments() &&
		!cmp.Equal(a.remoteDef.Spec.Assignments, a.localDef.Spec.Assignments)

	if a.assignmentsOp {
		log.Debug("Partition assignments have changed and will be updated")
	}
}

// Execute a request to alter assignments
func (a *applier) updateAssignments() error {
	if a.flags.DryRun {
		log.Info("Skipped altering partition assignments (dry-run not available)")
	} else {
		// TODO: Need to check for ongoing assignment ops on this topic(?)

		log.Info("Altering partition assignments...")

		if err := req.RequestAlterPartitionAssignments(
			a.cl,
			a.localDef.Metadata.Name,
			a.localDef.Spec.Assignments,
		); err != nil {
			return err
		} else {
			log.Info("Altered partition assignments for topic %q", a.localDef.Metadata.Name)
		}
	}

	return nil
}
