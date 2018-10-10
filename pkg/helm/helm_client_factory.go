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

package helm

import (
	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
)

//go:generate counterfeiter ./ HelmClientFactory
type HelmClientFactory interface {
	HelmClient(cluster k8s.Cluster) MyHelmClient
}

type helmClientFactory struct {
	logger  *logrus.Logger
	tlsConf *config.HelmTLSConfig
}

func (hcf helmClientFactory) HelmClient(cluster k8s.Cluster) MyHelmClient {
	return NewMyHelmClient(cluster, hcf.tlsConf, hcf.logger)
}

func NewHelmClientFactory(tlsConf *config.HelmTLSConfig, logger *logrus.Logger) HelmClientFactory {
	return &helmClientFactory{
		logger:  logger,
		tlsConf: tlsConf,
	}
}
