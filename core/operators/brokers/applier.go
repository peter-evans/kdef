package brokers

import (
	"errors"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/helpers/jsondiff"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
)

// Flags to configure an applier
type ApplierFlags struct {
	DryRun bool
}

// Create a new applier
func NewApplier(
	cl *client.Client,
	yamlDoc string,
	flags ApplierFlags,
) *applier {
	return &applier{
		srv:     kafka.NewService(cl),
		yamlDoc: yamlDoc,
		flags:   flags,
	}
}

// Applier operations
type applierOps struct {
	config kafka.ConfigOperations
}

// Determine if there are any pending operations
func (a applierOps) pending() bool {
	return len(a.config) > 0
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	srv     *kafka.Service
	yamlDoc string
	flags   ApplierFlags

	// internal
	localDef      def.BrokersDefinition
	remoteDef     def.BrokersDefinition
	remoteConfigs def.Configs
	ops           applierOps

	// result
	res res.ApplyResult
}

// Execute the applier
func (a *applier) Execute() *res.ApplyResult {
	if err := a.apply(); err != nil {
		a.res.Err = err.Error()
		log.Error(err)
	} else {
		// Consider the definition applied if there were ops and this is not a dry run
		if a.ops.pending() && !a.flags.DryRun {
			a.res.Applied = true
		}
	}

	return &a.res
}

// Perform the apply operation sequence
func (a *applier) apply() error {
	log.Debug("Validating brokers definition")
	if err := yaml.Unmarshal([]byte(a.yamlDoc), &a.localDef); err != nil {
		return err
	}
	// Set the local definition on the apply result
	a.res.LocalDef = &a.localDef

	if err := a.localDef.Validate(); err != nil {
		return err
	}

	// Fetch the remote definition and necessary metadata
	if err := a.fetchRemote(); err != nil {
		return err
	}

	// Build brokers operations
	if err := a.buildOps(); err != nil {
		return err
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

		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Completed apply for brokers %q", a.localDef.Metadata.Name)
	} else {
		log.Info("No changes to apply for brokers %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Fetch the remote definition and necessary metadata
func (a *applier) fetchRemote() error {
	log.Info("Fetching brokers configuration...")
	var err error
	a.remoteConfigs, err = a.srv.DescribeAllBrokerConfigs()
	if err != nil {
		return err
	}

	a.remoteDef = def.NewBrokersDefinition(a.remoteConfigs.ToMap())

	return nil
}

// Build brokers operations
func (a *applier) buildOps() error {
	if err := a.buildConfigOps(); err != nil {
		return err
	}
	return nil
}

// Update the apply result with the remote definition and human readable diff
func (a *applier) updateApplyResult() error {
	// Copy the remote definition
	remoteCopy := a.remoteDef.Copy()

	// Modify the remote definition to remove optional properties not specified in local
	// Further, set properties that are local only and have no remote state

	// The only configs we want to see are those specified in local and those in configOps
	// configOps could contain key deletions that should be shown in the diff
	for k := range remoteCopy.Spec.Configs {
		_, existsInLocal := a.localDef.Spec.Configs[k]
		existsInOps := a.ops.config.Contains(k)

		if !existsInLocal && !existsInOps {
			delete(remoteCopy.Spec.Configs, k)
		}
	}

	// Set properties that are local only and have no remote state
	remoteCopy.Metadata.Name = a.localDef.Metadata.Name
	remoteCopy.Spec.DeleteMissingConfigs = a.localDef.Spec.DeleteMissingConfigs

	// Compute diff
	diff, err := jsondiff.Diff(&remoteCopy, &a.localDef)
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
	log.Info("Brokers %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// Execute brokers update operations
func (a *applier) executeOps() error {
	if len(a.ops.config) > 0 {
		if err := a.updateConfigs(); err != nil {
			return err
		}
	}

	return nil
}

// Build alter configs operations
func (a *applier) buildConfigOps() error {
	log.Debug("Comparing local and remote definition configs for brokers %q", a.localDef.Metadata.Name)

	var err error
	a.ops.config, err = a.srv.NewConfigOps(
		a.localDef.Spec.Configs,
		a.remoteDef.Spec.Configs,
		a.remoteConfigs,
		a.localDef.Spec.DeleteMissingConfigs,
	)
	if err != nil {
		return err
	}

	return nil
}

// Update brokers configs
func (a *applier) updateConfigs() error {
	if a.ops.config.ContainsOp(kafka.DeleteConfigOperation) && !a.localDef.Spec.DeleteMissingConfigs {
		// This case should only occur when using non-incremental alter configs
		return errors.New("cannot apply configs because deletion of missing configs is not enabled")
	}

	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altering configs...")
	if err := a.srv.AlterAllBrokerConfigs(
		a.ops.config,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for brokers %q", a.localDef.Metadata.Name)
	}

	return nil
}
