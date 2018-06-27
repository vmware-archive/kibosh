package cli

import (
	"github.com/spf13/cobra"
	"io"
)

type chartsSaveCmd struct {
	baseBazaarCmd
}

func NewChartsSaveCmd(out io.Writer) *cobra.Command {
	cl := &chartsSaveCmd{}
	cl.out = out

	cmd := &cobra.Command{
		Use:   "save",
		Short: "save charts to repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cl.run()
		},
	}

	cl.baseBazaarCmd.parseCommonFlags(cmd)

	return cmd
}

func (cs *chartsSaveCmd) run() error {
	panic("implement")
}
