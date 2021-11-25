// Package acl implements operators for acl definition operations.
package acl

import (
	"regexp"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/helpers/acls"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
)

// ExporterOptions represents options to configure an exporter.
type ExporterOptions struct {
	Match        string
	Exclude      string
	ResourceType string
	AutoGroup    bool
}

// NewExporter creates a new exporter.
func NewExporter(
	cl *client.Client,
	opts ExporterOptions,
) *exporter { //revive:disable-line:unexported-return
	return &exporter{
		srv:  kafka.NewService(cl),
		opts: opts,
	}
}

type exporter struct {
	srv  *kafka.Service
	opts ExporterOptions
}

// Execute executes the export operation.
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Infof("Fetching remote ACLs...")
	aclDefs, err := e.getACLDefinitions()
	if err != nil {
		return nil, err
	}

	if len(aclDefs) == 0 {
		return nil, nil
	}

	results := make(res.ExportResults, len(aclDefs))
	for i, aclDef := range aclDefs {
		results[i] = res.ExportResult{
			ID:   aclDef.Metadata.Name,
			Type: aclDef.Metadata.Type,
			Def:  aclDef,
		}
	}

	results.Sort()

	return results, nil
}

func (e *exporter) getACLDefinitions() ([]def.ACLDefinition, error) {
	resourceACLs, err := e.srv.DescribeAllResourceACLs(
		e.opts.ResourceType,
	)
	if err != nil {
		return nil, err
	}

	matchRegExp, err := regexp.Compile(e.opts.Match)
	if err != nil {
		return nil, err
	}
	excludeRegExp, err := regexp.Compile(e.opts.Exclude)
	if err != nil {
		return nil, err
	}

	aclDefs := []def.ACLDefinition{}
	for _, resource := range resourceACLs {
		if !matchRegExp.MatchString(resource.ResourceName) {
			continue
		}
		if excludeRegExp.MatchString(resource.ResourceName) {
			continue
		}

		resACLs := resource.ACLs
		if e.opts.AutoGroup {
			resACLs = acls.MergeGroups(resACLs)
		}

		aclDef := def.NewACLDefinition(
			def.ResourceMetadataDefinition{
				Name: resource.ResourceName,
				Type: resource.ResourceType,
			},
			resACLs,
		)
		// Default to delete undefined ACLs.
		aclDef.Spec.DeleteUndefinedACLs = true

		aclDefs = append(aclDefs, aclDef)
	}

	return aclDefs, nil
}
