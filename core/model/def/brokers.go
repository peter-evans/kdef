// Package def implements definitions for Kafka resources.
package def

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/core/model/opt"
)

// BrokersSpecDefinition represents a brokers spec definition.
type BrokersSpecDefinition struct {
	Configs                ConfigsMap `json:"configs,omitempty"`
	DeleteUndefinedConfigs bool       `json:"deleteUndefinedConfigs"`
}

// BrokersDefinition represents a brokers resource definition.
type BrokersDefinition struct {
	ResourceDefinition
	Spec BrokersSpecDefinition `json:"spec"`
}

// Copy creates a copy of this BrokersDefinition.
func (b BrokersDefinition) Copy() BrokersDefinition {
	copiers := copy.New()
	copier := copiers.Get(&BrokersDefinition{}, &BrokersDefinition{})
	var brokersDefCopy BrokersDefinition
	copier.Copy(&brokersDefCopy, &b)
	return brokersDefCopy
}

// Validate validates the definition.
func (b BrokersDefinition) Validate() error {
	if err := b.ValidateResource(); err != nil {
		return err
	}

	return nil
}

// NewBrokersDefinition creates a brokers definition from metadata and config.
func NewBrokersDefinition(
	metadata ResourceMetadataDefinition,
	configsMap ConfigsMap,
) BrokersDefinition {
	brokersDef := BrokersDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "brokers",
			Metadata:   metadata,
		},
		Spec: BrokersSpecDefinition{
			Configs: configsMap,
		},
	}

	return brokersDef
}

// LoadBrokersDefinition loads a brokers definition from a document.
func LoadBrokersDefinition(
	defDoc string,
	format opt.DefinitionFormat,
) (BrokersDefinition, error) {
	var def BrokersDefinition

	switch format {
	case opt.YAMLFormat:
		if err := yaml.Unmarshal([]byte(defDoc), &def); err != nil {
			return def, err
		}
	case opt.JSONFormat:
		if err := json.Unmarshal([]byte(defDoc), &def); err != nil {
			return def, err
		}
	default:
		return def, fmt.Errorf("unsupported format")
	}

	return def, nil
}
