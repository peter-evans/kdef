// Package opt implements configuration options.
package opt

// DefinitionFormat represents the format of resource definitions.
type DefinitionFormat int8

const (
	UnsupportedFormat DefinitionFormat = 0
	YAMLFormat        DefinitionFormat = 1
	JSONFormat        DefinitionFormat = 2
)

// DefinitionFormatValidValues represents valid values for definition format.
var DefinitionFormatValidValues = []string{"yaml", "json"}

// ParseDefinitionFormat parses a definition format from a string.
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

// Ext returns the file extension for the format.
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
