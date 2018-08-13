// kibosh
//
// Copyright (C) 2015-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	"encoding/json"

	"github.com/cf-platform-eng/kibosh/pkg/config"
)

type configParams struct {
	ClusterConfig clusterConfig `json:"clusterConfig"`
}

type clusterConfig struct {
	Server                  string `json:"server"`
	Token                   string `json:"token"`
	CertificatAuthorityData string `json:"certificateAuthorityData"`
}

func isValidClusterConfig(clusterConfig *clusterConfig) bool {
	return len(clusterConfig.Server) > 0 && len(clusterConfig.Token) > 0 && len(clusterConfig.CertificatAuthorityData) > 0
}

func ExtractClusterConfig(params json.RawMessage) (config.ClusterCredentials, bool) {
	var c configParams
	err := json.Unmarshal(params, &c)

	if err == nil && isValidClusterConfig(&c.ClusterConfig) {
		clusterCreds := config.ClusterCredentials{Server: c.ClusterConfig.Server, Token: c.ClusterConfig.Token, CADataRaw: string(c.ClusterConfig.CertificatAuthorityData)}
		clusterCreds.ParseCAData()
		return clusterCreds, true
	}

	return config.ClusterCredentials{}, false
}
