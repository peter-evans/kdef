package apply

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/peter-evans/kdef/cli/in"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/res"
	"github.com/peter-evans/kdef/core/operators/broker"
	"github.com/peter-evans/kdef/core/operators/brokers"
	"github.com/peter-evans/kdef/core/operators/topic"
)

type applier interface {
	Execute() *res.ApplyResult
}

// Flags to configure an apply controller
type ApplyControllerFlags struct {
	// ApplierFlags
	DryRun         bool
	NonIncremental bool

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

// Create a new apply controller
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

// Execute the apply controller
func (a *applyController) Execute() error {
	var results res.ApplyResults

	if a.args[0] == "-" {
		log.Info("Reading definition(s) from stdin")
		yamlDocs, err := in.StdinToYamlDocs()
		if err != nil {
			return err
		}

		resourceDefs, err := getResourceDefinitions(yamlDocs)
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

				resourceDefs, err := getResourceDefinitions(yamlDocs)
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

// Get resource definitions contained in yaml documents
func getResourceDefinitions(yamlDocs []string) ([]def.ResourceDefinition, error) {
	kinds := make([]def.ResourceDefinition, len(yamlDocs))

	for i, yamlDoc := range yamlDocs {
		var resourceDef def.ResourceDefinition
		if err := yaml.Unmarshal([]byte(yamlDoc), &resourceDef); err != nil {
			return nil, err
		}

		if err := resourceDef.ValidateResource(); err != nil {
			return nil, err
		}

		kinds[i] = resourceDef
	}

	return kinds, nil
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
		var applier applier

		switch resourceDef.Kind {
		case "broker":
			applier = broker.NewApplier(cl, yamlDocs[i], broker.ApplierFlags{
				DryRun:         flags.DryRun,
				NonIncremental: flags.NonIncremental,
			})
		case "brokers":
			applier = brokers.NewApplier(cl, yamlDocs[i], brokers.ApplierFlags{
				DryRun:         flags.DryRun,
				NonIncremental: flags.NonIncremental,
			})
		case "topic":
			applier = topic.NewApplier(cl, yamlDocs[i], topic.ApplierFlags{
				DryRun:         flags.DryRun,
				NonIncremental: flags.NonIncremental,
			})
		}

		res := applier.Execute()
		results = append(results, res)
		if err := res.GetErr(); err != nil && !flags.ContinueOnError {
			return results
		}
	}

	return results
}
