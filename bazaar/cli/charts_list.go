package cli

import (
	"github.com/spf13/cobra"
	"io"
)

type chartsListCmd struct {
	message string
	target string
	out     io.Writer
}

func NewChartsListCmd(out io.Writer) *cobra.Command {
	cl := &chartsListCmd{out: out}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list charts in repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cl.run()
		},
	}

	cmd.Flags().StringVarP(&cl.target, "target", "t", "", "the bazaar API")

	return cmd
}

func (cl *chartsListCmd) run() error {
	return nil
}