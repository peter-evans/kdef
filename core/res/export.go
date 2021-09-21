package res

import (
	"encoding/json"

	"github.com/bradfitz/slice"
)

// An export result
type ExportResult struct {
	Id  string      `json:"id"`
	Def interface{} `json:"definition"`
}

// A collection of export results
type ExportResults []ExportResult

// Sort the collection by ID
func (e ExportResults) Sort() {
	// TODO: Use sort.Slice in the standard library after upgrading to Go 1.8
	slice.Sort(e[:], func(i, j int) bool {
		return e[i].Id < e[j].Id
	})
}

// An array of the IDs
func (e ExportResults) Ids() []string {
	ids := make([]string, len(e))
	for i, r := range e {
		ids[i] = r.Id
	}
	return ids
}

// Convert export results to JSON
func (e ExportResults) JSON() (string, error) {
	j, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(j), nil
}
