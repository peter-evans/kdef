package topics

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/topics"
)

// Creates the export topics command
func Command(cl *client.Client) *cobra.Command {
	flags := topics.ExporterFlags{}
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
		RunE: func(_ *cobra.Command, args []string) error {

			exporter := topics.NewExporter(cl, flags)
			if _, err := exporter.Execute(); err != nil {
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

	return cmd
}
