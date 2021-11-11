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
type ControllerOptions struct {
	// ExporterOptions for topic/acl
	Match   string
	Exclude string

	// ExporterOptions for topic
	TopicIncludeInternal bool
	TopicAssignments     opt.Assignments

	// ExporterOptions for acl
	ACLResourceType string
	ACLAutoGroup    bool

	// Export controller specific
	DefinitionFormat opt.DefinitionFormat
	OutputDir        string
	Overwrite        bool
}

// Create a new export controller
func NewExportController(
	cl *client.Client,
	args []string,
	opts ControllerOptions,
	kind string,
) *exportController { //revive:disable-line:unexported-return
	return &exportController{
		cl:   cl,
		args: args,
		opts: opts,
		kind: kind,
	}
}

// An export controller
type exportController struct {
	// constructor params
	cl   *client.Client
	args []string
	opts ControllerOptions
	kind string
}

// Execute the export controller
func (e *exportController) Execute() error {
	results, err := e.exportResources()
	if err != nil {
		return err
	}
	if results == nil {
		log.Infof("No %s(s) found", e.kind)
		return nil
	}

	log.Infof("Exporting %d %s definition(s)...", len(results), e.kind)

	stdout := len(e.opts.OutputDir) == 0
	if stdout && e.opts.DefinitionFormat == opt.JSONFormat {
		// For JSON to stdout the def docs are returned as a slice
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
					fmt.Sprintf("%s.%s", result.ID, e.opts.DefinitionFormat.Ext()),
				)

				dirPath := filepath.Dir(outputPath)
				if err := os.MkdirAll(dirPath, 0o755); err != nil {
					return fmt.Errorf("failed to create directory path %q: %v", dirPath, err)
				}

				if !e.opts.Overwrite {
					if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
						log.Infof("Skipping overwrite of existing file %q", outputPath)
						continue
					}
				}

				log.Infof("Writing %s definition file %q", e.kind, outputPath)
				if err = ioutil.WriteFile(outputPath, defDocBytes, 0o666); err != nil {
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
			ResourceType: e.opts.ACLResourceType,
			AutoGroup:    e.opts.ACLAutoGroup,
		})
	case "broker":
		exporter = broker.NewExporter(e.cl)
	case "brokers":
		exporter = brokers.NewExporter(e.cl)
	case "topic":
		exporter = topic.NewExporter(e.cl, topic.ExporterOptions{
			Match:           e.opts.Match,
			Exclude:         e.opts.Exclude,
			IncludeInternal: e.opts.TopicIncludeInternal,
			Assignments:     e.opts.TopicAssignments,
		})
	}

	results, err := exporter.Execute()
	if err != nil {
		return nil, err
	}

	return results, nil
}

// Get the byte slice of a definition document
func getDefDocBytes(def interface{}, format opt.DefinitionFormat) ([]byte, error) {
	var defDocBytes []byte
	var err error

	switch format {
	case opt.YAMLFormat:
		defDocBytes, err = yaml.Marshal(def)
	case opt.JSONFormat:
		defDocBytes, err = json.MarshalIndent(def, "", "  ")
		defDocBytes = append(defDocBytes, "\n"...)
	default:
		return nil, fmt.Errorf("unsupported format")
	}

	if err != nil {
		return nil, err
	}

	return defDocBytes, nil
}
