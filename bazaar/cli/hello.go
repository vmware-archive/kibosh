package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"errors"
)

type helloCmd struct {
	message string
	out     io.Writer
}

func NewHelloCmd(out io.Writer) *cobra.Command {
	hc := &helloCmd{out: out}

	cmd := &cobra.Command{
		Use:   "hello MESSAGE",
		Short: "say hello with the given message",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("the hello message is required")
			}
			hc.message = args[0]
			return hc.run()
		},
	}

	return cmd
}

func (hc *helloCmd) run() error {
	fmt.Fprintf(hc.out, "hello '%s'", hc.message)
	return nil
}
