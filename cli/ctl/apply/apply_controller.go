package apply

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/ctl/apply/docparse"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/operators/acl"
	"github.com/peter-evans/kdef/core/operators/broker"
	"github.com/peter-evans/kdef/core/operators/brokers"
	"github.com/peter-evans/kdef/core/operators/topic"
)

const cannotContinueOnError = "cannot continue on error"

type applier interface {
	Execute() *res.ApplyResult
}

// Options to configure an apply controller
type ControllerOptions struct {
	// ApplierOptions
	ReassAwaitTimeout int
	DefinitionFormat  opt.DefinitionFormat
	DryRun            bool

	// Apply controller specific
	ContinueOnError bool
	ExitCode        bool
	JSONOutput      bool
}

// Create a new apply controller
func NewApplyController(
	cl *client.Client,
	args []string,
	opts ControllerOptions,
) *applyController { //revive:disable-line:unexported-return
	return &applyController{
		cl:   cl,
		args: args,
		opts: opts,
	}
}

// An apply controller
type applyController struct {
	// constructor params
	cl   *client.Client
	args []string
	opts ControllerOptions
}

// Execute the apply controller
func (a *applyController) Execute() error {
	results := res.ApplyResults{}
	var ctlErrors bool

	if a.args[0] == "-" {
		// Apply defs from stdin
		res, err := a.applyDefsFromStdin()
		results = append(results, res...)
		if err != nil {
			log.Error(err)
			ctlErrors = true
		}
	} else {
		// Apply defs from file
		for _, arg := range a.args {
			basepath, pattern := doublestar.SplitPattern(arg)
			fsys := os.DirFS(basepath)

			err := doublestar.GlobWalk(fsys, pattern, func(p string, d fs.DirEntry) error {
				if d.IsDir() {
					return nil
				}

				res, err := a.applyDefsFromFile(filepath.Join(basepath, p))
				results = append(results, res...)
				if err != nil {
					log.Error(err)
					ctlErrors = true
				}
				if (err != nil || res.ContainsErr()) && !a.opts.ContinueOnError {
					return fmt.Errorf(cannotContinueOnError)
				}

				return nil
			})
			if err != nil {
				if err.Error() != cannotContinueOnError {
					log.Error(err)
					ctlErrors = true
				}
				if !a.opts.ContinueOnError {
					break
				}
			}
		}
	}

	// Output JSON
	if a.opts.JSONOutput {
		out, err := results.JSON()
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", out)
	}

	// Log an error if no valid resource definitions were found
	if len(results) == 0 {
		log.Error(fmt.Errorf("no valid resource definitions found"))
		ctlErrors = true
	}

	// Check if the apply controller had errors or the apply results contains any errors
	if ctlErrors || results.ContainsErr() {
		return fmt.Errorf("apply completed with errors")
	}

	// Cause the program to exit with 1 if there are unapplied changes
	if a.opts.ExitCode && results.ContainsUnappliedChanges() {
		return fmt.Errorf("unapplied changes exist")
	}

	return nil
}

// Apply resource definitions from stdin
func (a *applyController) applyDefsFromStdin() (res.ApplyResults, error) {
	log.Infof("Reading definition(s) from stdin")
	defDocs, err := docparse.FromStdin(docparse.Format(a.opts.DefinitionFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to read definition(s): %v", err)
	}
	return a.applyDefinitions(defDocs)
}

// Apply resource definitions from file
func (a *applyController) applyDefsFromFile(filepath string) (res.ApplyResults, error) {
	log.Infof("Reading definition(s) from file %q", filepath)
	defDocs, err := docparse.FromFile(filepath, docparse.Format(a.opts.DefinitionFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to read definition(s): %v", err)
	}
	return a.applyDefinitions(defDocs)
}

// Apply resource definitions using the associated applier
func (a *applyController) applyDefinitions(defDocs []string) (res.ApplyResults, error) {
	resourceDefs, err := getResourceDefinitions(defDocs, a.opts.DefinitionFormat)
	if err != nil {
		return nil, fmt.Errorf("invalid resource definition(s): %v", err)
	}

	var results res.ApplyResults
	for i, resourceDef := range resourceDefs {
		var applier applier

		switch resourceDef.Kind {
		case "acl":
			applier = acl.NewApplier(a.cl, defDocs[i], acl.ApplierOptions{
				DefinitionFormat: a.opts.DefinitionFormat,
				DryRun:           a.opts.DryRun,
			})
		case "broker":
			applier = broker.NewApplier(a.cl, defDocs[i], broker.ApplierOptions{
				DefinitionFormat: a.opts.DefinitionFormat,
				DryRun:           a.opts.DryRun,
			})
		case "brokers":
			applier = brokers.NewApplier(a.cl, defDocs[i], brokers.ApplierOptions{
				DefinitionFormat: a.opts.DefinitionFormat,
				DryRun:           a.opts.DryRun,
			})
		case "topic":
			applier = topic.NewApplier(a.cl, defDocs[i], topic.ApplierOptions{
				DefinitionFormat:  a.opts.DefinitionFormat,
				DryRun:            a.opts.DryRun,
				ReassAwaitTimeout: a.opts.ReassAwaitTimeout,
			})
		}

		res := applier.Execute()
		results = append(results, res)
		if res.GetErr() != nil && !a.opts.ContinueOnError {
			return results, nil
		}
	}

	return results, nil
}

// Get resource definitions for the definition documents
func getResourceDefinitions(defDocs []string, format opt.DefinitionFormat) ([]def.ResourceDefinition, error) {
	kinds := make([]def.ResourceDefinition, len(defDocs))

	for i, defDoc := range defDocs {
		var resourceDef def.ResourceDefinition

		switch format {
		case opt.YAMLFormat:
			if err := yaml.Unmarshal([]byte(defDoc), &resourceDef); err != nil {
				return nil, err
			}
		case opt.JSONFormat:
			if err := json.Unmarshal([]byte(defDoc), &resourceDef); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported format")
		}

		if err := resourceDef.ValidateResource(); err != nil {
			return nil, err
		}

		kinds[i] = resourceDef
	}

	return kinds, nil
}
