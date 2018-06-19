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
)

var _ = Describe("Repository", func() {
	var chartPath string
	var testChart *test.TestChart

	var logger lager.Logger
	var myRepository repository.Repository

	BeforeEach(func() {
		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())

		testChart = test.DefaultChart()
		err = testChart.WriteChart(chartPath)
		Expect(err).To(BeNil())

		logger = lager.NewLogger("test")
		myRepository = repository.NewRepository(chartPath, "", logger)
	})

	AfterEach(func() {
		os.RemoveAll(chartPath)
	})

	It("load chart returns error on failure", func() {
		os.RemoveAll(chartPath)
		_, err := myRepository.LoadCharts()
		Expect(err).NotTo(BeNil())

		println(err.Error())
	})

	It("returns single chart", func() {
		charts, err := myRepository.LoadCharts()
		Expect(err).To(BeNil())

		Expect(charts).To(HaveLen(1))
		Expect(charts[0].Metadata.Name).To(Equal("spacebears"))
	})
})
