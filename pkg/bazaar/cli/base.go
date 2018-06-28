package cli

import (
	"github.com/spf13/cobra"
	"io"
	"strings"
)

type baseBazaarCmd struct {
	target string
	user   string
	pass   string
	out    io.Writer
}

func (b *baseBazaarCmd) preRun(cmd *cobra.Command, args []string) {
	if strings.HasSuffix(b.target, "/") {
		b.target = b.target[:len(b.target)-1]
	}
}

func (b *baseBazaarCmd) addCommonFlags(cmd *cobra.Command) {
	//todo: -k flag for ignore ssl

	cmd.Flags().StringVarP(&b.target, "target", "t", "", "bazaar API url (required)")
	cobra.MarkFlagRequired(cmd.Flags(), "target")
	cmd.Flags().StringVarP(&b.user, "user", "u", "", "bazaar API user (required)")
	cobra.MarkFlagRequired(cmd.Flags(), "user")
	cmd.Flags().StringVarP(&b.pass, "password", "p", "", "bazaar API password (required)")
	cobra.MarkFlagRequired(cmd.Flags(), "password")
}
