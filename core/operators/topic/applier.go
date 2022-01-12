// Package topic implements operators for topic definition operations.
package topic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/helpers/assignments"
	"github.com/peter-evans/kdef/core/helpers/jsondiff"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/util/i32"
)

// ApplierOptions represents options to configure an applier.
type ApplierOptions struct {
	DefinitionFormat  opt.DefinitionFormat
	PropertyOverrides []string
	DryRun            bool
	ReassAwaitTimeout int
}

// NewApplier creates a new applier.
func NewApplier(
	cl *client.Client,
	defDoc string,
	opts ApplierOptions,
) *applier { //revive:disable-line:unexported-return
	return &applier{
		srv:    kafka.NewService(cl),
		defDoc: defDoc,
		opts:   opts,
	}
}

type applierOps struct {
	create            bool
	createAssignments def.PartitionAssignments
	config            kafka.ConfigOperations
	partitions        def.PartitionAssignments
	assignments       def.PartitionAssignments
	leaderElection    struct {
		leaders    []int32
		partitions []int32
	}
}

func (a applierOps) pending() bool {
	return a.create ||
		len(a.config) > 0 ||
		len(a.partitions) > 0 ||
		len(a.assignments) > 0 ||
		len(a.leaderElection.partitions) > 0
}

type applier struct {
	// Constructor fields.
	srv    *kafka.Service
	defDoc string
	opts   ApplierOptions

	// Internal fields.
	localDef             def.TopicDefinition
	remoteDef            *def.TopicDefinition
	remoteConfigs        def.Configs
	remotePartitionISR   def.PartitionAssignments
	brokers              meta.Brokers
	clusterReplicaCounts map[int32]int
	ops                  applierOps

	// Result fields.
	res           res.ApplyResult
	reassignments meta.PartitionReassignments
}

// Execute executes the applier.
func (a *applier) Execute(ctx context.Context) *res.ApplyResult {
	if err := a.apply(ctx); err != nil {
		a.res.Err = err.Error()
		log.Error(err)
	} else if a.ops.pending() && !a.opts.DryRun {
		a.res.Applied = true
	}

	a.res.Data = res.TopicApplyResultData{
		PartitionReassignments: a.reassignments,
	}

	return &a.res
}

// apply performs the apply operation sequence.
func (a *applier) apply(ctx context.Context) error {
	if err := a.createLocal(); err != nil {
		return err
	}

	log.Debugf("Validating topic definition")
	if err := a.localDef.Validate(); err != nil {
		return err
	}

	if err := a.tryFetchRemote(ctx); err != nil {
		return err
	}

	log.Debugf("Validating topic definition using cluster metadata")
	if err := a.localDef.ValidateWithMetadata(a.brokers); err != nil {
		return err
	}

	if err := a.buildOps(ctx); err != nil {
		return err
	}

	a.updateLocalState()

	if err := a.updateApplyResult(); err != nil {
		return err
	}

	if a.ops.pending() {
		if !log.Quiet {
			a.displayPendingOps()
		}

		if err := a.executeOps(ctx); err != nil {
			return err
		}

		// Check for in-progress partition reassignments as a result of operations involving assignments.
		if len(a.ops.assignments) > 0 {
			if err := a.fetchPartitionReassignments(ctx, false); err != nil {
				return err
			}
			if len(a.reassignments) > 0 {
				if a.opts.ReassAwaitTimeout > 0 {
					if err := a.awaitReassignments(ctx, a.opts.ReassAwaitTimeout); err != nil {
						return err
					}
				} else if !log.Quiet {
					a.displayPartitionReassignments()
				}
			}
		}

		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Completed apply for topic definition %q", a.localDef.Metadata.Name)
	} else {
		log.Infof("No changes to apply for topic definition %q", a.localDef.Metadata.Name)
	}

	return nil
}

// createLocal creates the local definition.
func (a *applier) createLocal() error {
	var err error
	a.localDef, err = def.LoadTopicDefinition(a.defDoc, a.opts.DefinitionFormat, a.opts.PropertyOverrides)
	if err != nil {
		return err
	}

	a.res.LocalDef = &a.localDef

	return nil
}

