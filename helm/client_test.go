package helm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cf-platform-eng/kibosh/helm"
	"io/ioutil"
	"os"
	"path/filepath"
)

var _ = Describe("Client", func() {
	chartYaml := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)
	valuesYaml := []byte(`
name: value
`)

	var myHelmClient MyHelmClient
	var chartPath string

	BeforeEach(func() {
		myHelmClient = NewMyHelmClient(nil, nil)

		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())

		err = ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), chartYaml, 0666)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		os.RemoveAll(chartPath)
	})

	It("should return error when no vals file", func() {
		_, err := myHelmClient.ReadDefaultVals(chartPath)

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("values.yaml"))
	})

	It("should return parsed contents", func() {
		err := ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
		Expect(err).To(BeNil())

		parseVals, err := myHelmClient.ReadDefaultVals(chartPath)
		Expect(err).To(BeNil())

		Expect(parseVals).To(Equal(parseVals))
	})
})
