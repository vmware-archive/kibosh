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

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/broker"
)

type API interface {
	ReloadCharts() http.Handler
}

type api struct {
	broker     *broker.PksServiceBroker
	repository Repository
	logger     lager.Logger
}

func NewAPI(b *broker.PksServiceBroker, r Repository, l lager.Logger) API {
	return &api{
		broker:     b,
		repository: r,
		logger:     l,
	}

}

func (api *api) ReloadCharts() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		charts, err := api.repository.LoadCharts()
		if err != nil {
			api.logger.Error("Unable to load charts", err)
			w.WriteHeader(500)
			w.Write([]byte("Unable to load charts"))
		} else {
			api.broker.SetCharts(charts)
			w.Write([]byte("Reloaded charts successfully"))
		}
	})
}
