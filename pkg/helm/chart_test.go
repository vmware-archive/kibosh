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
	"encoding/json"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/helm/pkg/chartutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/test"
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

	It("should load chart with .yml extensions", func() {
		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())

		testChart = test.DefaultChart()
		err = testChart.WriteChart(chartPath, true)
		Expect(err).To(BeNil())
		chart, err := helm.NewChart(chartPath, "")

		Expect(err).To(BeNil())
		Expect(chart).NotTo(BeNil())
		Expect(len(chart.Plans)).To(Equal(2))
		Expect(chart.Plans["small"]).NotTo(BeNil())
		Expect(chart.Plans["medium"]).NotTo(BeNil())
	})

	It("should load chart default values.yaml", func() {
		chart, err := helm.NewChart(chartPath, "")
		Expect(err).To(BeNil())

		values := map[string]interface{}{}
		err = yaml.Unmarshal(chart.TransformedValues, &values)

		Expect(err)
		Expect(values["count"]).To(Equal(1))
		Expect(values["name"]).To(Equal("value"))
	})

	It("loads default plan when no plans.yaml", func() {
		err := os.Remove(filepath.Join(chartPath, "plans.yaml"))
		Expect(err).To(BeNil())

		chart, err := helm.NewChart(chartPath, "")

		Expect(err).To(BeNil())

		Expect(chart.Plans).To(HaveLen(1))
		_, ok := chart.Plans["default"]
		Expect(ok).To(BeTrue())
	})

	Context("serialization", func() {
		It("serializes and desieralizes to json", func() {
			err := testChart.WriteChart(chartPath)
			Expect(err).To(BeNil())

			myChart, err := helm.NewChart(chartPath, "docker.example.com")

			serialized, err := json.Marshal(myChart)
			Expect(err).To(BeNil())

			var deserealized helm.MyChart
			err = json.Unmarshal(serialized, &deserealized)
			Expect(err).To(BeNil())
			Expect(deserealized).NotTo(BeNil())
			Expect(deserealized.Metadata).NotTo(BeNil())
			Expect(deserealized.Metadata.Name).To(Equal("spacebears"))
			Expect(deserealized.TransformedValues).To(Equal(myChart.TransformedValues))

			//Extensions has an `omitempty` that breaks equality comparision: nil != {}
			myChart.Plans["medium"].ClusterConfig.Extensions = nil
			myChart.Plans["medium"].ClusterConfig.Clusters["my-cluster"].Extensions = nil
			myChart.Plans["medium"].ClusterConfig.Contexts["context"].Extensions = nil
			myChart.Plans["medium"].ClusterConfig.Preferences.Extensions = nil
			Expect(myChart.Plans).To(Equal(deserealized.Plans))
		})
	})

	Context("archived chart (tgz)", func() {
		var chartArchivePath string
		BeforeEach(func() {
			chartToSave, err := helm.NewChart(chartPath, "")

			chartArchiveDirPath, err := ioutil.TempDir("", "chartarcive-")
			Expect(err).To(BeNil())

			chartArchivePath, err = chartutil.Save(chartToSave.Chart, chartArchiveDirPath)
			Expect(err).To(BeNil())
		})

		It("should load chart tgz", func() {
			loadedChart, err := helm.NewChart(chartArchivePath, "")

			Expect(err).To(BeNil())
			Expect(loadedChart).NotTo(BeNil())
			Expect(loadedChart.Metadata.Name).To(Equal("spacebears"))
		})

		It("should load values in chart tgz", func() {
			loadedChart, err := helm.NewChart(chartArchivePath, "")

			values := map[string]interface{}{}
			err = yaml.Unmarshal(loadedChart.TransformedValues, &values)

			Expect(err)
			Expect(values["count"]).To(Equal(1))
			Expect(values["name"]).To(Equal("value"))
		})

		It("loads plans", func() {
			loadedChart, err := helm.NewChart(chartArchivePath, "")

			Expect(err).To(BeNil())

			Expect(loadedChart.Plans).To(HaveLen(2))
			small := loadedChart.Plans["small"]
			Expect(small).NotTo(BeNil())
			Expect(*small.Free).To(BeTrue())

			medium := loadedChart.Plans["medium"]
			Expect(medium).NotTo(BeNil())
			Expect(*medium.Free).To(BeFalse())

			vals := map[string]interface{}{}
			err = yaml.Unmarshal(medium.Values, &vals)
			Expect(err).To(BeNil())
			Expect(vals["persistence"]).To(Equal(map[interface{}]interface{}{"size": "16Gi"}))
			Expect(medium.ClusterConfig.CurrentContext).To(Equal("my-context"))
		})

		It("should load default plans when no plans.yaml", func() {
			err := os.Remove(filepath.Join(chartPath, "plans.yaml"))
			err = os.RemoveAll(filepath.Join(chartPath, "plans"))

			chartToSave, err := helm.NewChart(chartPath, "")
			Expect(err).To(BeNil())

			chartArchiveDirPath, err := ioutil.TempDir("", "chartarcive-")
			Expect(err).To(BeNil())

			chartArchivePath, err = chartutil.Save(chartToSave.Chart, chartArchiveDirPath)
			Expect(err).To(BeNil())

			loadedChart, err := helm.NewChart(chartArchivePath, "")

			Expect(err).To(BeNil())
			Expect(loadedChart).NotTo(BeNil())
			Expect(loadedChart.Metadata.Name).To(Equal("spacebears"))

			Expect(loadedChart.Plans).To(HaveLen(1))
			_, ok := loadedChart.Plans["default"]
			Expect(ok).To(BeTrue())
		})
	})

	Context("load from dir", func() {
		var chartArchiveDirPath string

		BeforeEach(func() {
			chartToSave, err := helm.NewChart(chartPath, "")

			chartArchiveDirPath, err = ioutil.TempDir("", "chartarcive-")
			Expect(err).To(BeNil())

			_, err = chartutil.Save(chartToSave.Chart, chartArchiveDirPath)
			Expect(err).To(BeNil())
		})

		It("single chart", func() {
			charts, err := helm.LoadFromDir(chartArchiveDirPath, logrus.New())

			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(1))
			Expect(charts[0].Metadata.Name).To(Equal("spacebears"))
		})

		It("loads plans", func() {
			charts, err := helm.LoadFromDir(chartArchiveDirPath, logrus.New())

			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(1))
			Expect(charts[0].Plans).To(HaveLen(2))
			Expect(charts[0].Plans["small"]).NotTo(BeNil())
		})

		It("skips non-charts", func() {
			err := ioutil.WriteFile(filepath.Join(chartPath, "not-a-chart.tgz"), []byte("nope"), 0666)

			charts, err := helm.LoadFromDir(chartArchiveDirPath, logrus.New())

			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(1))
			Expect(charts[0].Metadata.Name).To(Equal("spacebears"))
		})

		It("multiple charts", func() {
			chartToSave2, err := helm.NewChart(chartPath, "")
			chartToSave2.Metadata.Name = "spacebears2"
			_, err = chartutil.Save(chartToSave2.Chart, chartArchiveDirPath)
			Expect(err).To(BeNil())

			charts, err := helm.LoadFromDir(chartArchiveDirPath, logrus.New())

			Expect(err).To(BeNil())

			Expect(charts).To(HaveLen(2))
			Expect(charts[0].Metadata.Name).To(Equal("spacebears"))
			Expect(charts[1].Metadata.Name).To(Equal("spacebears2"))
		})
	})

	It("should return error when no vals file", func() {
		err := os.Remove(filepath.Join(chartPath, "values.yaml"))
		Expect(err).To(BeNil())

		_, err = helm.NewChart(chartPath, "")

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("values.yaml"))
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

			Expect(strings.TrimSpace(string(chart.TransformedValues))).To(Equal(strings.TrimSpace(`
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
			Expect(strings.TrimSpace(string(chart.TransformedValues))).To(Equal(strings.TrimSpace(`
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
			Expect(strings.TrimSpace(string(chart.TransformedValues))).To(Equal(strings.TrimSpace(`
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
			Expect(strings.TrimSpace(string(chart.TransformedValues))).To(Equal(strings.TrimSpace(`
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

		It("loads credentials", func() {
			credsYaml := []byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: bXktY2VydA==
    server: https://127.0.0.1:8443
  name: my-cluster
contexts:
- context:
    cluster: my-cluster
    user: my-user
  name: my-cluster
current-context: my-cluster
kind: Config
preferences: {}
users:
- name: my-user
  user:
    token: bXktdG9rZW4=
`)

			testChart.PlansYaml = []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yaml"
  credentials: "small-creds.yaml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yaml"
`)

			err := testChart.WriteChart(chartPath)

			Expect(err).To(BeNil())

			credsFile, err := os.Create(filepath.Join(chartPath, "plans", "small-creds.yaml"))
			Expect(err).To(BeNil())

			_, err = credsFile.Write(credsYaml)
			if err != nil {
				Expect(err).To(BeNil())
			}
			credsFile.Close()

			myChart, err := helm.NewChart(chartPath, "")

			Expect(myChart.Plans["medium"].ClusterConfig).To(BeNil())

			smallClusterConfig := myChart.Plans["small"].ClusterConfig
			Expect(smallClusterConfig).NotTo(BeNil())

			currentContext := smallClusterConfig.CurrentContext
			Expect(currentContext).NotTo(Equal(""))

			cluster := smallClusterConfig.Clusters[currentContext]
			Expect(cluster.Server).To(Equal("https://127.0.0.1:8443"))
			auth := smallClusterConfig.AuthInfos[smallClusterConfig.Contexts[currentContext].AuthInfo]
			Expect(auth.Token).To(Equal("bXktdG9rZW4="))
		})

		It("returns error on file read", func() {
			err := os.Remove(filepath.Join(chartPath, "plans", "small.yaml"))
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
