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

package k8s_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cf-platform-eng/kibosh/config"
	. "github.com/cf-platform-eng/kibosh/k8s"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var creds *config.ClusterCredentials

	BeforeEach(func() {
		creds = &config.ClusterCredentials{
			CAData: "c29tZSByYW5kb20gc3R1ZmY=",
			Server: "127.0.0.1/api",
			Token:  "my-token",
		}
	})

	It("list pods", func() {
		var url string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			url = string(r.URL.Path)
		})
		testserver := httptest.NewServer(handler)
		creds.Server = testserver.URL

		cluster, err := NewCluster(creds)

		Expect(err).To(BeNil())

		cluster.ListPods("mynamespace", meta_v1.ListOptions{})

		Expect(url).To(Equal("/api/v1/namespaces/mynamespace/pods"))
	})
})
