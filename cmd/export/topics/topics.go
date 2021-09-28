package topics

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/operators/topic"
	"github.com/peter-evans/kdef/ctl/export"
	"github.com/peter-evans/kdef/util/str"
)

// Creates the export topics command
func Command(cl *client.Client) *cobra.Command {
	flags := export.ExportControllerFlags{}
	cmd := &cobra.Command{
		Use:     "topics",
		Aliases: []string{"topic"},
		Short:   "Export cluster topics to YAML definitions",
		Long:    "Export cluster topics to YAML definitions (Kafka 0.11.0+).",
		Example: `# export all topics to the directory "topics"
kdef export topics --output-dir "topics"

# export all topics to stdout
kdef export topics --quiet

# export all topics starting with "myapp"
kdef export topics --match "myapp.*"`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		PreRunE: func(_ *cobra.Command, args []string) error {
			if !str.Contains(flags.Assignments, topic.AssignmentsValidValues) {
				return fmt.Errorf("flag \"assignments\" must be one of %q", strings.Join(topic.AssignmentsValidValues, "|"))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			controller := export.NewExportController(cl, args, flags, "topic")
			if err := controller.Execute(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.OutputDir, "output-dir", "o", "", "output directory (must exist)")
	cmd.Flags().BoolVar(&flags.Overwrite, "overwrite", false, "overwrite existing files in output directory")
	cmd.Flags().StringVarP(&flags.Match, "match", "m", ".*", "regular expression matching topic names to include")
	cmd.Flags().StringVarP(&flags.Exclude, "exclude", "e", ".^", "regular expression matching topic names to exclude")
	cmd.Flags().BoolVarP(&flags.IncludeInternal, "include-internal", "i", false, "include internal topics")
	cmd.Flags().StringVar(
		&flags.Assignments,
		"assignments",
		"none",
		fmt.Sprintf("partition assignments to include in topic definitions [%s]", strings.Join(topic.AssignmentsValidValues, "|")),
	)

	return cmd
}
