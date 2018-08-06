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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"

	. "github.com/cf-platform-eng/kibosh/pkg/helm"
)

var _ = Describe("Client", func() {
	var myHelmClient MyHelmClient
	var chartPath string

	BeforeEach(func() {
		myHelmClient = NewMyHelmClient(nil, nil)

		var err error
		chartPath, err = ioutil.TempDir("", "chart-")
		Expect(err).To(BeNil())
	})

	BeforeEach(func() {
		os.RemoveAll(chartPath)
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

	It("returns an error when the override file is invalid", func() {
		base := []byte(`
foo: bar
`)
		override := []byte(`
- foo: "bar2"
`)
		_, err := myHelmClient.MergeValueBytes(base, override)
		Expect(err).ToNot(BeNil())
	})

})
