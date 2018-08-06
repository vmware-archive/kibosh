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

package bazaar_test

import (
	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Clearenv()

		os.Setenv("SECURITY_USER_NAME", "bob")
		os.Setenv("SECURITY_USER_PASSWORD", "abc123")
		os.Setenv("PORT", "9001")
		os.Setenv("HELM_CHART_DIR", "/home/somewhere")
		os.Setenv("KIBOSH_SERVER", "mykibosh.com")
		os.Setenv("KIBOSH_USER_NAME", "kevin")
		os.Setenv("KIBOSH_USER_PASSWORD", "monkey123")
	})

	It("parses config from environment", func() {
		c, err := bazaar.ParseConfig()
		Expect(err).To(BeNil())
		Expect(c.AdminUsername).To(Equal("bob"))
		Expect(c.AdminPassword).To(Equal("abc123"))
		Expect(c.HelmChartDir).To(Equal("/home/somewhere"))
		Expect(c.Port).To(Equal(9001))
		Expect(c.KiboshConfig.Server).To(Equal("mykibosh.com"))
		Expect(c.KiboshConfig.User).To(Equal("kevin"))
		Expect(c.KiboshConfig.Pass).To(Equal("monkey123"))

	})

})
