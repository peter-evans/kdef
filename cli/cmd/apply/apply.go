package apply

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/apply"
	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/model/opt"
)

// Creates the apply command
func Command(cOpts *config.Options) *cobra.Command {
	opts := apply.ControllerOptions{}
	var definitionFormat string

	cmd := &cobra.Command{
		Use:   "apply [definitions]",
		Short: "Apply definitions to cluster",
		Long: `Apply definitions to cluster.

Accepts one or more glob patterns matching the paths of definitions to apply.
Matching directories are ignored.

acl (Kafka 0.11.0+)
broker (Kafka 0.11.0+)
brokers (Kafka 0.11.0+)
topic (Kafka 2.4.0+)`,
		Example: `# apply all definitions in directory "topics" (dry-run)
kdef apply "topics/*.yml" --dry-run

# apply definitions in all directories under "resources" (dry-run)
kdef apply "resources/**/*.yml" --dry-run

# apply a topic definition from stdin (dry-run)
cat topics/my_topic.yml | kdef apply - --dry-run`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.MinimumNArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			opts.DefinitionFormat = opt.ParseDefinitionFormat(definitionFormat)
			if opts.DefinitionFormat == opt.UnsupportedFormat {
				return fmt.Errorf("\"format\" must be one of %q", strings.Join(opt.DefinitionFormatValidValues, "|"))
			}
			if opts.ReassAwaitTimeout < 0 {
				return fmt.Errorf("\"reass-await-timeout\" must be greater or equal to 0")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if opts.JSONOutput {
				log.Quiet = true
			}
			if opts.ExitCode {
				opts.DryRun = true
			}
			if opts.DryRun {
				log.InfoWithKeyf("dry-run", "Enabled")
			}

			cl, err := config.NewClient(cOpts)
			if err != nil {
				return err
			}

			controller := apply.NewApplyController(cl, args, opts)
			if err := controller.Execute(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(
		&definitionFormat,
		"format",
		"f",
		"yaml",
		fmt.Sprintf("resource definition format [%s]", strings.Join(opt.DefinitionFormatValidValues, "|")),
	)
	cmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "validate and review the operation only")
	cmd.Flags().BoolVarP(&opts.ExitCode, "exit-code", "e", false, "implies --dry-run and causes the program to exit with 1 if there are unapplied changes and 0 otherwise")
	cmd.Flags().BoolVar(&opts.JSONOutput, "json-output", false, "implies --quiet and outputs JSON apply results")
	cmd.Flags().BoolVarP(&opts.ContinueOnError, "continue-on-error", "c", false, "applying resource definitions is not interrupted if there are errors")
	cmd.Flags().IntVarP(&opts.ReassAwaitTimeout, "reass-await-timeout", "r", 0, "time in seconds to wait for topic partition reassignments to complete before timing out")

	return cmd
}