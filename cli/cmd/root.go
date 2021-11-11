package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/peter-evans/kdef/cli/cmd/apply"
	"github.com/peter-evans/kdef/cli/cmd/configure"
	"github.com/peter-evans/kdef/cli/cmd/export"
	"github.com/peter-evans/kdef/cli/config"
	"github.com/peter-evans/kdef/cli/log"
)

// Execute the root command
func Execute(version string) {
	if err := rootCmd(version).Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

// Creates the root command
func rootCmd(version string) *cobra.Command {
	var noColor bool
	var quiet bool
	var verbose bool

	cOpts := &config.Options{}

	cmd := &cobra.Command{
		Use:   "kdef",
		Short: "Declarative resource management for Kafka",
		Long: `Declarative resource management for Kafka.

kdef aims to provide an easy way to manage resources in a Kafka cluster
by having them defined explicitly in YAML. Changes to YAML resource
definitions can be reviewed and applied to a cluster.

kdef was designed to support being run in a CI-CD environment, allowing
teams to manage Kafka resource definitions in source control with pull
requests (GitOps).

Create a configuration file for your cluster:
kdef configure

For usage documentation visit http://github.com/peter-evans/kdef`,
		Args: cobra.NoArgs,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			color.NoColor = noColor
			log.Quiet = quiet
			if verbose {
				log.Verbose = verbose
			}
		},
		Version: version,
	}

	cmd.AddCommand(
		configure.Command(),
		apply.Command(cOpts),
		export.Command(cOpts),
	)

	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable coloured output")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "enable quiet mode (output errors only)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug output")
	cmd.PersistentFlags().StringVar(&cOpts.ConfigPath, "config-path", config.DefaultConfigPath(), "path to configuration file")
	cmd.PersistentFlags().StringArrayVarP(&cOpts.ConfigOpts, "config-opt", "X", nil, "flag provided config option (e.g. \"timeoutMs=6000\")")

	return cmd
}
