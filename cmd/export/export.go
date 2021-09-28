package export

import (
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/cmd/export/brokers"
	"github.com/peter-evans/kdef/cmd/export/topics"
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
		brokers.Command(cl),
		topics.Command(cl),
	)

	return cmd
}
