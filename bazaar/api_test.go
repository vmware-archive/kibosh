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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/repository/repositoryfakes"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

var _ = Describe("Api", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"

	var repo repositoryfakes.FakeRepository
	var logger lager.Logger
	var api bazaar.API
	var kiboshConfig *bazaar.KiboshConfig

	var kiboshAPIRequest *http.Request
	var kiboshAPITestServer *httptest.Server

	BeforeEach(func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			kiboshAPIRequest = r
		})
		kiboshAPITestServer = httptest.NewServer(handler)

		repo = repositoryfakes.FakeRepository{}
		logger = lager.NewLogger("APITest")
		kiboshConfig = &bazaar.KiboshConfig{
			Server: kiboshAPITestServer.URL,
			User:   "bob",
			Pass:   "monkey123",
		}
		api = bazaar.NewAPI(&repo, kiboshConfig, logger)
	})

	AfterEach(func() {
		kiboshAPITestServer.Close()
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

	Context("Save chart", func() {
		It("passes file to repository", func() {
			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.SaveChart()
			apiHandler.ServeHTTP(recorder, req)

			path := repo.SaveChartArgsForCall(0)
			saved, err := ioutil.ReadFile(path)
			Expect(err).To(BeNil())
			Expect(string(saved)).To(Equal("hello upload"))
		})

		It("calls kibosh reload charts", func() {
			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.SaveChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(kiboshAPIRequest.URL.Path).To(Equal("/reload_charts"))
			Expect(kiboshAPIRequest.Header.Get("Authorization")).To(
				Equal("Basic Ym9iOm1vbmtleTEyMw=="),
			)
		})

		It("writes error on reload chart failure", func() {
			kiboshAPITestServer.Close()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
			})
			kiboshAPITestServer = httptest.NewServer(handler)
			kiboshConfig = &bazaar.KiboshConfig{
				Server: kiboshAPITestServer.URL,
				User:   "bob",
				Pass:   "monkey123",
			}

			api = bazaar.NewAPI(&repo, kiboshConfig, logger)

			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.SaveChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(500))
		})

		It("writes error when save to repo fails", func() {
			repo.SaveChartReturns(errors.New("failed to save charts"))
			req, err := createRequestWithFile()
			Expect(err).To(BeNil())
			recorder := httptest.NewRecorder()

			apiHandler := api.SaveChart()
			apiHandler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(500))
			Expect(repo.SaveChartCallCount()).To(Equal(1))
		})

		It("set correct failure on non-POST request", func() {
			req, err := http.NewRequest("GET", "/charts/create", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.SaveChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(405))
		})
	})

	Context("Delete chart", func() {
		It("set correct failure on non-DELETE request", func() {
			req, err := http.NewRequest("GET", "/charts/mysql/", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.DeleteChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(405))
		})

		It("url parsing fails", func() {
			req, err := http.NewRequest("DELETE", "/charts", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.DeleteChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(500))
		})
		It("delete chart fails in repo", func() {
			repo.DeleteChartReturns(errors.New("Nope. Keeping the chart."))

			req, err := http.NewRequest("DELETE", "/charts/mysql", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.DeleteChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(500))
			Expect(recorder.Body).To(ContainSubstring("delete"))
		})

		It("delete chart fails in updating kibosh", func() {

			kiboshAPITestServer.Close()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(505)
			})
			kiboshAPITestServer = httptest.NewServer(handler)

			api = bazaar.NewAPI(&repo, kiboshConfig, logger)

			req, err := http.NewRequest("DELETE", "/charts/mysql", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.DeleteChart()
			apiHandler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(500))
			Expect(recorder.Body).To(ContainSubstring("Kibosh"))
		})

		It("successfully deleted chart", func() {

			req, err := http.NewRequest("DELETE", "/charts/mysql", nil)
			Expect(err).To(BeNil())

			recorder := httptest.NewRecorder()

			apiHandler := api.DeleteChart()
			apiHandler.ServeHTTP(recorder, req)

			chartDeleted := repo.DeleteChartArgsForCall(0)
			Expect(chartDeleted).To(Equal("mysql"))

			Expect(recorder.Code).To(Equal(200))
			Expect(recorder.Body).To(ContainSubstring("deleted"))
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
