package export

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/cmd/export/broker"
	"github.com/peter-evans/kdef/cmd/export/brokers"
	"github.com/peter-evans/kdef/cmd/export/topic"
)

// Creates the export command
func Command(cl *client.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export cluster resources to YAML definitions",
		Long:  "Export cluster resources to YAML definitions",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		broker.Command(cl),
		brokers.Command(cl),
		topic.Command(cl),
	)

	return cmd
}
