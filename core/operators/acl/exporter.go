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

// Options to configure an exporter
type ExporterOptions struct {
	Match        string
	Exclude      string
	ResourceType string
	AutoGroup    bool
}

// Create a new exporter
func NewExporter(
	cl *client.Client,
	opts ExporterOptions,
) *exporter {
	return &exporter{
		srv:  kafka.NewService(cl),
		opts: opts,
	}
}

// An exporter handling the export operation
type exporter struct {
	// constructor params
	srv  *kafka.Service
	opts ExporterOptions
}

// Execute the export operation
func (e *exporter) Execute() (res.ExportResults, error) {
	log.Info("Fetching acls...")
	aclDefs, err := e.getAclDefinitions()
	if err != nil {
		return nil, err
	}

	if len(aclDefs) == 0 {
		return nil, nil
	}

	results := make(res.ExportResults, len(aclDefs))
	for i, aclDef := range aclDefs {
		results[i] = res.ExportResult{
			Id:   aclDef.Metadata.Name,
			Type: aclDef.Metadata.Type,
			Def:  aclDef,
		}
	}

	results.Sort()

	return results, nil
}

// Return acl definitions for existing resources in a cluster
func (e *exporter) getAclDefinitions() ([]def.AclDefinition, error) {
	resourceAcls, err := e.srv.DescribeAllResourceAcls(
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

	aclDefs := []def.AclDefinition{}
	for _, resource := range resourceAcls {
		if !matchRegExp.MatchString(resource.ResourceName) {
			continue
		}
		if excludeRegExp.MatchString(resource.ResourceName) {
			continue
		}

		resAcls := resource.Acls
		if e.opts.AutoGroup {
			resAcls = acls.MergeGroups(resAcls)
		}

		aclDef := def.NewAclDefinition(
			resource.ResourceName,
			resource.ResourceType,
			resAcls,
		)
		// Default to delete undefined acls
		aclDef.Spec.DeleteUndefinedAcls = true

		aclDefs = append(aclDefs, aclDef)
	}

	return aclDefs, nil
}
