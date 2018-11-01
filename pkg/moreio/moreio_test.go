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

package moreio_test

import (
	"io/ioutil"
	"os"

	. "github.com/cf-platform-eng/kibosh/pkg/moreio"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("moreio", func() {
	Context("DirExists ", func() {
		It("dir is not readable", func() {
			Expect(DirExistsAndIsReadable("/foo/bar/baz")).To(BeFalse())

		})

		It("dir exist and is readable", func() {
			chartPath, err := ioutil.TempDir("", "chart-")
			defer os.RemoveAll(chartPath)

			Expect(err).To(BeNil())
			Expect(DirExistsAndIsReadable(chartPath)).To(BeTrue())
		})
	})

	Context("FileExists", func() {
		It("does not exist", func() {
			exists, err := FileExists("/foo/bar/baz")

			Expect(err).To(BeNil())
			Expect(exists).To(BeFalse())
		})

		It("exists", func() {
			path, err := ioutil.TempDir("", "")
			defer os.RemoveAll(path)
			Expect(err).To(BeNil())

			exists, err := FileExists(path)

			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})
})
