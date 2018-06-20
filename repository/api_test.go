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

package repository_test

import (
	"code.cloudfoundry.org/lager"
	"errors"
	"github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/repository"
	"github.com/cf-platform-eng/kibosh/repository/repositoryfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Api", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"

	var repo repositoryfakes.FakeRepository
	var bro broker.PksServiceBroker
	var logger lager.Logger
	var api repository.API

	BeforeEach(func() {
		repo = repositoryfakes.FakeRepository{}
		bro = broker.PksServiceBroker{}
		logger = lager.NewLogger("APITest")
		api = repository.NewAPI(&bro, &repo, logger)

	})

	It("sets charts on broker", func() {
		spacebearsChart := &helm.MyChart{
			Chart: &hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "spacebears",
					Description: "spacebears service and spacebears broker helm chart",
				},
			},
		}
		charts := []*helm.MyChart{spacebearsChart}

		repo.LoadChartsReturns(charts, nil)
		req, err := http.NewRequest("GET", "/reload_charts", nil)
		Expect(err).To(BeNil())

		recorder := httptest.NewRecorder()

		apiHandler := api.ReloadCharts()
		apiHandler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(200))
		broCharts := bro.GetCharts()
		Expect(len(broCharts)).To(Equal(1))
		Expect(broCharts[spacebearsServiceGUID].Metadata.Name).To(Equal("spacebears"))
	})

	It("500s on failure", func() {
		repo.LoadChartsReturns(nil, errors.New("something went south"))
		req, err := http.NewRequest("GET", "/reload_charts", nil)
		Expect(err).To(BeNil())

		recorder := httptest.NewRecorder()

		apiHandler := api.ReloadCharts()
		apiHandler.ServeHTTP(recorder, req)
		Expect(recorder.Code).To(Equal(500))
	})
})
