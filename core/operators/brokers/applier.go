package brokers

import (
	"errors"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/helpers/jsondiff"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/service"
)

// Flags to configure an applier
type ApplierFlags struct {
	DeleteMissingConfigs bool
	DryRun               bool
	NonIncremental       bool
}

// Applier operations
type applierOps struct {
	config service.ConfigOperations
}

// Determine if there are any pending operations
func (a applierOps) pending() bool {
	return len(a.config) > 0
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	cl      *client.Client
	yamlDoc string
	flags   ApplierFlags

	// internal
	localDef      def.BrokersDefinition
	remoteDef     def.BrokersDefinition
	remoteConfigs def.Configs
	ops           applierOps

	// TODO: refactor this to Execute() scope only?
	res res.ApplyResult
}

// Create a new applier
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

// Execute the applier
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

	// Fetch the remote definition
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

// Fetch remote brokers definition
func (a *applier) fetchRemote() error {
	log.Info("Fetching brokers configuration...")
	var err error
	a.remoteConfigs, err = service.DescribeAllBrokerConfigs(a.cl)
	if err != nil {
		return err
	}

	a.remoteDef = def.NewBrokersDefinition("brokers", a.remoteConfigs.ToMap())

	return nil
}

// Build brokers operations
func (a *applier) buildOps() error {
	a.buildConfigOps()
	return nil
}

// Update the apply result with the remote definition and human readable diff
func (a *applier) updateApplyResult() error {
	// Copy the remote definition
	remoteCopy := a.remoteDef.Copy()

	// Modify the remote definition to remove optional properties not specified in local
	// Further, add properties that are local only and have no remote state

	// The only configs we want to see are those specified in local and those in configOps
	// configOps could contain key deletions that should be shown in the diff
	for k := range remoteCopy.Spec.Configs {
		_, existsInLocal := a.localDef.Spec.Configs[k]
		existsInOps := a.ops.config.Contains(k)

		if !existsInLocal && !existsInOps {
			delete(remoteCopy.Spec.Configs, k)
		}
	}

	// Set metadata name to match local and remove it from the diff
	remoteCopy.Metadata.Name = a.localDef.Metadata.Name

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
func (a *applier) buildConfigOps() {
	log.Debug("Comparing local and remote definition configs for brokers %q", a.localDef.Metadata.Name)

	a.ops.config = service.NewConfigOps(
		a.localDef.Spec.Configs,
		a.remoteDef.Spec.Configs,
		a.remoteConfigs,
		a.flags.DeleteMissingConfigs,
		a.flags.NonIncremental,
	)
}

// Update brokers configs
func (a *applier) updateConfigs() error {
	if a.flags.NonIncremental {
		if a.ops.config.ContainsOp(service.DeleteConfigOperation) && !a.flags.DeleteMissingConfigs {
			return errors.New("cannot apply delete config operations because flag --delete-missing-configs is not set")
		}

		if err := a.alterConfigs(); err != nil {
			return err
		}
	} else {
		log.Debug("Checking if incremental alter configs is supported by the target cluster...")
		supported, err := service.IncrementalAlterConfigsIsSupported(a.cl)
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

	if err := service.AlterAllBrokerConfigs(
		a.cl,
		a.ops.config,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for brokers %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Execute a request to perform an incremental alter configs
func (a *applier) incrementalAlterConfigs() error {
	log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altering configs...")

	if err := service.IncrementalAlterAllBrokerConfigs(
		a.cl,
		a.ops.config,
		a.flags.DryRun,
	); err != nil {
		return err
	} else {
		log.InfoMaybeWithKey("dry-run", a.flags.DryRun, "Altered configs for brokers %q", a.localDef.Metadata.Name)
	}

	return nil
}
