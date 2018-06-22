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

package helm_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	var chartPath string
	var testChart *test.TestChart

	BeforeEach(func() {
		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())

		testChart = test.DefaultChart()
		err = testChart.WriteChart(chartPath)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		os.RemoveAll(chartPath)
	})

	It("should load chart", func() {
		chart, err := helm.NewChart(chartPath, "")

		Expect(err).To(BeNil())
		Expect(chart).NotTo(BeNil())
	})

	It("should return error when no vals file", func() {
		err := os.Remove(filepath.Join(chartPath, "values.yaml"))
		Expect(err).To(BeNil())

		_, err = helm.NewChart(chartPath, "")

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("values.yaml"))
	})

	It("reading default vals should return parsed contents", func() {
		chart, err := helm.NewChart(chartPath, "")
		Expect(err).To(BeNil())

		Expect(strings.TrimSpace(string(chart.Values))).
			To(Equal(strings.TrimSpace(string(testChart.ValuesYaml))))
	})

	It("returns error on bad base values yaml", func() {
		err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), []byte(`:foo`), 0666)
		Expect(err).To(BeNil())

		_, err = helm.NewChart(chartPath, "")

		Expect(err).NotTo(BeNil())
	})

	Context("ensure .helmignore", func() {
		It("adds ignore file with images when not present", func() {
			_, err := helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			ignoreContents, err := ioutil.ReadFile(filepath.Join(chartPath, ".helmignore"))
			Expect(err).To(BeNil())
			Expect(ignoreContents).To(Equal([]byte("images")))
		})

		It("appends image to ignore when present", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, ".helmignore"), []byte(`secrets`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			ignoreContents, err := ioutil.ReadFile(filepath.Join(chartPath, ".helmignore"))
			Expect(err).To(BeNil())
			Expect(string(ignoreContents)).To(Equal("secrets\nimages\n"))
		})

		It("appends image to ignore when present", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, ".helmignore"), []byte(`secrets
images
foo`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			ignoreContents, err := ioutil.ReadFile(filepath.Join(chartPath, ".helmignore"))
			Expect(err).To(BeNil())
			Expect(string(ignoreContents)).To(Equal("secrets\nimages\nfoo"))
		})
	})

	Context("override image sources", func() {
		It("does nothing if no private repo configure", func() {
			testChart.ValuesYaml = []byte(`
image: my-image
foo: bar
`)

			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: my-image
`)))
		})

		It("adds prefix in single image case", func() {
			testChart.ValuesYaml = []byte(`
image: my-image
foo: bar
`)
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "docker.example.com/some-scope")

			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: docker.example.com/some-scope/my-image
`)))
		})

		It("replaces existing prefixes if present", func() {
			testChart.ValuesYaml = []byte(`
image: quay.io/my-image
foo: bar
`)
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "docker.example.com/some-scope")

			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: docker.example.com/some-scope/my-image
`)))
		})

		It("adds prefix in multiple image case", func() {
			testChart.ValuesYaml = []byte(`
images:
  thing1:
    image: my-first-image
    tag: latest
  thing2:
    image: my-second-image
    tag: 1.2.3
`)
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "docker.example.com")

			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
images:
  thing1:
    image: docker.example.com/my-first-image
    tag: latest
  thing2:
    image: docker.example.com/my-second-image
    tag: 1.2.3
`)))
		})

		It("returns error on bad IMAGE format", func() {
			testChart.ValuesYaml = []byte(`
image:
  foo: quay.io/my-image
`)
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "docker.example.com")

			Expect(err).NotTo(BeNil())
		})

		It("returns error on bad IMAGES format", func() {
			testChart.ValuesYaml = []byte(`
images:
  thing1: foo
`)
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "docker.example.com")

			Expect(err).NotTo(BeNil())
		})

	})

	It("returns error on bad IMAGES format, inner structure", func() {
		testChart.ValuesYaml = []byte(`
images:
  thing1:
    image: true
`)
		err := testChart.WriteChart(chartPath)
		Expect(err).To(BeNil())

		_, err = helm.NewChart(chartPath, "docker.example.com")

		Expect(err).NotTo(BeNil())
	})

	Context("plans", func() {
		It("loads plan correctly", func() {
			myChart, err := helm.NewChart(chartPath, "")

			Expect(err).To(BeNil())
			Expect(myChart.Plans["small"].Name).To(Equal("small"))
			Expect(myChart.Plans["small"].File).To(Equal("small.yaml"))
			Expect(myChart.Plans["small"].Description).To(Equal("default (small) plan for mysql"))
			Expect(len(myChart.Plans)).To(Equal(2))
			Expect(myChart.Plans["small"].Values).To(Equal(testChart.PlanContents["small"]))
			Expect(myChart.Plans["medium"].Values).To(Equal(testChart.PlanContents["medium"]))

		})

		It("returns error on file read", func() {
			err := os.Remove(filepath.Join(chartPath, "plans.yaml"))
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")
			Expect(err).NotTo(BeNil())
		})

		It("returns error on file marshal", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), []byte(`:foo`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")

			Expect(err).NotTo(BeNil())
		})

		It("returns error invalid underscore in name", func() {

			err := ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), []byte(`
- name: small_plan
  description: invalid values plan
  file: small.yaml
`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid characters"))
		})

		It("returns error invalid spaces in name ", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), []byte(`
- name: small  plan
  description: invalid values plan
  file: small.yaml
`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid characters"))

		})

		It("returns error invalid uppercase letters in name ", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), []byte(`
- name: smallPlans
  description: invalid values plan
  file: small.yaml
`), 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("invalid characters"))
		})
	})


})
