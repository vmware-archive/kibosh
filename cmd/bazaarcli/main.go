package main

import (
	"github.com/cf-platform-eng/kibosh/pkg/bazaar/cli"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	cmd := newRootCmd(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd(args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bazaar",
		Short: "The Kibosh chart manager.",
	}

	flags := cmd.PersistentFlags()

	out := cmd.OutOrStdout()

	cmd.AddCommand(
		cli.NewChartsListCmd(out),
		cli.NewChartsSaveCmd(out),
		cli.NewChartsDeleteCmd(out),
	)

	flags.Parse(args)

	return cmd
}
