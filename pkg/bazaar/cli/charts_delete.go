package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"net/http"
)

type chartsDeleteCmd struct {
	baseBazaarCmd
	name string
}

func NewChartsDeleteCmd(out io.Writer) *cobra.Command {
	cd := &chartsDeleteCmd{}
	cd.out = out

	cmd := &cobra.Command{
		Use:   "delete CHART-NAME",
		Short: "delete chart from repository",
		PreRun: func(cmd *cobra.Command, args []string) {
			cd.preRun(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("missing chart name")
			}
			cd.name = args[0]
			return cd.run()
		},
	}

	cd.baseBazaarCmd.addCommonFlags(cmd)

	return cmd
}

func (cd *chartsDeleteCmd) run() error {
	client := &http.Client{}
	url := fmt.Sprintf("%s/charts/%s", cd.target, cd.name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	httphelpers.AddBasicAuthHeader(req, cd.user, cd.pass)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Non-OK response code from API [%v]", res.Status))
	}

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	responseJSON := bazaar.DisplayResponse{}
	err = json.Unmarshal(responseBody, &responseJSON)
	if err != nil {
		return err
	}

	cd.out.Write([]byte(fmt.Sprintf("Message from server: %s\n", responseJSON.Message)))

	return nil
}
