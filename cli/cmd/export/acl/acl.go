// Package acl implements the export acl command and executes the controller.
package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/ctl/export"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/util/str"
)

// Command creates the export acl command.
func Command(cOpts *config.Options) *cobra.Command {
	opts := export.ControllerOptions{}
	var defFormat string

	cmd := &cobra.Command{
		Use:   "acl [options]",
		Short: "Export resource acls to definitions",
		Long: `Export resource acls to definitions (Kafka 0.11.0+).

Exports to stdout by default. Supply the --output-dir option to create definition files.

Manual: https://peter-evans.github.io/kdef`,
		Example: `# export all resource acls to the directory "acls"
kdef export acl --output-dir "acls"

# export all topic acls to stdout
kdef export acl --type topic --quiet

# export all resource acls starting with "myapp"
kdef export acl --match "myapp.*"`,
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		PreRunE: func(_ *cobra.Command, args []string) error {
			opts.DefinitionFormat = opt.ParseDefinitionFormat(defFormat)
			if opts.DefinitionFormat == opt.UnsupportedFormat {
				return fmt.Errorf("\"format\" must be one of %q", strings.Join(opt.DefinitionFormatValidValues, "|"))
			}
			if !str.Contains(opts.ACLResourceType, opt.ACLResourceTypeValidValues) {
				return fmt.Errorf("\"type\" must be one of %q", strings.Join(opt.ACLResourceTypeValidValues, "|"))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			cl, err := config.NewClient(cOpts)
			if err != nil {
				return err
			}

			// TODO: Make constants for the kinds
			ctx := context.Background()
			ctl := export.NewExportController(cl, args, opts, "acl")
			if err := ctl.Execute(ctx); err != nil {
				return err
			}
			return nil
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
	cmd.Flags().StringVarP(&opts.Match, "match", "m", ".*", "regular expression matching resource names to include")
	cmd.Flags().StringVarP(&opts.Exclude, "exclude", "e", ".^", "regular expression matching resource names to exclude")
	cmd.Flags().StringVarP(
		&opts.ACLResourceType,
		"type",
		"t",
		"any",
		fmt.Sprintf("acl resource type [%s]", strings.Join(opt.ACLResourceTypeValidValues, "|")),
	)
	cmd.Flags().BoolVarP(&opts.ACLAutoGroup, "auto-group", "g", true, "combine acls into groups for easier management")

	return cmd
}
