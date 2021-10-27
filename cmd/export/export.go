package export

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cmd/export/acl"
	"github.com/peter-evans/kdef/cmd/export/broker"
	"github.com/peter-evans/kdef/cmd/export/brokers"
	"github.com/peter-evans/kdef/cmd/export/topic"
	"github.com/peter-evans/kdef/config"
)

// Creates the export command
func Command(cOpts *config.ConfigOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export cluster resources to YAML definitions",
		Long:  "Export cluster resources to YAML definitions",
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
