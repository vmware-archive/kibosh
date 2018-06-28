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

package bazaar

import (
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/kelseyhightower/envconfig"
)

type bazaarConfig struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`

	Port         int    `envconfig:"PORT" default:"8081"`
	HelmChartDir string `envconfig:"HELM_CHART_DIR" default:"charts"`

	RegistryConfig *config.RegistryConfig
	KiboshConfig   *KiboshConfig
}

type KiboshConfig struct {
	Server string `envconfig:"KIBOSH_SERVER" required:"true"`
	User   string `envconfig:"KIBOSH_USER_NAME" required:"true"`
	Pass   string `envconfig:"KIBOSH_USER_PASSWORD" required:"true"`
}

func ParseConfig() (*bazaarConfig, error) {
	c := &bazaarConfig{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
