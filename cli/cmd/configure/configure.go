package configure

import (
	"github.com/peter-evans/kdef/cli/config"
	"github.com/spf13/cobra"
)

// Creates the configure command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "configure",
		Short:                 "Interactive configuration setup",
		Long:                  "A short interactive prompt to guide through creating a configuration file",
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return config.Configure()
		},
	}

	return cmd
}
