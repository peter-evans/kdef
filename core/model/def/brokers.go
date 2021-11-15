// Package def implements definitions for Kafka resources.
package def

import (
	"github.com/gotidy/copy"
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
func NewBrokersDefinition(configsMap ConfigsMap) BrokersDefinition {
	brokersDef := BrokersDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "brokers",
			Metadata: ResourceMetadataDefinition{
				Name: "brokers",
			},
		},
		Spec: BrokersSpecDefinition{
			Configs: configsMap,
		},
	}

	return brokersDef
}
