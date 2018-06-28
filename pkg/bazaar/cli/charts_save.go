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

type chartsSaveCmd struct {
	baseBazaarCmd
	path string
}

func NewChartsSaveCmd(out io.Writer) *cobra.Command {
	cs := &chartsSaveCmd{}
	cs.out = out

	cmd := &cobra.Command{
		Use:   "save PATH-TO-CHART.tgz",
		Short: "save charts to repository",
		PreRun: func(cmd *cobra.Command, args []string) {
			cs.preRun(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("missing tar file")
			}
			cs.path = args[0]
			return cs.run()
		},
	}

	cs.baseBazaarCmd.addCommonFlags(cmd)

	return cmd
}

func (cs *chartsSaveCmd) run() error {
	url := cs.target + "/charts"
	req, err := httphelpers.CreateFormRequest(url, "chart", cs.path)
	if err != nil {
		return err
	}
	httphelpers.AddBasicAuthHeader(req, cs.user, cs.pass)

	client := &http.Client{}
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

	cs.out.Write([]byte(fmt.Sprintf("Message from server: %s", responseJSON.Message)))
	cs.out.Write([]byte("\n"))

	return nil
}
