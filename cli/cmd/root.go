// Package cmd implements the root command.
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

// Execute executes the root command.
func Execute(version string) {
	if err := rootCmd(version).Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

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
by having them defined explicitly in a human-readable format. Changes to
resource definitions can be reviewed like code and applied to a cluster.

kdef was designed to support being run in a CI-CD environment, allowing
teams to manage Kafka resource definitions in source control with pull
requests (GitOps).

Create a configuration file for your cluster:
kdef configure

Documentation: https://peter-evans.github.io/kdef`,
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

	cmd.SetUsageTemplate(usageTmpl)

	cmd.AddCommand(
		configure.Command(),
		apply.Command(cOpts),
		export.Command(cOpts),
	)

	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "enable quiet mode (output errors only)")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug output")
	cmd.PersistentFlags().StringVarP(&cOpts.ConfigPath, "config-path", "p", config.DefaultConfigPath(), "path to configuration file")
	cmd.PersistentFlags().StringArrayVarP(&cOpts.ConfigOpts, "config-opt", "X", nil, "option provided configuration (e.g. -X timeoutMs=6000)")

	return cmd
}

const usageTmpl = `USAGE:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <command>{{end}}{{if gt (len .Aliases) 0}}

ALIASES:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

EXAMPLES:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

SUBCOMMANDS:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

OPTIONS:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

GLOBAL OPTIONS:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} <command> --help" for more information about a command.{{end}}
`
