package def

import (
	"encoding/json"

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

// Convert a brokers definition to JSON
func (b BrokersDefinition) JSON() (string, error) {
	j, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	return string(j), nil
}

// Create a copy of this BrokersDefinition
func (b BrokersDefinition) Copy() BrokersDefinition {
	copiers := copy.New()
	copier := copiers.Get(&BrokersDefinition{}, &BrokersDefinition{})
	brokersDefCopy := BrokersDefinition{}
	copier.Copy(&brokersDefCopy, &b)
	return brokersDefCopy
}

// Validate a brokers definition
func (b BrokersDefinition) Validate() error {
	if err := b.ValidateResource(); err != nil {
		return err
	}

	return nil
}

// Create a brokers definition from metadata and config
func NewBrokersDefinition(
	name string,
	configsMap ConfigsMap,
) BrokersDefinition {
	brokersDef := BrokersDefinition{
		ResourceDefinition: ResourceDefinition{
			ApiVersion: "v1",
			Kind:       "brokers",
			Metadata: ResourceMetadataDefinition{
				Name: name,
			},
		},
		Spec: BrokersSpecDefinition{
			Configs: configsMap,
		},
	}

	return brokersDef
}
