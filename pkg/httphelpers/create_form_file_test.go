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

package httphelpers_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"io/ioutil"
)

var _ = Describe("Save charts", func() {
	var testRequest *http.Request
	var testServer *httptest.Server

	BeforeEach(func() {
	})

	AfterEach(func() {
		testServer.Close()
	})

	It("correctly adds file to request", func() {
		file, err := ioutil.TempFile("", "")
		Expect(err).To(BeNil())
		_, err = file.Write([]byte("some random content stuff"))
		Expect(err).To(BeNil())

		var files = []string{file.Name()}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(4096)
			Expect(err).To(BeNil())
			testRequest = r
		})
		testServer = httptest.NewServer(handler)

		req, err := httphelpers.CreateFormRequest(testServer.URL, "my_file", files)
		Expect(err).To(BeNil())

		res, err := http.DefaultClient.Do(req)
		Expect(err).To(BeNil())
		Expect(res.StatusCode).To(Equal(200))

		Expect(testRequest.Method).To(Equal("POST"))

		formFile, _, err := testRequest.FormFile("my_file")
		Expect(err).To(BeNil())

		fileContents, err := ioutil.ReadAll(formFile)
		Expect(err).To(BeNil())

		Expect(fileContents).To(Equal([]byte("some random content stuff")))
	})

	It("returns error on non-existant file", func() {
		_, err := httphelpers.CreateFormRequest(testServer.URL, "my_file", []string{""})
		Expect(err).NotTo(BeNil())
	})
})
