// Package export implements the export command.
package export

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/cmd/export/acl"
	"github.com/peter-evans/kdef/cli/cmd/export/broker"
	"github.com/peter-evans/kdef/cli/cmd/export/brokers"
	"github.com/peter-evans/kdef/cli/cmd/export/topic"
	"github.com/peter-evans/kdef/cli/config"
)

func Command(cOpts *config.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export cluster resources to definitions",
		Long:  "Export cluster resources to definitions",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		acl.Command(cOpts),
		broker.Command(cOpts),
		brokers.Command(cOpts),
		topic.Command(cOpts),
	)

	return cmd
}
