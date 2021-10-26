package acl

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/config"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/util/str"
	"github.com/peter-evans/kdef/ctl/export"
)

// Creates the export acl command
func Command(cOpts *config.ConfigOptions) *cobra.Command {
	opts := export.ExportControllerOptions{}
	var definitionFormat string

	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Export resource acls to definitions",
		Long:  "Export resource acls to definitions (Kafka 0.11.0+).",
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
			opts.DefinitionFormat = opt.ParseDefinitionFormat(definitionFormat)
			if opts.DefinitionFormat == opt.UnsupportedFormat {
				return fmt.Errorf("\"format\" must be one of %q", strings.Join(opt.DefinitionFormatValidValues, "|"))
			}
			if !str.Contains(opts.AclResourceType, opt.AclResourceTypeValidValues) {
				return fmt.Errorf("\"type\" must be one of %q", strings.Join(opt.AclResourceTypeValidValues, "|"))
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			cl, err := config.NewClient(cOpts)
			if err != nil {
				return err
			}

			// TODO: Make constants for the kinds
			controller := export.NewExportController(cl, args, opts, "acl")
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
	cmd.Flags().StringVar(
		&opts.AclResourceType,
		"type",
		"any",
		fmt.Sprintf("acl resource type [%s]", strings.Join(opt.AclResourceTypeValidValues, "|")),
	)
	cmd.Flags().BoolVarP(&opts.AclAutoGroup, "auto-group", "g", true, "combine acls into groups for easier management")

	return cmd
}
