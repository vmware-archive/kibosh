package cli

import (
	"encoding/json"
	"fmt"
	"github.com/cf-platform-eng/kibosh/auth"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/gosuri/uitable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
)

type chartsListCmd struct {
	baseBazaarCmd
}

func NewChartsListCmd(out io.Writer) *cobra.Command {
	cl := &chartsListCmd{}
	cl.out = out

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list charts in repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cl.run()
		},
	}

	cl.baseBazaarCmd.parseCommonFlags(cmd)

	return cmd
}

func (cl *chartsListCmd) run() error {
	client := &http.Client{}
	url := fmt.Sprintf("%s/charts", cl.target)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", auth.BasicAuthorizationHeaderVal(cl.user, cl.pass))

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Non-OK response code from API [%v]", res.Status))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var charts []bazaar.DisplayChart
	json.Unmarshal(body, &charts)
	if err != nil {
		return err
	}

	table := uitable.New()
	table.AddRow("NAME", "VERSION", "PLANS")
	for _, c := range charts {
		table.AddRow(c.Name, c.Version, fmt.Sprintf("%+v", c.Plans))
	}

	cl.out.Write(table.Bytes())
	cl.out.Write([]byte("\n"))

	return nil
}
