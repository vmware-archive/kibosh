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

	var chartPath string

	BeforeEach(func() {
		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), chartYaml, 0666)
		Expect(err).To(BeNil())
		err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
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
})
