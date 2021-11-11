package def

import (
	"fmt"

	"github.com/peter-evans/kdef/core/util/str"
)

// Definition kinds and associated versions
var definitionKindVersions = map[string][]string{
	"acl":     {"v1"},
	"broker":  {"v1"},
	"brokers": {"v1"},
	"topic":   {"v1"},
}

// Resource metadata labels
type ResourceMetadataLabels map[string]string

// Resource metadata definition
type ResourceMetadataDefinition struct {
	Labels ResourceMetadataLabels `json:"labels,omitempty"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type,omitempty"`
}

// Top-level definition of a resource
type ResourceDefinition struct {
	APIVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Metadata   ResourceMetadataDefinition `json:"metadata"`
}

// Validate resource definition
func (r ResourceDefinition) ValidateResource() error {
	if versions, ok := definitionKindVersions[r.Kind]; ok {
		if !str.Contains(r.APIVersion, versions) {
			return fmt.Errorf("invalid definition apiVersion %q", r.APIVersion)
		}
	} else {
		return fmt.Errorf("invalid definition kind %q", r.Kind)
	}

	if len(r.Metadata.Name) == 0 {
		return fmt.Errorf("metadata name must be supplied")
	}

	return nil
}
