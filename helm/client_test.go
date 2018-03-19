package helm_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cf-platform-eng/kibosh/helm"
	"github.com/ghodss/yaml"
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

	It("merge values bytres overrides", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
foo: not bar
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
		}))
	})

	It("keeps non-specified base values", func() {
		base := []byte(`
foo: bar
baz: qux
`)
		override := []byte(`
foo: not bar
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
			"baz": "qux",
		}))
	})

	It("add override values not in base", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
foo: not bar
baz: qux
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())
		Expect(merged).To(Equal(map[string]interface{}{
			"foo": "not bar",
			"baz": "qux",
		}))
	})

	It("nested override", func() {
		base := []byte(`
images:
  thing1:
    image: "my-first-image"
    imageTag: "5.7.14"
  thing2:
    image: "my-second-image"
    imageTag: "1.2.3"
`)
		override := []byte(`
images:
  thing1:
    image: "example.com/my-first-image"
`)

		mergedBytes, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).To(BeNil())

		merged := map[string]interface{}{}
		err = yaml.Unmarshal(mergedBytes, &merged)
		Expect(err).To(BeNil())


		Expect(merged).To(Equal(map[string]interface{}{
			"images": map[string]interface{}{
				"thing1": map[string]interface{}{
					"image":    "example.com/my-first-image",
					"imageTag": "5.7.14",
				},
				"thing2": map[string]interface{}{
					"image":    "my-second-image",
					"imageTag": "1.2.3",
				},
			},
		}))
	})
})
