// Package configure implements the configure command.
package configure

import (
	"github.com/peter-evans/kdef/cli/config"
	"github.com/spf13/cobra"
)

// Command creates the configure command.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "configure",
		Short:                 "Interactive configuration setup",
		Long:                  "A short interactive prompt to guide through creating a configuration file.",
		SilenceUsage:          true,
		SilenceErrors:         true,
		DisableFlagsInUseLine: true,
		Args:                  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return config.Configure()
		},
	}

	return cmd
}
