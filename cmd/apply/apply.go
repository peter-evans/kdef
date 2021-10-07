package apply

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/ctl/apply"
)

// Creates the apply command
func Command(cl *client.Client) *cobra.Command {
	flags := apply.ApplyControllerFlags{}
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
		PreRunE: func(_ *cobra.Command, args []string) error {
			if flags.ReassAwaitTimeout < 0 {
				return fmt.Errorf("reassignments await timeout seconds must be greater or equal to 0")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if flags.JsonOutput {
				log.Quiet = true
			}
			if flags.ExitCode {
				flags.DryRun = true
			}
			if flags.DryRun {
				log.InfoWithKey("dry-run", "Enabled")
			}

			controller := apply.NewApplyController(cl, args, flags)
			if err := controller.Execute(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&flags.DryRun, "dry-run", "d", false, "validate and review the operation only")
	cmd.Flags().BoolVarP(&flags.ExitCode, "exit-code", "e", false, "implies --dry-run and causes the program to exit with 1 if there are unapplied changes and 0 otherwise")
	cmd.Flags().BoolVar(&flags.JsonOutput, "json-output", false, "implies --quiet and outputs JSON apply results")
	cmd.Flags().BoolVarP(&flags.ContinueOnError, "continue-on-error", "c", false, "applying resource definitions is not interrupted if there are errors")
	cmd.Flags().IntVarP(&flags.ReassAwaitTimeout, "reass-await-timeout", "r", 0, "time in seconds to wait for topic partition reassignments to complete before timing out")

	return cmd
}