// tryFetchRemote fetches the remote definition and necessary metadata.
func (a *applier) tryFetchRemote(ctx context.Context) error {
	log.Infof("Fetching remote topic...")
	var err error
	a.remoteDef, a.remoteConfigs, a.remotePartitionISR, a.brokers, err = a.srv.TryRequestTopic(ctx, a.localDef.Metadata)
	if err != nil {
		return err
	}

	if a.localDef.Spec.HasManagedAssignments() && a.localDef.Spec.ManagedAssignments.Selection == def.SelectionTopicClusterUse {
		// Describe metadata for all topics in the cluster.
		metadata, err := a.srv.DescribeMetadata(ctx, nil, true)
		if err != nil {
			return err
		}
		a.clusterReplicaCounts = make(map[int32]int)
		for _, t := range metadata.Topics {
			for _, replicas := range t.PartitionAssignments {
				for _, brokerID := range replicas {
					a.clusterReplicaCounts[brokerID]++
				}
			}
		}
	}

	a.ops.create = (a.remoteDef == nil)
	if a.ops.create {
		log.Debugf("Topic %q does not exist", a.localDef.Metadata.Name)
	}

	return nil
}

// buildOps builds topic operations.
func (a *applier) buildOps(ctx context.Context) error {
	if a.ops.create {
		a.buildCreateOp()
	} else {
		if err := a.buildConfigOps(ctx); err != nil {
			return err
		}
		if err := a.buildPartitionsOp(); err != nil {
			return err
		}
		a.buildAssignmentsOp()
		a.buildLeaderElectionOp()
	}
	return nil
}

// updateLocalState updates the state property group of the local definition.
func (a *applier) updateLocalState() {
	// The state property group of the local definition is updated to show the underlying state changes.
	if a.localDef.Spec.HasManagedAssignments() {
		switch {
		case a.ops.create:
			a.localDef.State = &def.TopicStateDefinition{
				Assignments: a.ops.createAssignments,
			}
		case len(a.ops.assignments) > 0:
			// Includes partition ops
			a.localDef.State = &def.TopicStateDefinition{
				Assignments: a.ops.assignments,
			}
		case len(a.ops.partitions) > 0:
			a.localDef.State = &def.TopicStateDefinition{
				Assignments: a.remoteDef.Spec.Assignments,
			}
			a.localDef.State.Assignments = append(a.localDef.State.Assignments, a.ops.partitions...)
		}
	}

	if len(a.ops.leaderElection.partitions) > 0 {
		if a.localDef.State != nil {
			a.localDef.State.Leaders = a.ops.leaderElection.leaders
		} else {
			a.localDef.State = &def.TopicStateDefinition{
				Leaders: a.ops.leaderElection.leaders,
			}
		}
	}
}

// updateApplyResult updates the apply result with the remote definition and human readable diff.
func (a *applier) updateApplyResult() error {
	var remoteCopy *def.TopicDefinition
	if !a.ops.create {
		c := a.remoteDef.Copy()
		remoteCopy = &c
	}

	// Modify the remote definition to remove optional properties not specified in local.
	// Further, set properties that are local only and have no remote state.
	if remoteCopy != nil {
		if !a.localDef.Spec.HasAssignments() {
			remoteCopy.Spec.Assignments = nil
		}

		if !a.localDef.Spec.HasManagedAssignments() {
			remoteCopy.Spec.ManagedAssignments = nil
		} else {
			if !a.localDef.Spec.ManagedAssignments.HasRackConstraints() {
				remoteCopy.Spec.ManagedAssignments.RackConstraints = nil
			}
			remoteCopy.Spec.ManagedAssignments.Balance = a.localDef.Spec.ManagedAssignments.Balance
			remoteCopy.Spec.ManagedAssignments.Selection = a.localDef.Spec.ManagedAssignments.Selection
		}

		// The only configs we want to see are those specified in local and those in configOps.
		// configOps could contain key deletions that should be shown in the diff.
		for k := range remoteCopy.Spec.Configs {
			_, existsInLocal := a.localDef.Spec.Configs[k]
			existsInOps := a.ops.config.Contains(k)

			if !existsInLocal && !existsInOps {
				delete(remoteCopy.Spec.Configs, k)
			}
		}

		remoteCopy.Spec.DeleteUndefinedConfigs = a.localDef.Spec.DeleteUndefinedConfigs
		remoteCopy.Spec.MaintainLeaders = a.localDef.Spec.MaintainLeaders

		if a.localDef.State == nil {
			remoteCopy.State = nil
		} else {
			if a.localDef.State.Assignments == nil {
				remoteCopy.State.Assignments = nil
			}
			if a.localDef.State.Leaders == nil {
				remoteCopy.State.Leaders = nil
			}
		}
	}

	diff, err := jsondiff.Diff(remoteCopy, &a.localDef)
	if err != nil {
		return fmt.Errorf("failed to compute diff: %v", err)
	}

	if diffExists := (len(diff) > 0); diffExists != a.ops.pending() {
		return fmt.Errorf("existence of diff was %v, but expected %v", diffExists, a.ops.pending())
	}

	a.res.RemoteDef = remoteCopy
	a.res.Diff = diff

	return nil
}

