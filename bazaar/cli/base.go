package cli

import (
	"github.com/spf13/cobra"
	"io"
)

type baseBazaarCmd struct {
	target string
	user   string
	pass   string
	out    io.Writer
}

func (b *baseBazaarCmd) parseCommonFlags(cmd *cobra.Command) {
	//todo: -k flag for ignore ssl

	cmd.Flags().StringVarP(&b.target, "target", "t", "", "bazaar API url")
	cobra.MarkFlagRequired(cmd.Flags(), "target")
	cmd.Flags().StringVarP(&b.user, "user", "u", "", "bazaar API user")
	cobra.MarkFlagRequired(cmd.Flags(), "user")
	cmd.Flags().StringVarP(&b.pass, "password", "p", "", "bazaar API password")
	cobra.MarkFlagRequired(cmd.Flags(), "password")
}
