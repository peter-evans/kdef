package def

import (
	"github.com/gotidy/copy"
)

// Top-level brokers definition
type BrokersDefinition struct {
	ResourceDefinition
	Spec BrokersSpecDefinition `json:"spec"`
}

// Brokers spec definition
type BrokersSpecDefinition struct {
	Configs ConfigsMap `json:"configs,omitempty"`
}

// Create a copy of this BrokersDefinition
func (b BrokersDefinition) Copy() BrokersDefinition {
	copiers := copy.New()
	copier := copiers.Get(&BrokersDefinition{}, &BrokersDefinition{})
	brokersDefCopy := BrokersDefinition{}
	copier.Copy(&brokersDefCopy, &b)
	return brokersDefCopy
}

// Validate definition
func (b BrokersDefinition) Validate() error {
	if err := b.ValidateResource(); err != nil {
		return err
	}

	return nil
}

// Create a brokers definition from metadata and config
func NewBrokersDefinition(configsMap ConfigsMap) BrokersDefinition {
	brokersDef := BrokersDefinition{
		ResourceDefinition: ResourceDefinition{
			ApiVersion: "v1",
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
