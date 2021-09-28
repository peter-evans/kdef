package def

// Map of resource configs
type ConfigsMap map[string]*string

// The source of a config item
type ConfigSource int8

// Config sources
const (
	ConfigSourceUnknown                    ConfigSource = 0
	ConfigSourceDynamicTopicConfig         ConfigSource = 1
	ConfigSourceDynamicBrokerConfig        ConfigSource = 2
	ConfigSourceDynamicDefaultBrokerConfig ConfigSource = 3
	ConfigSourceStaticBrokerConfig         ConfigSource = 4
	ConfigSourceDefaultConfig              ConfigSource = 5
	ConfigSourceDynamicBrokerLoggerConfig  ConfigSource = 6
)

// A config item
type ConfigItem struct {
	Name        string
	Value       *string
	IsSensitive bool
	ReadOnly    bool
	Source      ConfigSource
}

// An array of ConfigItem
type Configs []ConfigItem

// A map of the configs
func (c Configs) ToMap() ConfigsMap {
	configsMap := ConfigsMap{}
	for _, config := range c {
		configsMap[config.Name] = config.Value
	}
	return configsMap
}
