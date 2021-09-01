package apply

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/peter-evans/kdef/cli/in"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/res"
	"github.com/peter-evans/kdef/core/topics"
)

// Flags to configure an apply controller
type ApplyControllerFlags struct {
	// ApplierFlags
	DeleteMissingConfigs bool
	DryRun               bool
	NonIncremental       bool

	// Apply controller specific
	ContinueOnError bool
	ExitCode        bool
	JsonOutput      bool
}

// An apply controller
type applyController struct {
	// constructor params
	cl    *client.Client
	args  []string
	flags ApplyControllerFlags
}

// Creates a new apply controller
func NewApplyController(
	cl *client.Client,
	args []string,
	flags ApplyControllerFlags,
) *applyController {
	return &applyController{
		cl:    cl,
		args:  args,
		flags: flags,
	}
}

// Executes the apply controller
func (a *applyController) Execute() error {
	var results res.ApplyResults

	if a.args[0] == "-" {
		log.Info("Reading definition(s) from stdin")
		yamlDocs, err := in.StdinToYamlDocs()
		if err != nil {
			return err
		}

		resourceDefs, err := def.GetResourceDefinitions(yamlDocs)
		if err != nil {
			return err
		}

		results = applyYamlDocs(a.cl, a.flags, yamlDocs, resourceDefs)
	} else {
	mainloop:
		for _, arg := range a.args {
			matchCount := 0

			matches, err := filepath.Glob(arg)
			if err != nil {
				return err
			}

			for _, match := range matches {
				matchCount++

				log.Info("Reading definition(s) from file %q", match)
				yamlDocs, err := in.FileToYamlDocs(match)
				if err != nil {
					return err
				}

				resourceDefs, err := def.GetResourceDefinitions(yamlDocs)
				if err != nil {
					return err
				}

				res := applyYamlDocs(a.cl, a.flags, yamlDocs, resourceDefs)
				results = append(results, res...)
				if res.ContainsErr() && !a.flags.ContinueOnError {
					break mainloop
				}

			}

			if matchCount == 0 {
				return errors.New("no definition files found")
			}
		}
	}

	if a.flags.JsonOutput {
		out, err := results.JSON()
		if err != nil {
			return err
		}
		fmt.Print(out)
	}

	// Check the apply results for any errors
	if results.ContainsErr() {
		return fmt.Errorf("apply completed with errors")
	}

	// Cause the program to exit with 1 if there are unapplied changes
	if a.flags.ExitCode && results.ContainsUnappliedChanges() {
		return fmt.Errorf("unapplied changes exist")
	}

	return nil
}

// Apply YAML resource documents using an applier
func applyYamlDocs(
	cl *client.Client,
	flags ApplyControllerFlags,
	yamlDocs []string,
	resourceDefs []def.ResourceDefinition,
) res.ApplyResults {
	var results res.ApplyResults

	for i, resourceDef := range resourceDefs {
		switch resourceDef.Kind {
		case "topic":
			applier := topics.NewApplier(cl, yamlDocs[i], topics.ApplierFlags{
				DeleteMissingConfigs: flags.DeleteMissingConfigs,
				DryRun:               flags.DryRun,
				NonIncremental:       flags.NonIncremental,
			})
			res := applier.Execute()
			results = append(results, res)
			if err := res.GetErr(); err != nil && !flags.ContinueOnError {
				return results
			}
		}
	}

	return results
}
