package export

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/res"
	"github.com/peter-evans/kdef/core/topics"
)

// Flags to configure an export controller
type ExportControllerFlags struct {
	// ExporterFlags
	Match           string
	Exclude         string
	IncludeInternal bool
	Assignments     string

	// Export controller specific
	OutputDir string
	Overwrite bool
}

// An export controller
type exportController struct {
	// constructor params
	cl    *client.Client
	args  []string
	flags ExportControllerFlags
	kind  string
}

// Create a new export controller
func NewExportController(
	cl *client.Client,
	args []string,
	flags ExportControllerFlags,
	kind string,
) *exportController {
	return &exportController{
		cl:    cl,
		args:  args,
		flags: flags,
		kind:  kind,
	}
}

// Execute the export controller
func (e *exportController) Execute() error {
	results, err := e.exportResources()
	if err != nil {
		return err
	}
	if results == nil {
		log.Info("No %ss found", e.kind)
		return nil
	}

	log.Info("Exporting %d %s definitions...", len(results), e.kind)
	for _, result := range results {
		// Marshal to yaml
		defDocBytes, err := yaml.Marshal(result.Def)
		if err != nil {
			return err
		}

		if len(e.flags.OutputDir) > 0 {
			outputPath := filepath.Join(e.flags.OutputDir, fmt.Sprintf("%s.yml", result.Id))

			if !e.flags.Overwrite {
				if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
					log.Info("Skipping overwrite of existing file %q", outputPath)
					continue
				}
			}

			log.Info("Writing %s definition file %q", e.kind, outputPath)
			if err = ioutil.WriteFile(outputPath, defDocBytes, 0644); err != nil {
				return err
			}
		} else {
			// Ignores --quiet
			fmt.Printf("---\n%s", string(defDocBytes))
		}
	}

	return nil
}

// Execute a resource's exporter
func (e *exportController) exportResources() (res.ExportResults, error) {
	var results res.ExportResults
	var err error

	switch e.kind {
	case "topic":
		exporter := topics.NewExporter(e.cl, topics.ExporterFlags{
			Match:           e.flags.Match,
			Exclude:         e.flags.Exclude,
			IncludeInternal: e.flags.IncludeInternal,
			Assignments:     e.flags.Assignments,
		})
		results, err = exporter.Execute()
	}

	if err != nil {
		return nil, err
	}

	return results, nil
}
