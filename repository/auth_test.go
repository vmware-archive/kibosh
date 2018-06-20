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

package repository_test

import (
	"encoding/base64"
	"fmt"
	"net/http/httptest"

	"github.com/cf-platform-eng/kibosh/repository"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	Context("CheckAuth", func() {
		It("false on no auth header", func() {
			filter := repository.NewAuthFilter("bobtheadmin", "monkey123")

			req := httptest.NewRequest("GET", "https://www.example.com/admin", nil)

			Expect(filter.CheckAuth(req)).To(BeFalse())
		})

		It("false on bad auth header", func() {
			filter := repository.NewAuthFilter("bobtheadmin", "monkey123")

			req := httptest.NewRequest("GET", "https://www.example.com/admin", nil)
			auth := base64.StdEncoding.EncodeToString(
				[]byte(fmt.Sprintf("%s:%s", "bobtheadmin", "password")),
			)
			req.Header.Add("Authentication", auth)

			Expect(filter.CheckAuth(req)).To(BeFalse())
		})

		It("true on correct auth header", func() {
			filter := repository.NewAuthFilter("bobtheadmin", "monkey123")

			req := httptest.NewRequest("GET", "https://www.example.com/admin", nil)
			auth := base64.StdEncoding.EncodeToString(
				[]byte(fmt.Sprintf("%s:%s", "bobtheadmin", "monkey123")),
			)
			req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))

			Expect(filter.CheckAuth(req)).To(BeTrue())
		})
	})
})
