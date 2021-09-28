package def

import (
	"fmt"

	"github.com/peter-evans/kdef/util/str"
)

// Top-level definition of a resource
type ResourceDefinition struct {
	ApiVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Metadata   ResourceMetadataDefinition `json:"metadata"`
}

// Resource metadata definition
type ResourceMetadataDefinition struct {
	Labels ResourceMetadataLabels `json:"labels,omitempty"`
	Name   string                 `json:"name"`
}

// Resource metadata labels
type ResourceMetadataLabels map[string]string

// Definition kinds and associated versions
var definitionKindVersions = map[string][]string{
	"brokers": {"v1"},
	"topic":   {"v1"},
}

// Validate a resource definition
func (r ResourceDefinition) ValidateResource() error {
	if versions, ok := definitionKindVersions[r.Kind]; ok {
		if !str.Contains(r.ApiVersion, versions) {
			return fmt.Errorf("invalid definition apiVersion %q", r.ApiVersion)
		}
	} else {
		return fmt.Errorf("invalid definition kind %q", r.Kind)
	}

	if len(r.Metadata.Name) == 0 {
		return fmt.Errorf("metadata name must be supplied")
	}

	return nil
}
