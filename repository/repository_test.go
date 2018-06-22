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
	"github.com/cf-platform-eng/kibosh/repository"
	"github.com/cf-platform-eng/kibosh/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
	"k8s.io/helm/pkg/chartutil"
	"github.com/cf-platform-eng/kibosh/helm"
)

var _ = Describe("Repository", func() {
	var chartPath string
	var testChart *test.TestChart

	var logger lager.Logger

	It("load chart returns error on failure", func() {
		myRepository := repository.NewRepository(chartPath, "", logger)
		_, err := myRepository.LoadCharts()
		Expect(err).NotTo(BeNil())
	})

	Context("single charts", func() {
		BeforeEach(func() {
			var err error
			chartPath, err = ioutil.TempDir("", "chart-")
			Expect(err).To(BeNil())

			testChart = test.DefaultChart()
			err = testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			logger = lager.NewLogger("test")
		})

		AfterEach(func() {
			os.RemoveAll(chartPath)
		})

		It("returns single chart", func() {
			myRepository := repository.NewRepository(chartPath, "", logger)
			charts, err := myRepository.LoadCharts()
			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(1))
			Expect(charts[0].Metadata.Name).To(Equal("spacebears"))
		})
	})

	Context("multiple charts", func() {
		BeforeEach(func() {
			var err error
			chartPath, err = ioutil.TempDir("", "chart-")
			Expect(err).To(BeNil())

			testChart = test.DefaultChart()
			testChart.ChartYaml = []byte(`
name: postgres
description: store some data, relational style
version: 0.0.1
`)
			c1Dir := filepath.Join(chartPath, "c1")
			err = os.Mkdir(c1Dir, 0700)
			Expect(err).To(BeNil())
			err = testChart.WriteChart(c1Dir)
			Expect(err).To(BeNil())

			testChart = test.DefaultChart()
			testChart.ChartYaml = []byte(`
name: mysql
description: it's the M in all those acronums
version: 0.0.1
`)

			c2Dir := filepath.Join(chartPath, "c2")
			err = os.Mkdir(c2Dir, 0700)
			Expect(err).To(BeNil())
			err = testChart.WriteChart(c2Dir)
			Expect(err).To(BeNil())

			notChartDir := filepath.Join(chartPath, "notchart")
			err = os.Mkdir(notChartDir, 0700)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			os.RemoveAll(chartPath)
		})

		It("loads multiple charts", func() {
			logger = lager.NewLogger("test")
			myRepository := repository.NewRepository(chartPath, "", logger)

			charts, err := myRepository.LoadCharts()
			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(2))
			Expect(charts[0].Metadata.Name).To(Equal("postgres"))
			Expect(charts[1].Metadata.Name).To(Equal("mysql"))
		})

		It("bubbles up chart load errors", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, "c2", "Chart.yaml"), []byte(`bad::::yaml`), 0666)
			Expect(err).To(BeNil())

			logger = lager.NewLogger("test")
			myRepository := repository.NewRepository(chartPath, "", logger)

			_, err = myRepository.LoadCharts()
			Expect(err).NotTo(BeNil())
		})
	})
	Context("save chart", func() {

		It("save chart untars file", func() {

			tarDir, err := ioutil.TempDir("", "")

			err = testChart.WriteChart(tarDir)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(tarDir, "docker.example.com")

			tarfile, err := chartutil.Save(chart.Chart, tarDir)
			myRepository := repository.NewRepository(tarfile, "", logger)
			err = myRepository.SaveChart(tarfile)
			Expect(err).To(BeNil())



		})
	})
})
