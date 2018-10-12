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

package repository

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/broker"
	"github.com/cf-platform-eng/kibosh/pkg/cf"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pkg/errors"
	"strings"
)

type API interface {
	ReloadCharts() http.Handler
}

type api struct {
	broker     *broker.PksServiceBroker
	repository Repository
	cfClient   cf.Client
	conf       *config.Config
	logger     *logrus.Logger
}

func NewAPI(b *broker.PksServiceBroker, r Repository, c cf.Client, conf *config.Config, l *logrus.Logger) API {
	return &api{
		broker:     b,
		repository: r,
		cfClient:   c,
		conf:       conf,
		logger:     l,
	}

}

func (api *api) ReloadCharts() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		charts, err := api.repository.LoadCharts()
		if err != nil {
			api.logger.WithError(err).Error("Unable to load charts")
			w.WriteHeader(500)
			w.Write([]byte("Unable to load charts"))
			return
		} else {
			api.broker.SetCharts(charts)

			if api.cfClient != nil {
				err = api.refreshCloudFoundry(charts)
				if err != nil {
					w.WriteHeader(500)
					w.Write([]byte(err.Error()))
					return
				}
			}
		}

		w.Write([]byte("Reloaded charts successfully"))
	})
}

func (api *api) refreshCloudFoundry(charts []*helm.MyChart) error {
	bro, err := api.cfClient.GetServiceBrokerByName(api.conf.CFClientConfig.BrokerName)

	if err == nil {
		_, err = api.cfClient.UpdateServiceBroker(bro.Guid, cfclient.UpdateServiceBrokerRequest{
			BrokerURL: api.conf.CFClientConfig.BrokerURL,
			Username:  api.conf.AdminUsername,
			Password:  api.conf.AdminPassword,
			Name:      api.conf.CFClientConfig.BrokerName,
		})

		if err != nil {
			api.logger.WithError(err).Error("Reloaded charts, but unable to update the broker")
			return errors.New("Reloaded charts, but unable to update the broker")
		}
	} else if strings.Contains(err.Error(), "Unable to find service broker") {
		_, err = api.cfClient.CreateServiceBroker(cfclient.CreateServiceBrokerRequest{
			BrokerURL: api.conf.CFClientConfig.BrokerURL,
			Username:  api.conf.AdminUsername,
			Password:  api.conf.AdminPassword,
			Name:      api.conf.CFClientConfig.BrokerName,
		})

		if err != nil {
			api.logger.WithError(err).Error("Reloaded charts, but unable to register broker")
			return errors.New("Reloaded charts, but unable to register broker")
		}
	} else {
		return errors.New("Reloaded charts, but failed talking to CF")
	}

	return nil
}
