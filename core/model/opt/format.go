package opt

type DefinitionFormat int8

const (
	UnsupportedFormat DefinitionFormat = 0
	YAMLFormat        DefinitionFormat = 1
	JSONFormat        DefinitionFormat = 2
)

// Valid values for definition format
var DefinitionFormatValidValues = []string{"yaml", "json"}

// Parse a definition format from a string
func ParseDefinitionFormat(format string) DefinitionFormat {
	switch format {
	case "yaml":
		return YAMLFormat
	case "json":
		return JSONFormat
	default:
		return UnsupportedFormat
	}
}

// File extension for the format
func (d DefinitionFormat) Ext() string {
	switch d {
	case YAMLFormat:
		return "yml"
	case JSONFormat:
		return "json"
	default:
		return "unsupported"
	}
}
