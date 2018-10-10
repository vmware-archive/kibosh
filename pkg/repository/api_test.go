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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/broker"
	"github.com/cf-platform-eng/kibosh/pkg/cf/cffakes"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/cf-platform-eng/kibosh/pkg/repository/repositoryfakes"
	"github.com/cloudfoundry-community/go-cfclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

var _ = Describe("Api", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"

	var spacebearsChart *helm.MyChart
	var repo repositoryfakes.FakeRepository
	var bro broker.PksServiceBroker
	var cfClient cffakes.FakeClient
	var conf *config.Config
	var logger *logrus.Logger
	var api repository.API

	BeforeEach(func() {
		spacebearsChart = &helm.MyChart{
			Chart: &hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "spacebears",
					Description: "spacebears service and spacebears broker helm chart",
				},
			},
		}
		repo = repositoryfakes.FakeRepository{}
		bro = broker.PksServiceBroker{}
		cfClient = cffakes.FakeClient{}
		conf = &config.Config{
			AdminUsername: "bob_the_broker",
			AdminPassword: "monkey123",
			CFClientConfig: &config.CFClientConfig{
				BrokerName: "bazaaaar",
				BrokerURL:  "https://broker.exmaple.com",
			},
		}
		logger = logrus.New()
		api = repository.NewAPI(&bro, &repo, &cfClient, conf, logger)
	})

	It("sets charts on broker", func() {
		charts := []*helm.MyChart{spacebearsChart}

		repo.LoadChartsReturns(charts, nil)
		req, err := http.NewRequest("GET", "/reload_charts", nil)
		Expect(err).To(BeNil())

		recorder := httptest.NewRecorder()

		apiHandler := api.ReloadCharts()
		apiHandler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(200))
		broCharts := bro.GetChartsMap()
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

	Context("reload self in cf", func() {
		It("calls cf to create broker in reload charts", func() {
			cfClient.GetServiceBrokerByNameReturns(
				cfclient.ServiceBroker{}, errors.New("Unable to find service broker, yo"),
			)
			req, err := http.NewRequest("GET", "/reload_charts", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.ReloadCharts()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(200))

			Expect(cfClient.CreateServiceBrokerCallCount()).To(Equal(1))

			request := cfClient.CreateServiceBrokerArgsForCall(0)
			Expect(request.Username).To(Equal("bob_the_broker"))
			Expect(request.Password).To(Equal("monkey123"))
			Expect(request.BrokerURL).To(Equal("https://broker.exmaple.com"))
			Expect(request.Name).To(Equal("bazaaaar"))
		})

		It("calls cf to update broker in reload charts", func() {
			cfClient.GetServiceBrokerByNameReturns(
				cfclient.ServiceBroker{Guid: "myguid"}, nil,
			)
			req, err := http.NewRequest("GET", "/reload_charts", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.ReloadCharts()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(200))

			Expect(cfClient.UpdateServiceBrokerCallCount()).To(Equal(1))

			guid, request := cfClient.UpdateServiceBrokerArgsForCall(0)
			Expect(guid).To(Equal("myguid"))
			Expect(request.Username).To(Equal("bob_the_broker"))
			Expect(request.Password).To(Equal("monkey123"))
			Expect(request.BrokerURL).To(Equal("https://broker.exmaple.com"))
			Expect(request.Name).To(Equal("bazaaaar"))
		})

		It("calls cf to update broker in reload charts failed", func() {
			cfClient.GetServiceBrokerByNameReturns(
				cfclient.ServiceBroker{}, errors.New("Danger! No! Bad!"),
			)
			req, err := http.NewRequest("GET", "/reload_charts", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.ReloadCharts()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(500))
			Expect(cfClient.UpdateServiceBrokerCallCount()).To(Equal(0))
			Expect(cfClient.CreateServiceBrokerCallCount()).To(Equal(0))

			body, _ := ioutil.ReadAll(recorder.Body)
			Expect(body).To(ContainSubstring("failed talking to CF"))
		})

		It("is cool with nil cf conf", func() {
			req, err := http.NewRequest("GET", "/reload_charts", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			api = repository.NewAPI(&bro, &repo, nil, conf, logger)

			apiHandler := api.ReloadCharts()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(200))

			Expect(cfClient.CreateServiceBrokerCallCount()).To(Equal(0))
		})
	})
})
