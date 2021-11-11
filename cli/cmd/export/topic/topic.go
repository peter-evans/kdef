package topic

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/export"
	"github.com/peter-evans/kdef/core/model/opt"
)

// Creates the export topic command
func Command(cOpts *config.Options) *cobra.Command {
	opts := export.ControllerOptions{}
	var definitionFormat string
	var assignments string

	cmd := &cobra.Command{
		Use:   "topic",
		Short: "Export topics to definitions",
		Long:  "Export topics to definitions (Kafka 0.11.0+).",
		Example: `# export all topics to the directory "topics"
kdef export topic --output-dir "topics"

# export all topics to stdout
kdef export topic --quiet

# export all topics starting with "myapp"
kdef export topic --match "myapp.*"`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		PreRunE: func(_ *cobra.Command, args []string) error {
			opts.DefinitionFormat = opt.ParseDefinitionFormat(definitionFormat)
			if opts.DefinitionFormat == opt.UnsupportedFormat {
				return fmt.Errorf("\"format\" must be one of %q", strings.Join(opt.DefinitionFormatValidValues, "|"))
			}
			opts.TopicAssignments = opt.ParseAssignments(assignments)
			if opts.TopicAssignments == opt.UnsupportedAssignments {
				return fmt.Errorf("\"assignments\" must be one of %q", strings.Join(opt.AssignmentsValidValues, "|"))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			cl, err := config.NewClient(cOpts)
			if err != nil {
				return err
			}

			controller := export.NewExportController(cl, args, opts, "topic")
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
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "o", "", "output directory (must exist)")
	cmd.Flags().BoolVar(&opts.Overwrite, "overwrite", false, "overwrite existing files in output directory")
	cmd.Flags().StringVarP(&opts.Match, "match", "m", ".*", "regular expression matching topic names to include")
	cmd.Flags().StringVarP(&opts.Exclude, "exclude", "e", ".^", "regular expression matching topic names to exclude")
	cmd.Flags().BoolVarP(&opts.TopicIncludeInternal, "include-internal", "i", false, "include internal topics")
	cmd.Flags().StringVar(
		&assignments,
		"assignments",
		"none",
		fmt.Sprintf("partition assignments to include in topic definitions [%s]", strings.Join(opt.AssignmentsValidValues, "|")),
	)

	return cmd
}
