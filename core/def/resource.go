package def

import (
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/util/str"
)

// Top-level definition of a resource
type ResourceDefinition struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

// Definition kinds and associated versions
var definitionKindVersions = map[string][]string{
	"topic": {"v1"},
}

// Validate a resource definition
func (t ResourceDefinition) validate() error {
	if versions, ok := definitionKindVersions[t.Kind]; ok {
		if !str.Contains(t.ApiVersion, versions) {
			return fmt.Errorf("invalid definition apiVersion %q", t.ApiVersion)
		}
	} else {
		return fmt.Errorf("invalid definition kind %q", t.Kind)
	}

	return nil
}

// Get resource definitions contained in yaml documents
func GetResourceDefinitions(yamlDocs []string) ([]ResourceDefinition, error) {
	kinds := make([]ResourceDefinition, len(yamlDocs))

	for i, yamlDoc := range yamlDocs {
		var resourceDef ResourceDefinition
		if err := yaml.Unmarshal([]byte(yamlDoc), &resourceDef); err != nil {
			return nil, err
		}

		if err := resourceDef.validate(); err != nil {
			return nil, err
		}

		kinds[i] = resourceDef
	}

	return kinds, nil
}
