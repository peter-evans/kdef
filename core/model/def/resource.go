// Package def implements definitions for Kafka resources.
package def

import (
	"fmt"

	"github.com/peter-evans/kdef/core/util/str"
)

var definitionKindVersions = map[string][]string{
	KindACL:     {"v1"},
	KindBroker:  {"v1"},
	KindBrokers: {"v1"},
	KindTopic:   {"v1"},
}

// ResourceMetadataLabels represents resource metadata labels.
type ResourceMetadataLabels map[string]string

// ResourceMetadataDefinition represents a resource metadata definition.
type ResourceMetadataDefinition struct {
	Labels ResourceMetadataLabels `json:"labels,omitempty"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type,omitempty"`
}

// ResourceDefinition represents a resource definition.
type ResourceDefinition struct {
	APIVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Metadata   ResourceMetadataDefinition `json:"metadata"`
}

// ValidateResource validates the resource definition.
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
