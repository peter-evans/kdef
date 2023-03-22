// Package broker implements the export broker command and executes the controller.
package broker

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/export"
	"github.com/peter-evans/kdef/core/model/def"
	"github.com/peter-evans/kdef/core/model/opt"
)

// Command creates the export broker command.
func Command(cOpts *config.Options) *cobra.Command {
	opts := export.ControllerOptions{}
	var defFormat string

	cmd := &cobra.Command{
		Use:   "broker [options]",
		Short: "Export per-broker configuration to definitions",
		Long: `Export per-broker configuration to definitions (Kafka 0.11.0+).

Exports to stdout by default. Supply the --output-dir option to create definition files.

Manual: https://peter-evans.github.io/kdef`,
		Example: `# export broker definitions to the directory "broker"
kdef export broker --output-dir "broker"

# export broker definitions to stdout
kdef export broker --quiet`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		PreRunE: func(_ *cobra.Command, args []string) error {
			opts.DefinitionFormat = opt.ParseDefinitionFormat(defFormat)
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

			ctx := context.Background()
			ctl := export.NewExportController(cl, opts, def.KindBroker)
			return ctl.Execute(ctx)
		},
	}

	cmd.Flags().StringVarP(
		&defFormat,
		"format",
		"f",
		"yaml",
		fmt.Sprintf("resource definition format [%s]", strings.Join(opt.DefinitionFormatValidValues, "|")),
	)
	cmd.Flags().StringVarP(
		&opts.OutputDir,
		"output-dir",
		"o",
		"",
		"output directory path for definition files; non-existent directories will be created",
	)
	cmd.Flags().BoolVarP(&opts.Overwrite, "overwrite", "w", false, "overwrite existing files in output directory")

	return cmd
}
