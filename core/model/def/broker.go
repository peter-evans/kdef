// Package def implements definitions for Kafka resources.
package def

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/gotidy/copy"
	"github.com/peter-evans/kdef/core/model/meta"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/util/i32"
)

// BrokerSpecDefinition represents a broker spec definition.
type BrokerSpecDefinition struct {
	Configs                ConfigsMap `json:"configs,omitempty"`
	DeleteUndefinedConfigs bool       `json:"deleteUndefinedConfigs"`
}

// BrokerDefinition represents a broker resource definition.
type BrokerDefinition struct {
	ResourceDefinition
	Spec BrokerSpecDefinition `json:"spec"`
}

// Copy creates a copy of this BrokerDefinition.
func (b BrokerDefinition) Copy() BrokerDefinition {
	copiers := copy.New()
	copier := copiers.Get(&BrokerDefinition{}, &BrokerDefinition{})
	var brokerDefCopy BrokerDefinition
	copier.Copy(&brokerDefCopy, &b)
	return brokerDefCopy
}

// Validate validates the definition.
func (b BrokerDefinition) Validate() error {
	if err := b.ValidateResource(); err != nil {
		return err
	}

	if _, err := i32.ParseStr(b.Metadata.Name); err != nil {
		return fmt.Errorf("metadata name must be an integer broker id")
	}

	return nil
}

// ValidateWithMetadata further validates the definition using metadata.
func (b BrokerDefinition) ValidateWithMetadata(brokers meta.Brokers) error {
	// Check the value of metadata name is a valid broker ID
	brokerID, err := i32.ParseStr(b.Metadata.Name)
	if err != nil {
		return err
	}
	if !i32.Contains(brokerID, brokers.IDs()) {
		return fmt.Errorf("metadata name must be the id of an available broker")
	}

	return nil
}

// NewBrokerDefinition creates a broker definition from metadata and config.
func NewBrokerDefinition(
	metadata ResourceMetadataDefinition,
	configsMap ConfigsMap,
) BrokerDefinition {
	brokerDef := BrokerDefinition{
		ResourceDefinition: ResourceDefinition{
			APIVersion: "v1",
			Kind:       "broker",
			Metadata:   metadata,
		},
		Spec: BrokerSpecDefinition{
			Configs: configsMap,
		},
	}

	return brokerDef
}

// LoadBrokerDefinition loads a broker definition from a document.
func LoadBrokerDefinition(
	defDoc string,
	format opt.DefinitionFormat,
) (BrokerDefinition, error) {
	var def BrokerDefinition

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
