package topics

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/req"
	"github.com/peter-evans/kdef/core/res"
	"github.com/peter-evans/kdef/core/topics/assignments"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// Flags to configure an applier
type ApplierFlags struct {
	DeleteMissingConfigs bool
	DryRun               bool
	NonIncremental       bool
}

// Applier operations
type applierOps struct {
	create      bool
	config      req.ConfigOperations
	partitions  def.PartitionAssignments
	assignments def.PartitionAssignments
}

// Determine if there are any pending operations
func (a applierOps) pending() bool {
	return a.create ||
		len(a.config) > 0 ||
		len(a.partitions) > 0 ||
		len(a.assignments) > 0
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	cl      *client.Client
	yamlDoc string
	flags   ApplierFlags

	// internal
	// TODO: can I refactor this out?
	remoteTopicConfigs kmsg.DescribeConfigsResponseResource
	brokers            []int32
	localDef           def.TopicDefinition
	remoteDef          def.TopicDefinition
	ops                applierOps

	// TODO: refactor this to Execute() scope only?
	res           res.ApplyResult
	reassignments res.PartitionReassignments
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

	a.res.Data = res.TopicApplyResultData{
		PartitionReassignments: a.reassignments,
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
	if err := a.localDef.ValidateWithMetadata(a.brokers); err != nil {
		return err
	}

	if !a.ops.create {
		// Build topic update operations and determine if updates are required
		if err := a.buildOps(); err != nil {
			return err
		}
	}

	// Update the apply result with the remote definition and human readable diff
	if err := a.updateApplyResult(); err != nil {
		return err
	}

	if a.ops.pending() {
		// Display diff and pending operations
		if !log.Quiet {
			a.displayPendingOps()
		}

		// Execute operations
		if err := a.executeOps(); err != nil {
			return err
		}

		// Check for in-progress partition reassignments as a result of operations
		if len(a.ops.assignments) > 0 {
			if err := a.fetchPartitionReassignments(); err != nil {
				return err
			}
			if len(a.reassignments) > 0 {
				if a.localDef.Spec.Reassignment.AwaitTimeoutSec > 0 {
					// Wait for reassignments to complete
					if err := a.awaitReassignments(a.localDef.Spec.Reassignment.AwaitTimeoutSec); err != nil {
						return err
					}
				} else if !log.Quiet {
					// Display in-progress partition reassignments
					a.displayPartitionReassignments()
				}
			}
		}

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

	// Store available brokers
	for _, broker := range metadataResp.Brokers {
		a.brokers = append(a.brokers, broker.NodeID)
	}

	// TODO: When I need to fetch racks create a metadata section in 'a'
	// Maybe define the struct for metadata in req

	// Check if the topic exists
	topicMetadata := metadataResp.Topics[0]
	a.ops.create = topicMetadata.ErrorCode == kerr.UnknownTopicOrPartition.Code
	if a.ops.create {
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
func (a *applier) buildOps() error {
	a.buildConfigOps()

	if err := a.buildPartitionsOp(); err != nil {
		return err
	}

	a.buildAssignmentsOp()

	return nil
}

// Update the apply result with the remote definition and human readable diff
func (a *applier) updateApplyResult() error {
	// Copy the remote definition
	var remoteCopy *def.TopicDefinition
	if !a.ops.create {
		c := a.remoteDef.Copy()
		remoteCopy = &c
	}

	// Modify the remote definition to remove optional properties not specified in local
	// Further, add properties that are local only and have no remote state
	if remoteCopy != nil {
		// Remove assignments if not specified in local
		if !a.localDef.Spec.HasAssignments() {
			remoteCopy.Spec.Assignments = nil
		}

		// The only configs we want to see are those specified in local and those in configOps
		// configOps could contain key deletions that should be shown in the diff
		for k := range remoteCopy.Spec.Configs {
			_, existsInLocal := a.localDef.Spec.Configs[k]
			existsInOps := a.ops.config.Contains(k)

			if !existsInLocal && !existsInOps {
				delete(remoteCopy.Spec.Configs, k)
			}
		}

		// Add properties that are local only and have no remote state
		remoteCopy.Spec.Reassignment.AwaitTimeoutSec = a.localDef.Spec.Reassignment.AwaitTimeoutSec
	}

	// Compute diff
	diff, err := def.DiffTopicDefinitions(remoteCopy, &a.localDef)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %v", err)
	}

	// Check the diff against the pending operations
	if diffExists := (len(diff) > 0); diffExists != a.ops.pending() {
		return fmt.Errorf("existence of diff was %v, but expected %v", diffExists, a.ops.pending())
	}

	// Update the apply result
	a.res.RemoteDef = remoteCopy
	a.res.Diff = diff

	return nil
}

// Display pending operations
func (a *applier) displayPendingOps() {
	if a.ops.create {
		log.Info("Topic %q does not exist and will be created", a.localDef.Metadata.Name)
	}

	log.Info("Topic %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// Execute topic update operations
func (a *applier) executeOps() error {
	if a.ops.create {
		if err := a.createTopic(); err != nil {
			return err
		}
	}

	if len(a.ops.config) > 0 {
		if err := a.updateConfigs(); err != nil {
			return err
		}
	}

	if len(a.ops.partitions) > 0 {
		if err := a.updatePartitions(); err != nil {
			return err
		}
	}

	if len(a.ops.assignments) > 0 {
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
				a.ops.config = append(a.ops.config, req.ConfigOperation{
					Name:         k,
					Value:        v,
					CurrentValue: cv,
					Op:           req.SetConfigOperation,
				})
			}
		} else {
			// Config does not exist
			log.Debug("Config key %q is missing from remote config and will be added", k)
			a.ops.config = append(a.ops.config, req.ConfigOperation{
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
				a.ops.config = append(a.ops.config, req.ConfigOperation{
					Name:         config.Name,
					CurrentValue: config.Value,
					Op:           req.DeleteConfigOperation,
				})
			}
		}
	}

	// TODO: remove
	// Sort the config operations
	a.ops.config.Sort()
}

// Update topic configs
func (a *applier) updateConfigs() error {
	if a.flags.NonIncremental {
		if a.ops.config.ContainsOp(req.DeleteConfigOperation) && !a.flags.DeleteMissingConfigs {
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
		a.ops.config,
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
		a.ops.config,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Build partitions operation
func (a *applier) buildPartitionsOp() error {
	if a.localDef.Spec.Partitions < a.remoteDef.Spec.Partitions {
		return fmt.Errorf("decreasing the number of partitions is not supported")
	}

	if a.localDef.Spec.Partitions > a.remoteDef.Spec.Partitions {
		log.Debug(
			"The number of partitions has changed and will be increased from %d to %d",
			a.remoteDef.Spec.Partitions,
			a.localDef.Spec.Partitions,
		)
		a.ops.partitions = assignments.AddPartitions(
			a.remoteDef.Spec.Assignments,
			a.localDef.Spec.Partitions,
			a.brokers,
		)
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
		a.ops.partitions,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Created partitions for topic %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Build assignments operation
func (a *applier) buildAssignmentsOp() {
	if a.localDef.Spec.HasAssignments() {
		if !cmp.Equal(a.remoteDef.Spec.Assignments, a.localDef.Spec.Assignments) {
			log.Debug("Partition assignments have changed and will be updated")
			a.ops.assignments = a.localDef.Spec.Assignments
		}
	} else {
		if a.localDef.Spec.ReplicationFactor != a.remoteDef.Spec.ReplicationFactor {
			log.Debug("Replication factor has changed and will be updated")
			if len(a.ops.partitions) > 0 {
				// Assignments will include new partitions that will be added
				newAssignments := assignments.Copy(a.remoteDef.Spec.Assignments)
				newAssignments = append(newAssignments, a.ops.partitions...)
				a.ops.assignments = assignments.AlterReplicationFactor(
					newAssignments,
					a.localDef.Spec.ReplicationFactor,
					a.brokers,
				)
			} else {
				a.ops.assignments = assignments.AlterReplicationFactor(
					a.remoteDef.Spec.Assignments,
					a.localDef.Spec.ReplicationFactor,
					a.brokers,
				)
			}
		}
	}
}

// Execute a request to list partition reassignments
func (a *applier) fetchPartitionReassignments() error {
	log.Debug("Fetching in-progress partition reassignments for topic %q", a.localDef.Metadata.Name)

	partitions := make([]int32, a.localDef.Spec.Partitions)
	for i := range partitions {
		partitions[i] = int32(i)
	}

	reassResp, err := req.RequestListPartitionReassignments(
		a.cl,
		a.localDef.Metadata.Name,
		partitions,
	)
	if err != nil {
		return err
	}

	a.reassignments = res.PartitionReassignments{}
	for _, r := range reassResp {
		a.reassignments = append(a.reassignments, res.PartitionReassignment{
			Partition:        r.Partition,
			Replicas:         r.Replicas,
			AddingReplicas:   r.AddingReplicas,
			RemovingReplicas: r.RemovingReplicas,
		})
	}

	return nil
}

// Display in-progress partition reassignments
func (a *applier) displayPartitionReassignments() {
	log.Info("In-progress partition reassignments for topic %q:", a.localDef.Metadata.Name)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Partition", "Replicas", "Adding Replicas", "Removing Replicas"})
	for _, r := range a.reassignments {
		t.AppendRow([]interface{}{
			fmt.Sprint(r.Partition),
			fmt.Sprint(r.Replicas),
			fmt.Sprint(r.AddingReplicas),
			fmt.Sprint(r.RemovingReplicas),
		})
	}
	t.SetStyle(table.StyleLight)
	t.Render()
}

// Execute a request to alter assignments
func (a *applier) updateAssignments() error {
	if a.flags.DryRun {
		// AlterPartitionAssignments has no 'ValidateOnly' for dry-run mode so we check
		// in-progress partition reassignments and error if found.
		if err := a.fetchPartitionReassignments(); err != nil {
			return err
		}
		if len(a.reassignments) > 0 {
			// Kafka would return a very similiar error if we attempted to execute the reassignment
			return fmt.Errorf("a partition reassignment is in progress for the topic %q", a.localDef.Metadata.Name)
		}

		log.Info("Skipped altering partition assignments (dry-run not available)")
	} else {
		log.Info("Altering partition assignments...")

		if err := req.RequestAlterPartitionAssignments(
			a.cl,
			a.localDef.Metadata.Name,
			a.ops.assignments,
		); err != nil {
			return err
		} else {
			log.Info("Altered partition assignments for topic %q", a.localDef.Metadata.Name)
		}
	}

	return nil
}

// Await the completion of in-progress partition reassignments
func (a *applier) awaitReassignments(timeoutSec int) error {
	log.Info("Awaiting completion of partition reassignments (timeout: %d seconds)...", timeoutSec)
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	remaining := 0
	for {
		select {
		case <-timeout:
			log.Info("Awaiting completion of partition reassignments timed out after %d seconds", timeoutSec)
			return nil
		default:
			if err := a.fetchPartitionReassignments(); err != nil {
				return err
			}
			if len(a.reassignments) > 0 {
				if !log.Quiet && len(a.reassignments) != remaining {
					a.displayPartitionReassignments()
				}
				remaining = len(a.reassignments)
			} else {
				log.Info("Partition reassignments completed")
				return nil
			}

			time.Sleep(5 * time.Second)
			continue
		}
	}
}
