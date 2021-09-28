package brokers

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/ctl/export"
)

// Creates the export brokers command
func Command(cl *client.Client) *cobra.Command {
	flags := export.ExportControllerFlags{}
	cmd := &cobra.Command{
		Use:   "brokers",
		Short: "Export brokers configuration to a YAML definition",
		// TODO: check version
		Long: "Export brokers configuration to a YAML definition (Kafka 0.11.0+).",
		Example: `# export brokers definition to the directory "brokers"
kdef export brokers --output-dir "brokers"

# export brokers definition to stdout
kdef export brokers --quiet`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, args []string) error {
			controller := export.NewExportController(cl, args, flags, "brokers")
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
