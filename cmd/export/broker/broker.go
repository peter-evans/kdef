package broker

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/ctl/export"
)

// Creates the export broker command
func Command(cl *client.Client) *cobra.Command {
	flags := export.ExportControllerFlags{}
	cmd := &cobra.Command{
		Use:   "broker",
		Short: "Export per-broker configuration to YAML definitions",
		Long:  "Export per-broker configuration to YAML definitions (Kafka 0.11.0+).",
		Example: `# export broker definitions to the directory "broker"
kdef export broker --output-dir "broker"

# export broker definitions to stdout
kdef export broker --quiet`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, args []string) error {
			controller := export.NewExportController(cl, args, flags, "broker")
			if err := controller.Execute(); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.OutputDir, "output-dir", "o", "", "output directory (must exist)")
	cmd.Flags().BoolVar(&flags.Overwrite, "overwrite", false, "overwrite existing files in output directory")

	return cmd
}