// displayPendingOps displays pending operations.
func (a *applier) displayPendingOps() {
	if a.ops.create {
		log.Infof("Topic %q does not exist and will be created", a.localDef.Metadata.Name)
	}

	log.Infof("topic definition %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// executeOps executes update operations.
func (a *applier) executeOps(ctx context.Context) error {
	if a.ops.create {
		if err := a.createTopic(ctx); err != nil {
			return err
		}
	}

	if len(a.ops.config) > 0 {
		if err := a.updateConfigs(ctx); err != nil {
			return err
		}
	}

	if len(a.ops.partitions) > 0 {
		if err := a.updatePartitions(ctx); err != nil {
			return err
		}
	}

	if len(a.ops.assignments) > 0 {
		if err := a.updateAssignments(ctx); err != nil {
			return err
		}
	}

	if len(a.ops.leaderElection.partitions) > 0 {
		if err := a.electPartitionLeaders(ctx); err != nil {
			return err
		}
	}

	return nil
}

// buildCreateOp builds a create operation.
func (a *applier) buildCreateOp() {
	switch {
	case a.localDef.Spec.HasAssignments():
		a.ops.createAssignments = a.localDef.Spec.Assignments
	case a.localDef.Spec.HasManagedAssignments() && a.localDef.Spec.ManagedAssignments.HasRackConstraints():
		// Make an empty set of assignments.
		newAssignments := make(def.PartitionAssignments, len(a.localDef.Spec.ManagedAssignments.RackConstraints))
		for i := range newAssignments {
			newAssignments[i] = make([]int32, len(a.localDef.Spec.ManagedAssignments.RackConstraints[0]))
		}
		// Populate local assignments from defined rack constraints.
		a.ops.createAssignments = assignments.SyncRackConstraints(
			newAssignments,
			a.localDef.Spec.ManagedAssignments.RackConstraints,
			a.brokers.BrokersByRack(),
			a.clusterReplicaCounts,
		)
	default:
		a.ops.createAssignments = assignments.AddPartitions(
			[][]int32{},
			a.localDef.Spec.Partitions,
			a.localDef.Spec.ReplicationFactor,
			a.clusterReplicaCounts,
			a.brokers.IDs(),
		)
	}
}

// createTopic executes a request to create a topic.
func (a *applier) createTopic(ctx context.Context) error {
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Creating topic...")

	if err := a.srv.CreateTopic(
		ctx,
		a.localDef,
		a.ops.createAssignments,
		a.opts.DryRun,
	); err != nil {
		return err
	}
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Created topic %q", a.localDef.Metadata.Name)

	return nil
}

// buildConfigOps builds alter configs operations.
func (a *applier) buildConfigOps(ctx context.Context) error {
	log.Debugf("Comparing local and remote configs for topic %q", a.localDef.Metadata.Name)

	var err error
	a.ops.config, err = a.srv.NewConfigOps(
		ctx,
		a.localDef.Spec.Configs,
		a.remoteDef.Spec.Configs,
		a.remoteConfigs,
		a.localDef.Spec.DeleteUndefinedConfigs,
	)
	if err != nil {
		return err
	}

	return nil
}

// updateConfigs updates topic configs.
func (a *applier) updateConfigs(ctx context.Context) error {
	if a.ops.config.ContainsOp(kafka.DeleteConfigOperation) && !a.localDef.Spec.DeleteUndefinedConfigs {
		// This case should only occur when using non-incremental alter configs.
		return errors.New("cannot apply configs because deletion of undefined configs is not enabled")
	}

	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altering configs...")
	if err := a.srv.AlterTopicConfigs(
		ctx,
		a.localDef.Metadata.Name,
		a.ops.config,
		a.opts.DryRun,
	); err != nil {
		return err
	}
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altered configs for topic %q", a.localDef.Metadata.Name)

	return nil
}

// buildPartitionsOp builds a partitions operation.
func (a *applier) buildPartitionsOp() error {
	if a.localDef.Spec.Partitions < a.remoteDef.Spec.Partitions {
		return fmt.Errorf("decreasing the number of partitions is not supported")
	}

	if a.localDef.Spec.Partitions > a.remoteDef.Spec.Partitions {
		log.Debugf(
			"The number of partitions has changed and will be increased from %d to %d",
			a.remoteDef.Spec.Partitions,
			a.localDef.Spec.Partitions,
		)
		// It's not necessary to cater specifically for rack constraints here because any miss-placements
		// will be reassigned and migrate to the correct broker very quickly in the cluster.
		targetRepFactor := len(a.remoteDef.Spec.Assignments[0])
		a.ops.partitions = assignments.AddPartitions(
			a.remoteDef.Spec.Assignments,
			a.localDef.Spec.Partitions,
			targetRepFactor,
			a.clusterReplicaCounts,
			a.brokers.IDs(),
		)
	}

	return nil
}

// updatePartitions executes a request to create partitions.
func (a *applier) updatePartitions(ctx context.Context) error {
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Creating partitions...")

	if err := a.srv.CreatePartitions(
		ctx,
		a.localDef.Metadata.Name,
		a.localDef.Spec.Partitions,
		a.ops.partitions,
		a.opts.DryRun,
	); err != nil {
		return err
	}
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Created partitions for topic %q", a.localDef.Metadata.Name)

	return nil
}

// buildAssignmentsOp builds an assignments operation.
func (a *applier) buildAssignmentsOp() {
	if a.localDef.Spec.HasAssignments() {
		if !cmp.Equal(a.remoteDef.Spec.Assignments, a.localDef.Spec.Assignments) {
			log.Debugf("Partition assignments have changed and will be updated")
			a.ops.assignments = a.localDef.Spec.Assignments
		}
	} else { // Managed assignments.
		if a.localDef.Spec.ManagedAssignments.HasRackConstraints() {
			var newAssignments def.PartitionAssignments
			newAssignments = assignments.Copy(a.remoteDef.Spec.Assignments)
			if len(a.ops.partitions) > 0 {
				newAssignments = append(newAssignments, a.ops.partitions...)
			}
			newAssignments = assignments.SyncRackConstraints(
				newAssignments,
				a.localDef.Spec.ManagedAssignments.RackConstraints,
				a.brokers.BrokersByRack(),
				a.clusterReplicaCounts,
			)
			if !cmp.Equal(a.remoteDef.Spec.Assignments, newAssignments) {
				log.Debugf("Partition assignments are out of sync with defined racks and will be updated")
				a.ops.assignments = newAssignments
			}
		} else if a.localDef.Spec.ReplicationFactor != a.remoteDef.Spec.ReplicationFactor {
			log.Debugf("Replication factor has changed and will be updated")
			var newAssignments def.PartitionAssignments
			newAssignments = assignments.Copy(a.remoteDef.Spec.Assignments)
			if len(a.ops.partitions) > 0 {
				newAssignments = append(newAssignments, a.ops.partitions...)
			}
			a.ops.assignments = assignments.AlterReplicationFactor(
				newAssignments,
				a.localDef.Spec.ReplicationFactor,
				a.clusterReplicaCounts,
				a.brokers.IDs(),
			)
		}

		if a.localDef.Spec.ManagedAssignments.Balance == def.BalanceAll {
			prebalancedAssignments := a.remoteDef.Spec.Assignments
			if len(a.ops.assignments) > 0 {
				prebalancedAssignments = a.ops.assignments
			}

			var rebalancedAssignments def.PartitionAssignments
			if a.localDef.Spec.ManagedAssignments.HasRackConstraints() {
				rebalancedAssignments = assignments.RebalanceWithRackConstraints(
					prebalancedAssignments,
					a.localDef.Spec.ManagedAssignments.RackConstraints,
					a.clusterReplicaCounts,
					a.brokers.BrokersByRack(),
				)
			} else {
				rebalancedAssignments = assignments.Rebalance(
					prebalancedAssignments,
					a.clusterReplicaCounts,
					a.brokers.IDs(),
				)
			}

			if !cmp.Equal(prebalancedAssignments, rebalancedAssignments) {
				log.Debugf("Partition assignments have been rebalanced and will be updated")
				a.ops.assignments = rebalancedAssignments
			}
		}
	}
}

// fetchPartitionReassignments executes a request to list partition reassignments.
func (a *applier) fetchPartitionReassignments(ctx context.Context, suppressLog bool) error {
	if !(suppressLog) {
		log.Debugf("Fetching in-progress partition reassignments for topic %q", a.localDef.Metadata.Name)
	}

	partitions := make([]int32, a.localDef.Spec.Partitions)
	for i := range partitions {
		partitions[i] = int32(i)
	}

	var err error
	a.reassignments, err = a.srv.ListPartitionReassignments(
		ctx,
		a.localDef.Metadata.Name,
		partitions,
	)
	if err != nil {
		return err
	}

	return nil
}

// displayPartitionReassignments displays in-progress partition reassignments.
func (a *applier) displayPartitionReassignments() {
	log.Infof("In-progress partition reassignments for topic %q:", a.localDef.Metadata.Name)
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

// updateAssignments executes a request to alter assignments.
func (a *applier) updateAssignments(ctx context.Context) error {
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altering partition assignments...")

	if a.opts.DryRun {
		// AlterPartitionAssignments has no 'ValidateOnly' for dry-run mode so we check
		// in-progress partition reassignments and error if found.
		if err := a.fetchPartitionReassignments(ctx, false); err != nil {
			return err
		}
		if len(a.reassignments) > 0 {
			// Kafka would return a very similiar error if we attempted to execute the reassignment.
			return fmt.Errorf("a partition reassignment is in progress for the topic %q", a.localDef.Metadata.Name)
		}
	} else if err := a.srv.AlterPartitionAssignments(
		ctx,
		a.localDef.Metadata.Name,
		a.ops.assignments,
	); err != nil {
		return err
	}

	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altered partition assignments for topic %q", a.localDef.Metadata.Name)

	return nil
}

// awaitReassignments awaits the completion of in-progress partition reassignments.
func (a *applier) awaitReassignments(ctx context.Context, timeoutSec int) error {
	log.Infof("Awaiting completion of partition reassignments (timeout: %d seconds)...", timeoutSec)
	timeout := time.After(time.Duration(timeoutSec) * time.Second)

	remaining := 0
	for {
		select {
		case <-timeout:
			log.Infof("Awaiting completion of partition reassignments timed out after %d seconds", timeoutSec)
			return nil
		default:
			if err := a.fetchPartitionReassignments(ctx, true); err != nil {
				return err
			}
			if len(a.reassignments) > 0 {
				if !log.Quiet && len(a.reassignments) != remaining {
					a.displayPartitionReassignments()
				}
				remaining = len(a.reassignments)
			} else {
				log.Infof("Partition reassignments completed")
				return nil
			}

			time.Sleep(5 * time.Second)
			continue
		}
	}
}

// buildLeaderElectionOp builds a leader election operation.
func (a *applier) buildLeaderElectionOp() {
	if a.localDef.Spec.MaintainLeaders {
		assignments := a.remoteDef.Spec.Assignments
		if len(a.ops.assignments) > 0 {
			assignments = a.ops.assignments
		}

		a.ops.leaderElection.leaders = make([]int32, len(assignments))
		for partition, replicas := range assignments {
			preferredLeader := replicas[0]
			a.ops.leaderElection.leaders[partition] = preferredLeader

			if partition >= len(a.remoteDef.Spec.Assignments) {
				// Additional partitions that don't yet exist at the remote.
				continue
			}

			// If the preferred leader is not in the middle of being reassigned,
			// AND the current leader is not the preferred leader, set for election.
			if a.remoteDef.Spec.Assignments[partition][0] == preferredLeader &&
				a.remoteDef.State.Leaders[partition] != preferredLeader {
				// Check that the preferred leader is an in-sync replica.
				if i32.Contains(preferredLeader, a.remotePartitionISR[partition]) {
					a.ops.leaderElection.partitions = append(a.ops.leaderElection.partitions, int32(partition))
				} else {
					log.Warnf(
						"Cannot elect preferred leader %q of partition %q because it is not an in-sync replica.",
						fmt.Sprint(preferredLeader),
						partition,
					)
				}
			}
		}
	}
}

// electPartitionLeaders executes a request to elect partition leaders.
func (a *applier) electPartitionLeaders(ctx context.Context) error {
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Electing partition leaders...")

	if !a.opts.DryRun {
		if err := a.srv.ElectLeaders(
			ctx,
			a.localDef.Metadata.Name,
			a.ops.leaderElection.partitions,
		); err != nil {
			return err
		}
	}

	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Elected partition leaders for topic %q", a.localDef.Metadata.Name)

	return nil
}
