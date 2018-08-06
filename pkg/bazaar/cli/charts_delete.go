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
		body, _ := ioutil.ReadAll(res.Body)
		return errors.New(fmt.Sprintf("Non-OK response code from API [%v]\nMessage from server: %v\n", res.Status, string(body)))
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
