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
	"io"
	"strings"

	"github.com/spf13/cobra"
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
