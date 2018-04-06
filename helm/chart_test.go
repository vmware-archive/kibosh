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

	"github.com/cf-platform-eng/kibosh/helm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"strings"
)

var _ = Describe("Broker", func() {
	chartYaml := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	valuesYaml := []byte(`
name: value
`)
	plansYaml := []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yaml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yaml"	
`)

	smallYaml := []byte(``)
	mediumYaml := []byte(`
persistence:
  size: 16Gi
`)

	var chartPath string

	BeforeEach(func() {
		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())
		err = os.Mkdir(filepath.Join(chartPath, "plans"), 0700)
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), chartYaml, 0666)
		Expect(err).To(BeNil())
		err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
		Expect(err).To(BeNil())
		err = ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), plansYaml, 0666)
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(filepath.Join(chartPath, "plans", "small.yaml"), smallYaml, 0666)
		Expect(err).To(BeNil())
		err = ioutil.WriteFile(filepath.Join(chartPath, "plans", "medium.yaml"), mediumYaml, 0666)
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
		err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
		Expect(err).To(BeNil())

		chart, err := helm.NewChart(chartPath, "")
		Expect(err).To(BeNil())

		Expect(chart.Values).To(Equal([]byte("name: value\n")))
	})

	It("returns error on bad base values yaml", func() {
		err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), []byte(`:foo`), 0666)
		Expect(err).To(BeNil())

		_, err = helm.NewChart(chartPath, "")

		Expect(err).NotTo(BeNil())
	})

	Context("override image sources", func() {
		It("does nothing if no private repo configure", func() {
			valuesYaml = []byte(`
image: my-image
foo: bar
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: my-image
`)))
		})

		It("adds prefix in single image case", func() {
			valuesYaml = []byte(`
image: my-image
foo: bar
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "docker.example.com/some-scope")

			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: docker.example.com/some-scope/my-image
`)))
		})

		It("replaces existing prefixes if present", func() {
			valuesYaml = []byte(`
image: quay.io/my-image
foo: bar
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

			chart, err := helm.NewChart(chartPath, "docker.example.com/some-scope")

			Expect(err).To(BeNil())
			Expect(strings.TrimSpace(string(chart.Values))).To(Equal(strings.TrimSpace(`
foo: bar
image: docker.example.com/some-scope/my-image
`)))
		})

		It("adds prefix in multiple image case", func() {
			valuesYaml = []byte(`
images:
  thing1:
    image: my-first-image
    tag: latest
  thing2:
    image: my-second-image
    tag: 1.2.3
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
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
			valuesYaml = []byte(`
image:
  foo: quay.io/my-image
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "docker.example.com")

			Expect(err).NotTo(BeNil())
		})

		It("returns error on bad IMAGES format", func() {
			valuesYaml = []byte(`
images:
  thing1: foo
`)
			err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

			_, err = helm.NewChart(chartPath, "docker.example.com")

			Expect(err).NotTo(BeNil())
		})

	})

	It("returns error on bad IMAGES format, inner structure", func() {
		valuesYaml = []byte(`
images:
  thing1:
    image: true
`)
		err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
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
			Expect(myChart.Plans["small"].Values).To(Equal(smallYaml))
			Expect(myChart.Plans["medium"].Values).To(Equal(mediumYaml))

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
