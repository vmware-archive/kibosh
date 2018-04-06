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

package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cf-platform-eng/kibosh/cmd/loader"
)

var _ = Describe("Config", func() {
	Context("ImageValues", func() {
		It("validate false when neither image nor images", func() {
			imageValues := ImageValues{}

			Expect(imageValues.ValidateImages()).To(BeFalse())
		})

		It("validate false when images individual images missing name", func() {
			imageValues := ImageValues{
				Images: map[string]ImageValues{
					"thing1": {
						ImageTag: "latest",
					},
				},
			}

			Expect(imageValues.ValidateImages()).To(BeFalse())
		})

		It("validate true with legit single image", func() {
			imageValues := ImageValues{
				Image:    "mysql",
				ImageTag: "5.7.14",
			}

			Expect(imageValues.ValidateImages()).To(BeTrue())
		})

		It("validate true with legit multiple images", func() {
			imageValues := ImageValues{
				Images: map[string]ImageValues{
					"thing1": {
						Image:    "mysql",
						ImageTag: "5.7.14",
					},
					"thing2": {
						Image:    "spacebears",
						ImageTag: "0.1.1",
					},
				},
			}

			Expect(imageValues.ValidateImages()).To(BeTrue())
		})

	})
	Context("ParsedValues", func() {
		valuesYaml := []byte(`
---
images:
  thing1:
    image: "my-first-image"
    imageTag: "5.7.14"
  thing2:
    image: "my-second-image"
    imageTag: "1.2.3"
`)
		var chartPath string
		var err error
		BeforeEach(func() {

			chartPath, err = ioutil.TempDir("", "chart-")
			Expect(err).To(BeNil())
			err = os.Mkdir(filepath.Join(chartPath, "images"), 0700)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), valuesYaml, 0666)
			Expect(err).To(BeNil())

		})

		AfterEach(func() {
			os.RemoveAll(chartPath)
		})

		It("file read error", func() {
			err := os.Remove(filepath.Join(chartPath, "values.yaml"))
			Expect(err).To(BeNil())
			_, err = ParseValues(chartPath)
			Expect(err).NotTo(BeNil())

		})

		It("file parsed success", func() {

			parsedImages, err := ParseValues(chartPath)
			Expect(err).To(BeNil())
			Expect(len(parsedImages.Images)).To(Equal(2))

		})
	})

	Context("DirExists ", func() {
		It("dir is not readable", func() {
			Expect(DirExistsAndIsReadable("/foo/bar/baz")).To(BeFalse())

		})
		It("dir exist and is readable", func() {
			chartPath, err := ioutil.TempDir("", "chart-")
			Expect(err).To(BeNil())
			Expect(DirExistsAndIsReadable(chartPath)).To(BeTrue())
			os.RemoveAll(chartPath)
		})
	})
})
