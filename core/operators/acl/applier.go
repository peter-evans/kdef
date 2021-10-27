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

// Options to configure an applier
type ApplierOptions struct {
	DefinitionFormat opt.DefinitionFormat
	DryRun           bool
}

// Create a new applier
func NewApplier(
	cl *client.Client,
	defDoc string,
	opts ApplierOptions,
) *applier {
	return &applier{
		srv:    kafka.NewService(cl),
		defDoc: defDoc,
		opts:   opts,
	}
}

// Applier operations
type applierOps struct {
	addAcls    def.AclEntryGroups
	deleteAcls def.AclEntryGroups
}

// Determine if there are any pending operations
func (a applierOps) pending() bool {
	return len(a.addAcls) > 0 ||
		len(a.deleteAcls) > 0
}

// An applier handling the apply operation
type applier struct {
	// constructor params
	srv    *kafka.Service
	defDoc string
	opts   ApplierOptions

	// internal
	localDef   def.AclDefinition
	remoteDef  def.AclDefinition
	remoteAcls def.AclEntryGroups
	ops        applierOps

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
		if a.ops.pending() && !a.opts.DryRun {
			a.res.Applied = true
		}
	}

	return &a.res
}

// Perform the apply operation sequence
func (a *applier) apply() error {
	// Create the local definition
	if err := a.createLocal(); err != nil {
		return err
	}

	log.Debug("Validating acl definition")
	if err := a.localDef.Validate(); err != nil {
		return err
	}

	// Fetch the remote definition and necessary metadata
	if err := a.fetchRemote(); err != nil {
		return err
	}

	// Build acl operations
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

		log.InfoMaybeWithKey("dry-run", a.opts.DryRun, "Completed apply for acl %q", a.localDef.Metadata.Name)
	} else {
		log.Info("No changes to apply for acl %q", a.localDef.Metadata.Name)
	}

	return nil
}

// Create the local definition
func (a *applier) createLocal() error {
	switch a.opts.DefinitionFormat {
	case opt.YamlFormat:
		if err := yaml.Unmarshal([]byte(a.defDoc), &a.localDef); err != nil {
			return err
		}
	case opt.JsonFormat:
		if err := json.Unmarshal([]byte(a.defDoc), &a.localDef); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format")
	}

	// Explode the local acl entry groups to one entry per group
	var explodedAcls def.AclEntryGroups
	for _, group := range a.localDef.Spec.Acls {
		for _, principal := range group.Principals {
			for _, host := range group.Hosts {
				for _, operation := range group.Operations {
					explodedAcls = append(explodedAcls, def.AclEntryGroup{
						Principals:     []string{principal},
						Hosts:          []string{host},
						Operations:     []string{operation},
						PermissionType: group.PermissionType,
					})
				}
			}
		}
	}
	explodedAcls.Sort()
	a.localDef.Spec.Acls = explodedAcls

	// Set the local definition on the apply result
	a.res.LocalDef = &a.localDef

	return nil
}

// Fetch the remote definition and necessary metadata
func (a *applier) fetchRemote() error {
	log.Info("Fetching acls...")
	var err error
	a.remoteAcls, err = a.srv.DescribeResourceAcls(
		a.localDef.Metadata.Name,
		a.localDef.Metadata.Type,
	)
	if err != nil {
		return err
	}

	a.remoteDef = def.NewAclDefinition(
		a.localDef.Metadata.Name,
		a.localDef.Metadata.Type,
		a.remoteAcls,
	)

	return nil
}

// Build acl operations
func (a *applier) buildOps() error {
	if err := a.buildAclOps(); err != nil {
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
	remoteCopy.Spec.DeleteUndefinedAcls = a.localDef.Spec.DeleteUndefinedAcls

	if !a.localDef.Spec.DeleteUndefinedAcls {
		// Remove acls from the remote def that are not in local to prevent them showing in the diff
		_, intersection := acls.DiffPatchIntersection(remoteCopy.Spec.Acls, a.localDef.Spec.Acls)
		intersection.Sort()
		remoteCopy.Spec.Acls = intersection
	}

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
	log.Info("Acl %q diff (local -> remote):", a.localDef.Metadata.Name)
	fmt.Print(a.res.Diff)
}

// Execute topic update operations
func (a *applier) executeOps() error {
	if len(a.ops.addAcls) > 0 {
		if err := a.addAcls(); err != nil {
			return err
		}
	}

	if len(a.ops.deleteAcls) > 0 {
		if err := a.deleteAcls(); err != nil {
			return err
		}
	}

	return nil
}

// Build acl operations
func (a *applier) buildAclOps() error {
	log.Debug("Comparing local and remote definition acls for %q", a.localDef.Metadata.Name)

	a.ops.addAcls, _ = acls.DiffPatchIntersection(a.localDef.Spec.Acls, a.remoteAcls)
	if a.localDef.Spec.DeleteUndefinedAcls {
		a.ops.deleteAcls, _ = acls.DiffPatchIntersection(a.remoteAcls, a.localDef.Spec.Acls)
	}

	return nil
}

// Add acls
func (a *applier) addAcls() error {
	if a.opts.DryRun {
		log.Info("Skipped adding acls (dry-run not available)")
	} else {
		log.InfoMaybeWithKey("dry-run", a.opts.DryRun, "Adding acls...")
		if err := a.srv.CreateAcls(
			a.localDef.Metadata.Name,
			a.localDef.Metadata.Type,
			a.ops.addAcls,
		); err != nil {
			return err
		} else {
			log.InfoMaybeWithKey("dry-run", a.opts.DryRun, "Added acls for %q", a.localDef.Metadata.Name)
		}
	}

	return nil
}

// Delete acls
func (a *applier) deleteAcls() error {
	if a.opts.DryRun {
		log.Info("Skipped deleting acls (dry-run not available)")
	} else {
		log.InfoMaybeWithKey("dry-run", a.opts.DryRun, "Deleting acls...")
		if err := a.srv.DeleteAcls(
			a.localDef.Metadata.Name,
			a.localDef.Metadata.Type,
			a.ops.deleteAcls,
		); err != nil {
			return err
		} else {
			log.InfoMaybeWithKey("dry-run", a.opts.DryRun, "Deleted acls for %q", a.localDef.Metadata.Name)
		}
	}

	return nil
}
