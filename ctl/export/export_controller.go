package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/operators/broker"
	"github.com/peter-evans/kdef/core/operators/brokers"
	"github.com/peter-evans/kdef/core/operators/topic"
)

// Options to configure an export controller
type ExportControllerOptions struct {
	// ExporterOptions
	Match           string
	Exclude         string
	IncludeInternal bool
	Assignments     opt.Assignments

	// Export controller specific
	DefinitionFormat opt.DefinitionFormat
	OutputDir        string
	Overwrite        bool
}

// An export controller
type exportController struct {
	// constructor params
	cl   *client.Client
	args []string
	opts ExportControllerOptions
	kind string
}

// Create a new export controller
func NewExportController(
	cl *client.Client,
	args []string,
	opts ExportControllerOptions,
	kind string,
) *exportController {
	return &exportController{
		cl:   cl,
		args: args,
		opts: opts,
		kind: kind,
	}
}

// Execute the export controller
func (e *exportController) Execute() error {
	results, err := e.exportResources()
	if err != nil {
		return err
	}
	if results == nil {
		log.Info("No %s(s) found", e.kind)
		return nil
	}

	log.Info("Exporting %d %s definition(s)...", len(results), e.kind)

	stdout := len(e.opts.OutputDir) == 0
	if stdout && e.opts.DefinitionFormat == opt.JsonFormat {
		// For JSON to stdout the def docs are returned as an array
		defDocBytes, err := getDefDocBytes(results.Defs(), e.opts.DefinitionFormat)
		if err != nil {
			return err
		}
		// Ignores --quiet
		fmt.Print(string(defDocBytes))
	} else {
		for _, result := range results {
			defDocBytes, err := getDefDocBytes(result.Def, e.opts.DefinitionFormat)
			if err != nil {
				return err
			}

			if stdout {
				// Ignores --quiet
				fmt.Printf("---\n%s", string(defDocBytes))
			} else {
				outputPath := filepath.Join(e.opts.OutputDir, fmt.Sprintf("%s.%s", result.Id, e.opts.DefinitionFormat.Ext()))

				if !e.opts.Overwrite {
					if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
						log.Info("Skipping overwrite of existing file %q", outputPath)
						continue
					}
				}

				log.Info("Writing %s definition file %q", e.kind, outputPath)
				if err = ioutil.WriteFile(outputPath, defDocBytes, 0644); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Execute a resource's exporter
func (e *exportController) exportResources() (res.ExportResults, error) {
	var results res.ExportResults
	var err error

	switch e.kind {
	case "broker":
		exporter := broker.NewExporter(e.cl)
		results, err = exporter.Execute()
	case "brokers":
		exporter := brokers.NewExporter(e.cl)
		results, err = exporter.Execute()
	case "topic":
		exporter := topic.NewExporter(e.cl, topic.ExporterOptions{
			Match:           e.opts.Match,
			Exclude:         e.opts.Exclude,
			IncludeInternal: e.opts.IncludeInternal,
			Assignments:     e.opts.Assignments,
		})
		results, err = exporter.Execute()
	}

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Get a byte array for the definition document
func getDefDocBytes(def interface{}, format opt.DefinitionFormat) ([]byte, error) {
	var defDocBytes []byte
	var err error

	switch format {
	case opt.YamlFormat:
		defDocBytes, err = yaml.Marshal(def)
	case opt.JsonFormat:
		defDocBytes, err = json.MarshalIndent(def, "", "  ")
		defDocBytes = append(defDocBytes, "\n"...)
	default:
		return nil, fmt.Errorf("unsupported format")
	}

	if err != nil {
		return nil, err
	} else {
		return defDocBytes, nil
	}
}
