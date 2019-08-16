// kibosh
//
// Copyright (c) 2017-Present Pivotal Software, Inc. All Rights Reserved.
//
// This program and the accompanying materials are made available under the terms of the under the Apache License,
// Version 2.0 (the "License”); you may not use this file except in compliance with the License. You may
// obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the
// License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/gosuri/uitable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		PreRun: func(cmd *cobra.Command, args []string) {
			cl.preRun(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cl.run()
		},
	}

	cl.baseBazaarCmd.addCommonFlags(cmd)

	return cmd
}

func (cl *chartsListCmd) run() error {
	client := &http.Client{}
	url := fmt.Sprintf("%s/charts", cl.target)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	httphelpers.AddBasicAuthHeader(req, cl.user, cl.pass)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		return errors.New(fmt.Sprintf("Non-OK response code from API [%v]\nMessage from server: %v\n", res.Status, string(body)))
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
