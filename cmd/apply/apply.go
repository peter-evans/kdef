package apply

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/config"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/ctl/apply"
)

// Creates the apply command
func Command(cOpts *config.ConfigOptions) *cobra.Command {
	opts := apply.ApplyControllerOptions{}
	var definitionFormat string

	cmd := &cobra.Command{
		Use:   "apply [definitions]",
		Short: "Apply definitions to cluster",
		Long: `Apply definitions to cluster.

broker (Kafka 0.11.0+)
brokers (Kafka 0.11.0+)
topic (Kafka 2.4.0+)`,
		Example: `# apply all definitions in directory "topics" (dry-run)
kdef apply topics/* --dry-run

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
			if opts.JsonOutput {
				log.Quiet = true
			}
			if opts.ExitCode {
				opts.DryRun = true
			}
			if opts.DryRun {
				log.InfoWithKey("dry-run", "Enabled")
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
	cmd.Flags().BoolVar(&opts.JsonOutput, "json-output", false, "implies --quiet and outputs JSON apply results")
	cmd.Flags().BoolVarP(&opts.ContinueOnError, "continue-on-error", "c", false, "applying resource definitions is not interrupted if there are errors")
	cmd.Flags().IntVarP(&opts.ReassAwaitTimeout, "reass-await-timeout", "r", 0, "time in seconds to wait for topic partition reassignments to complete before timing out")

	return cmd
}
