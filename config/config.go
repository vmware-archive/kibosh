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

package config

import (
	"github.com/kelseyhightower/envconfig"

	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ClusterCredentials struct {
	CADataRaw string `envconfig:"CA_DATA"`
	CAData    []byte
	Server    string `envconfig:"SERVER"`
	Token     string `envconfig:"TOKEN"`
}

type RegistryConfig struct {
	Server string `envconfig:"REG_SERVER"`
	User   string `envconfig:"REG_USER"`
	Pass   string `envconfig:"REG_PASS"`
	Email  string `envconfig:"REG_EMAIL"`
}

type config struct {
	AdminUsername string `envconfig:"SECURITY_USER_NAME" required:"true"`
	AdminPassword string `envconfig:"SECURITY_USER_PASSWORD" required:"true"`
	ServiceID     string `envconfig:"SERVICE_ID" required:"true"`
	ServiceName   string `envconfig:"SERVICE_NAME" required:"true"`

	Port         int    `envconfig:"PORT" default:"8080"`
	HelmChartDir string `envconfig:"HELM_CHART_DIR" default:"charts"`

	ClusterCredentials *ClusterCredentials
	RegistryConfig     *RegistryConfig
}

func (r RegistryConfig) HasRegistryConfig() bool {
	return r.Server != ""
}

func (r RegistryConfig) GetDockerConfigJson() ([]byte, error) {
	if r.Server == "" || r.Email == "" || r.Pass == "" || r.User == "" {
		return nil, errors.New("environment didn't have a proper registry config")
	}

	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			r.Server: map[string]interface{}{
				"username": r.User,
				"password": r.Pass,
				"email":    r.Email,
			},
		},
	}
	dockerConfigJson, err := json.Marshal(dockerConfig)
	if err != nil {
		return nil, err
	}

	return dockerConfigJson, nil
}

func Parse() (*config, error) {
	c := &config{}
	err := envconfig.Process("", c)
	if err != nil {
		return nil, err
	}

	err = c.ClusterCredentials.parseCAData()
	if err != nil {
		return nil, err
	}

	c.ServiceName = strings.Replace(c.ServiceName, "_", "-", -1)
	match, err := regexp.MatchString(`^[0-9a-z.\-]+$`, c.ServiceName)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, errors.New(fmt.Sprintf("Name [%s] contains invalid characters", c.ServiceName))
	}

	return c, nil
}

func (c *ClusterCredentials) parseCAData() error {
	c.CADataRaw = strings.TrimSpace(c.CADataRaw)
	if strings.Index(c.CADataRaw, "-----BEGIN CERTIFICATE-----") == 0 {
		c.CAData = []byte(c.CADataRaw)
	} else {
		data, err := base64.StdEncoding.DecodeString(c.CADataRaw)
		if err != nil {
			return err
		} else {
			c.CAData = data
		}
	}

	return nil
}
