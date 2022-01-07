// Package brokers implements operators for brokers definition operations.
package brokers

import (
	"context"
	"errors"
	"fmt"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/helpers/jsondiff"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
)

// ApplierOptions represents options to configure an applier.
type ApplierOptions struct {
	DefinitionFormat  opt.DefinitionFormat
	PropertyOverrides []string
	DryRun            bool
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
	config kafka.ConfigOperations
}

func (a applierOps) pending() bool {
	return len(a.config) > 0
}

type applier struct {
	// Constructor fields.
	srv    *kafka.Service
	defDoc string
	opts   ApplierOptions

	// Internal fields.
	localDef      def.BrokersDefinition
	remoteDef     def.BrokersDefinition
	remoteConfigs def.Configs
	ops           applierOps

	// Result fields.
	res res.ApplyResult
}

// Execute executes the applier.
func (a *applier) Execute(ctx context.Context) *res.ApplyResult {
	if err := a.apply(ctx); err != nil {
		a.res.Err = err.Error()
		log.Error(err)
	} else if a.ops.pending() && !a.opts.DryRun {
		a.res.Applied = true
	}

	return &a.res
}

// apply performs the apply operation sequence.
func (a *applier) apply(ctx context.Context) error {
	if err := a.createLocal(); err != nil {
		return err
	}

	log.Debugf("Validating brokers definition")
	if err := a.localDef.Validate(); err != nil {
		return err
	}

	if err := a.fetchRemote(ctx); err != nil {
		return err
	}

	if err := a.buildOps(ctx); err != nil {
		return err
	}

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

		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Completed apply for brokers definition %q", a.localDef.Metadata.Name)
	} else {
		log.Infof("No changes to apply for brokers definition %q", a.localDef.Metadata.Name)
	}

	return nil
}

// createLocal creates the local definition.
func (a *applier) createLocal() error {
	var err error
	a.localDef, err = def.LoadBrokersDefinition(a.defDoc, a.opts.DefinitionFormat)
	if err != nil {
		return err
	}

	a.res.LocalDef = &a.localDef

	return nil
}

// fetchRemote fetches the remote definition and necessary metadata.
func (a *applier) fetchRemote(ctx context.Context) error {
	log.Infof("Fetching remote cluster-wide broker configuration...")
	var err error
	a.remoteConfigs, err = a.srv.DescribeAllBrokerConfigs(ctx)
	if err != nil {
		return err
	}

	a.remoteDef = def.NewBrokersDefinition(
		a.localDef.Metadata,
		a.remoteConfigs.ToMap(),
	)

	return nil
}

// buildOps builds topic operations.
func (a *applier) buildOps(ctx context.Context) error {
	if err := a.buildConfigOps(ctx); err != nil {
		return err
	}
	return nil
}

// updateApplyResult updates the apply result with the remote definition and human readable diff.
func (a *applier) updateApplyResult() error {
	remoteCopy := a.remoteDef.Copy()

	// Modify the remote definition to remove optional properties not specified in local.
	// Further, set properties that are local only and have no remote state.

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

	diff, err := jsondiff.Diff(&remoteCopy, &a.localDef)
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
	log.Infof("brokers definition %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// executeOps executes update operations.
func (a *applier) executeOps(ctx context.Context) error {
	if len(a.ops.config) > 0 {
		if err := a.updateConfigs(ctx); err != nil {
			return err
		}
	}

	return nil
}

// buildConfigOps builds alter configs operations.
func (a *applier) buildConfigOps(ctx context.Context) error {
	log.Debugf("Comparing local and remote configs for brokers definition %q", a.localDef.Metadata.Name)

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

// updateConfigs updates brokers configs.
func (a *applier) updateConfigs(ctx context.Context) error {
	if a.ops.config.ContainsOp(kafka.DeleteConfigOperation) && !a.localDef.Spec.DeleteUndefinedConfigs {
		// This case should only occur when using non-incremental alter configs
		return errors.New("cannot apply configs because deletion of undefined configs is not enabled")
	}

	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altering configs...")
	if err := a.srv.AlterAllBrokerConfigs(
		ctx,
		a.ops.config,
		a.opts.DryRun,
	); err != nil {
		return err
	}
	log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Altered configs for brokers definition %q", a.localDef.Metadata.Name)

	return nil
}
