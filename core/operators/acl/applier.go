// Package acl implements operators for acl definition operations.
package acl

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/helpers/acls"
	"github.com/peter-evans/kdef/core/helpers/jsondiff"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
)

// ApplierOptions represents options to configure an applier.
type ApplierOptions struct {
	DefinitionFormat opt.DefinitionFormat
	DryRun           bool
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
	addACLs    def.ACLEntryGroups
	deleteACLs def.ACLEntryGroups
}

func (a applierOps) pending() bool {
	return len(a.addACLs) > 0 ||
		len(a.deleteACLs) > 0
}

type applier struct {
	// Constructor fields.
	srv    *kafka.Service
	defDoc string
	opts   ApplierOptions

	// Internal fields.
	localDef   def.ACLDefinition
	remoteDef  def.ACLDefinition
	remoteACLs def.ACLEntryGroups
	ops        applierOps

	// Result fields.
	res res.ApplyResult
}

// Execute executes the applier.
func (a *applier) Execute() *res.ApplyResult {
	if err := a.apply(); err != nil {
		a.res.Err = err.Error()
		log.Error(err)
	} else if a.ops.pending() && !a.opts.DryRun {
		a.res.Applied = true
	}

	return &a.res
}

// apply performs the apply operation sequence.
func (a *applier) apply() error {
	if err := a.createLocal(); err != nil {
		return err
	}

	log.Debugf("Validating acl definition")
	if err := a.localDef.Validate(); err != nil {
		return err
	}

	if err := a.fetchRemote(); err != nil {
		return err
	}

	if err := a.buildOps(); err != nil {
		return err
	}

	if err := a.updateApplyResult(); err != nil {
		return err
	}

	if a.ops.pending() {
		if !log.Quiet {
			a.displayPendingOps()
		}

		if err := a.executeOps(); err != nil {
			return err
		}

		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Completed apply for acl definition %q", a.localDef.Metadata.Name)
	} else {
		log.Infof("No changes to apply for acl definition %q", a.localDef.Metadata.Name)
	}

	return nil
}

// createLocal creates the local definition.
func (a *applier) createLocal() error {
	switch a.opts.DefinitionFormat {
	case opt.YAMLFormat:
		if err := yaml.Unmarshal([]byte(a.defDoc), &a.localDef); err != nil {
			return err
		}
	case opt.JSONFormat:
		if err := json.Unmarshal([]byte(a.defDoc), &a.localDef); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format")
	}

	// Explode the local acl entry groups to one entry per group.
	var explodedACLs def.ACLEntryGroups
	for _, group := range a.localDef.Spec.ACLs {
		for _, principal := range group.Principals {
			for _, host := range group.Hosts {
				for _, operation := range group.Operations {
					explodedACLs = append(explodedACLs, def.ACLEntryGroup{
						Principals:     []string{principal},
						Hosts:          []string{host},
						Operations:     []string{operation},
						PermissionType: group.PermissionType,
					})
				}
			}
		}
	}
	explodedACLs.Sort()
	a.localDef.Spec.ACLs = explodedACLs

	a.res.LocalDef = &a.localDef

	return nil
}

// fetchRemote fetches the remote definition and necessary metadata.
func (a *applier) fetchRemote() error {
	log.Infof("Fetching ACLs...")
	var err error
	a.remoteACLs, err = a.srv.DescribeResourceACLs(
		a.localDef.Metadata.Name,
		a.localDef.Metadata.Type,
	)
	if err != nil {
		return err
	}

	a.remoteDef = def.NewACLDefinition(
		a.localDef.Metadata.Name,
		a.localDef.Metadata.Type,
		a.remoteACLs,
	)

	return nil
}

// buildOps builds acl operations.
func (a *applier) buildOps() error {
	if err := a.buildACLOps(); err != nil {
		return err
	}
	return nil
}

// updateApplyResult updates the apply result with the remote definition and human readable diff.
func (a *applier) updateApplyResult() error {
	remoteCopy := a.remoteDef.Copy()

	// Modify the remote definition to remove optional properties not specified in local.
	// Further, set properties that are local only and have no remote state.
	remoteCopy.Spec.DeleteUndefinedACLs = a.localDef.Spec.DeleteUndefinedACLs

	if !a.localDef.Spec.DeleteUndefinedACLs {
		// Remove ACLs from the remote def that are not in local to prevent them showing in the diff.
		_, intersection := acls.DiffPatchIntersection(remoteCopy.Spec.ACLs, a.localDef.Spec.ACLs)
		intersection.Sort()
		remoteCopy.Spec.ACLs = intersection
	}

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
	log.Infof("acl definition %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// executeOps executes update operations.
func (a *applier) executeOps() error {
	if len(a.ops.addACLs) > 0 {
		if err := a.addACLs(); err != nil {
			return err
		}
	}

	if len(a.ops.deleteACLs) > 0 {
		if err := a.deleteACLs(); err != nil {
			return err
		}
	}

	return nil
}

// buildACLOps builds acl operations.
func (a *applier) buildACLOps() error {
	log.Debugf("Comparing local and remote ACLs for definition %q", a.localDef.Metadata.Name)

	a.ops.addACLs, _ = acls.DiffPatchIntersection(a.localDef.Spec.ACLs, a.remoteACLs)
	if a.localDef.Spec.DeleteUndefinedACLs {
		a.ops.deleteACLs, _ = acls.DiffPatchIntersection(a.remoteACLs, a.localDef.Spec.ACLs)
	}

	return nil
}

// addACLs adds ACLs.
func (a *applier) addACLs() error {
	if a.opts.DryRun {
		log.Infof("Skipped adding ACLs (dry-run not available)")
	} else {
		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Adding ACLs...")
		if err := a.srv.CreateACLs(
			a.localDef.Metadata.Name,
			a.localDef.Metadata.Type,
			a.ops.addACLs,
		); err != nil {
			return err
		}
		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Added ACLs for definition %q", a.localDef.Metadata.Name)
	}

	return nil
}

// deleteACLs deletes ACLs.
func (a *applier) deleteACLs() error {
	if a.opts.DryRun {
		log.Infof("Skipped deleting ACLs (dry-run not available)")
	} else {
		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Deleting ACLs...")
		if err := a.srv.DeleteACLs(
			a.localDef.Metadata.Name,
			a.localDef.Metadata.Type,
			a.ops.deleteACLs,
		); err != nil {
			return err
		}
		log.InfoMaybeWithKeyf("dry-run", a.opts.DryRun, "Deleted ACLs for definition %q", a.localDef.Metadata.Name)
	}

	return nil
}
