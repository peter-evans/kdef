package apply

import (
	"errors"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/in"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/def"
	"github.com/peter-evans/kdef/core/topics"
)

// Creates the apply command
func Command(cl *client.Client) *cobra.Command {
	flags := topics.ApplierFlags{}
	cmd := &cobra.Command{
		Use:   "apply [definitions]",
		Short: "Apply YAML definitions to cluster",
		Long:  "Apply YAML definitions to cluster (Kafka 0.11.0+).",
		Example: `# apply all definitions in directory "topics" (dry-run)
kdef apply topics/* --dry-run

# apply a topic definition from stdin (dry-run)
cat topics/my_topic.yml | kdef apply - --dry-run`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if flags.ExitCode {
				flags.DryRun = true
			}
			if flags.DryRun {
				log.InfoWithKey("dry-run", "Enabled")
			}

			for _, arg := range args {
				if arg == "-" {
					log.Info("Reading definition(s) from stdin")
					yamlDocs, err := in.StdinToYamlDocs()
					if err != nil {
						return err
					}

					if err := applyYamlDocs(cl, flags, yamlDocs); err != nil {
						return err
					}

					break
				} else {
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

						if err := applyYamlDocs(cl, flags, yamlDocs); err != nil {
							return err
						}
					}

					if matchCount == 0 {
						return errors.New("no definition files found")
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flags.DeleteMissingConfigs, "delete-missing-configs", false, `allow deletion of dynamic config keys not present in the definition
CAUTION: this will permanently delete configuration keys — always confirm with --dry-run`)
	cmd.Flags().BoolVarP(&flags.DryRun, "dry-run", "d", false, "validate and review the operation only")
	cmd.Flags().BoolVarP(&flags.ExitCode, "exit-code", "e", false, "implies --dry-run and causes the program to exit with 1 if there are unapplied changes and 0 otherwise.")
	cmd.Flags().BoolVarP(&flags.NonIncremental, "non-inc", "n", false, `use the non-incremental alter configs request method
required by clusters that do not support incremental alter configs (Kafka 0.11.0 to 2.2.0)`)

	return cmd
}

// Apply YAML resource documents using an applier
func applyYamlDocs(cl *client.Client, flags topics.ApplierFlags, yamlDocs []string) error {
	resourceDefs, err := def.GetResourceDefinitions(yamlDocs)
	if err != nil {
		return err
	}

	for i, resourceDef := range resourceDefs {
		switch resourceDef.Kind {
		case "topic":
			applier := topics.NewApplier(cl, yamlDocs[i], flags)
			if err := applier.Execute(); err != nil {
				return err
			}
		}
	}

	return nil
}
