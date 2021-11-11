package broker

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/export"
	"github.com/peter-evans/kdef/core/model/opt"
)

// Creates the export broker command
func Command(cOpts *config.Options) *cobra.Command {
	opts := export.ControllerOptions{}
	var definitionFormat string

	cmd := &cobra.Command{
		Use:   "broker",
		Short: "Export per-broker configuration to definitions",
		Long:  "Export per-broker configuration to definitions (Kafka 0.11.0+).",
		Example: `# export broker definitions to the directory "broker"
kdef export broker --output-dir "broker"

# export broker definitions to stdout
kdef export broker --quiet`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		PreRunE: func(_ *cobra.Command, args []string) error {
			opts.DefinitionFormat = opt.ParseDefinitionFormat(definitionFormat)
			if opts.DefinitionFormat == opt.UnsupportedFormat {
				return fmt.Errorf("\"format\" must be one of %q", strings.Join(opt.DefinitionFormatValidValues, "|"))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			cl, err := config.NewClient(cOpts)
			if err != nil {
				return err
			}

			controller := export.NewExportController(cl, args, opts, "broker")
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

	return cmd
}