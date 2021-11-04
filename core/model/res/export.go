package res

import (
	"encoding/json"

	"github.com/bradfitz/slice" //nolint
)

// An export result
type ExportResult struct {
	// TODO: Change Id to name?
	Id   string      `json:"id"`
	Type string      `json:"type,omitempty"`
	Def  interface{} `json:"definition"`
}

// A slice of ExportResult
type ExportResults []ExportResult

// Sort by ID
func (e ExportResults) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8
	//nolint
	slice.Sort(e[:], func(i, j int) bool {
		return e[i].Type < e[j].Type ||
			e[i].Type == e[j].Type && e[i].Id < e[j].Id
	})
}

// A slice of the IDs
func (e ExportResults) Ids() []string {
	ids := make([]string, len(e))
	for i, r := range e {
		ids[i] = r.Id
	}
	return ids
}

// A slice of the definitions
func (e ExportResults) Defs() []interface{} {
	defs := make([]interface{}, len(e))
	for i, r := range e {
		defs[i] = r.Def
	}
	return defs
}

// Convert export results to JSON
func (e ExportResults) JSON() (string, error) {
	j, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(j), nil
}
