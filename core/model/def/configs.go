// Package def implements definitions for Kafka resources.
package def

// ConfigsMap represents a map of resource configs.
type ConfigsMap map[string]*string

// ConfigSource represents the source of a config key.
type ConfigSource int8

// ConfigSource types.
const (
	ConfigSourceUnknown                    ConfigSource = 0
	ConfigSourceDynamicTopicConfig         ConfigSource = 1
	ConfigSourceDynamicBrokerConfig        ConfigSource = 2
	ConfigSourceDynamicDefaultBrokerConfig ConfigSource = 3
	ConfigSourceStaticBrokerConfig         ConfigSource = 4
	ConfigSourceDefaultConfig              ConfigSource = 5
	ConfigSourceDynamicBrokerLoggerConfig  ConfigSource = 6
)

// ConfigKey represents a config key.
type ConfigKey struct {
	Name        string
	Value       *string
	IsSensitive bool
	ReadOnly    bool
	Source      ConfigSource
}

// IsDynamic determines if the config key is dynamic.
func (c ConfigKey) IsDynamic() bool {
	return c.Source == ConfigSourceDynamicTopicConfig ||
		c.Source == ConfigSourceDynamicBrokerConfig ||
		c.Source == ConfigSourceDynamicDefaultBrokerConfig ||
		c.Source == ConfigSourceDynamicBrokerLoggerConfig
}

// Configs represents a slice of ConfigKey.
type Configs []ConfigKey

// ToMap returns a map of the configs.
func (c Configs) ToMap() ConfigsMap {
	configsMap := ConfigsMap{}
	for _, config := range c {
		configsMap[config.Name] = config.Value
	}
	return configsMap
}

// ToExportableMap returns an exportable map of the configs (sensitive keys filtered out).
func (c Configs) ToExportableMap() ConfigsMap {
	configsMap := ConfigsMap{}
	for _, config := range c {
		if !config.IsSensitive {
			configsMap[config.Name] = config.Value
		}
	}
	return configsMap
}
