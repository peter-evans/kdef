// Package res implements structures handling the result of operations.
package res

import (
	"encoding/json"

	"github.com/bradfitz/slice" //nolint
)

// ExportResult represents an export result.
type ExportResult struct {
	ID   string      `json:"id"`
	Type string      `json:"type,omitempty"`
	Def  interface{} `json:"definition"`
}

// ExportResults represents a slice of ExportResult.
type ExportResults []ExportResult

// Sort sorts by ID.
func (e ExportResults) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8.
	//nolint
	slice.Sort(e[:], func(i, j int) bool {
		return e[i].Type < e[j].Type ||
			e[i].Type == e[j].Type && e[i].ID < e[j].ID
	})
}

// IDs returns a slice of the IDs.
func (e ExportResults) IDs() []string {
	ids := make([]string, len(e))
	for i, r := range e {
		ids[i] = r.ID
	}
	return ids
}

// Defs returns a slice of the definitions.
func (e ExportResults) Defs() []interface{} {
	defs := make([]interface{}, len(e))
	for i, r := range e {
		defs[i] = r.Def
	}
	return defs
}

// JSON converts export results to JSON.
func (e ExportResults) JSON() (string, error) {
	j, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(j), nil
}
