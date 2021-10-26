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
	"github.com/peter-evans/kdef/core/operators/acl"
	"github.com/peter-evans/kdef/core/operators/broker"
	"github.com/peter-evans/kdef/core/operators/brokers"
	"github.com/peter-evans/kdef/core/operators/topic"
)

type exporter interface {
	Execute() (res.ExportResults, error)
}

// Options to configure an export controller
type ExportControllerOptions struct {
	// ExporterOptions for topic/acl
	Match   string
	Exclude string

	// TODO: add topic prefix
	// ExporterOptions for topic
	IncludeInternal bool
	Assignments     opt.Assignments

	// ExporterOptions for acl
	AclResourceType string
	AclAutoGroup    bool

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
				outputPath := filepath.Join(
					e.opts.OutputDir,
					result.Type,
					fmt.Sprintf("%s.%s", result.Id, e.opts.DefinitionFormat.Ext()),
				)

				dirPath := filepath.Dir(outputPath)
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return fmt.Errorf("failed to create directory path %q: %v", dirPath, err)
				}

				if !e.opts.Overwrite {
					if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
						log.Info("Skipping overwrite of existing file %q", outputPath)
						continue
					}
				}

				log.Info("Writing %s definition file %q", e.kind, outputPath)
				if err = ioutil.WriteFile(outputPath, defDocBytes, 0666); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Execute a resource's exporter
func (e *exportController) exportResources() (res.ExportResults, error) {
	var exporter exporter
	switch e.kind {
	case "acl":
		exporter = acl.NewExporter(e.cl, acl.ExporterOptions{
			Match:        e.opts.Match,
			Exclude:      e.opts.Exclude,
			ResourceType: e.opts.AclResourceType,
			AutoGroup:    e.opts.AclAutoGroup,
		})
	case "broker":
		exporter = broker.NewExporter(e.cl)
	case "brokers":
		exporter = brokers.NewExporter(e.cl)
	case "topic":
		exporter = topic.NewExporter(e.cl, topic.ExporterOptions{
			Match:           e.opts.Match,
			Exclude:         e.opts.Exclude,
			IncludeInternal: e.opts.IncludeInternal,
			Assignments:     e.opts.Assignments,
		})
	}

	results, err := exporter.Execute()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Get the byte array of a definition document
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
