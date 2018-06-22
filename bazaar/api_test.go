// kibosh
//
// Copyright (c) 2017-Present Pivotal Software, Inc. All Rights Reserved.
//
// This program and the accompanying materials are made available under the terms of the under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may
// obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the
// License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing permissions and
// limitations under the License.

package bazaar_test

import (
	"code.cloudfoundry.org/lager"
	"errors"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/repository/repositoryfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	"net/http"
	"net/http/httptest"

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"strings"
)

var _ = Describe("Api", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"

	var repo repositoryfakes.FakeRepository
	var bro broker.PksServiceBroker
	var logger lager.Logger
	var api bazaar.API

	BeforeEach(func() {
		repo = repositoryfakes.FakeRepository{}
		bro = broker.PksServiceBroker{}
		logger = lager.NewLogger("APITest")
		api = bazaar.NewAPI(&repo, logger)

	})

	Context("List charts", func() {

		It("List charts on bazaar", func() {
			spacebearsChart := &helm.MyChart{
				Chartpath: "/foo/bar",
				Plans: map[string]helm.Plan{
					"plan1": {
						Name: "plan1",
					},
					"plan2": {
						Name: "plan2",
					},
				},
				Chart: &hapi_chart.Chart{
					Metadata: &hapi_chart.Metadata{
						Name:        "spacebears",
						Description: "spacebears service and spacebears broker helm chart",
					},
				},
			}
			charts := []*helm.MyChart{spacebearsChart}

			repo.LoadChartsReturns(charts, nil)
			req, err := http.NewRequest("GET", "/charts/", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.ListCharts()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(200))
			rawBody, err := ioutil.ReadAll(recorder.Body)
			Expect(err).To(BeNil())
			body := []map[string]interface{}{}
			err = json.Unmarshal(rawBody, &body)
			Expect(err).To(BeNil())

			Expect(body[0]["name"]).To(Equal("spacebears"))
			Expect(body[0]["chartpath"]).To(Equal("/foo/bar"))
		})

		It("500s on failure", func() {
			repo.LoadChartsReturns(nil, errors.New("something went south"))
			req, err := http.NewRequest("GET", "/charts/", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.ListCharts()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(500))
		})
	})

	Context("Create chart", func() {
		It("passes file to repository", func() {

			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.CreateChart()
			apiHandler.ServeHTTP(recorder, req)

			path := repo.SaveChartArgsForCall(0)
			saved, err := ioutil.ReadFile(path)
			Expect(err).To(BeNil())
			Expect(string(saved)).To(Equal("hello upload"))
		})

		It("save to repo fails", func() {
			repo.SaveChartReturns(errors.New("failed to save charts"))
			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.CreateChart()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(500))
			Expect(repo.SaveChartCallCount()).To(Equal(1))
		})

		It("set correct failure on get request", func() {

			req, err := http.NewRequest("GET", "/charts/create", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.CreateChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(405))
		})

	})

})

func createRequestWithFile() (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("chart", "chart.txt")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, strings.NewReader("hello upload"))
	if err != nil {
		return nil, err
	}
	boundary := writer.Boundary()
	_, err = io.Copy(part, strings.NewReader(fmt.Sprintf("\r\n--%s--\r\n", boundary)))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "/charts/create", body)
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)

	return req, nil

}
