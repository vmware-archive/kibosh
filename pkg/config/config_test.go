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

package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	. "github.com/cf-platform-eng/kibosh/pkg/config"
	"io/ioutil"
	"path/filepath"
)

var _ = Describe("Config", func() {
	Context("config parsing", func() {
		BeforeEach(func() {
			os.Clearenv()
			os.Setenv("CA_DATA", "c29tZSByYW5kb20gc3R1ZmY=")
			os.Setenv("SERVER", "127.0.0.1/api")
			os.Setenv("TOKEN", "my-token")

			os.Setenv("SECURITY_USER_NAME", "bob")
			os.Setenv("SECURITY_USER_PASSWORD", "abc123")
			os.Setenv("PORT", "9001")
			os.Setenv("HELM_CHART_DIR", "/home/somewhere")

			os.Setenv("CF_API_ADDRESS", "https://api.mycf.example.com")
			os.Setenv("CF_USERNAME", "admin")
			os.Setenv("CF_PASSWORD", "monkey123")
			os.Setenv("CF_SKIP_SSL_VALIDATION", "true")

		})

		It("parses config from environment", func() {
			c, err := Parse()
			Expect(err).To(BeNil())
			Expect(c.AdminUsername).To(Equal("bob"))
			Expect(c.AdminPassword).To(Equal("abc123"))
			Expect(c.HelmChartDir).To(Equal("/home/somewhere"))
			Expect(c.Port).To(Equal(9001))

			Expect(c.ClusterCredentials.Server).To(Equal("127.0.0.1/api"))
		})

		It("parses cf config", func() {
			c, err := Parse()
			Expect(err).To(BeNil())
			Expect(c.CFClientConfig.ApiAddress).To(Equal("https://api.mycf.example.com"))
			Expect(c.CFClientConfig.Username).To(Equal("admin"))
			Expect(c.CFClientConfig.Password).To(Equal("monkey123"))
			Expect(c.CFClientConfig.SkipSslValidation).To(BeTrue())
		})

		It("has registry config", func() {
			c, err := Parse()
			Expect(err).To(BeNil())

			Expect(c.RegistryConfig.HasRegistryConfig()).To(Equal(false))
		})

		It("errors trying to serialize reg config when not present", func() {
			c, err := Parse()
			Expect(err).To(BeNil())

			_, err = c.RegistryConfig.GetDockerConfigJson()
			Expect(err).NotTo(BeNil())
		})

		Context("credentials", func() {
			BeforeEach(func() {
				os.Setenv("CA_DATA", "c29tZSByYW5kb20gc3R1ZmY=")
				os.Setenv("SERVER", "127.0.0.1/api")
				os.Setenv("TOKEN", "my-token")
			})

			It("parses cluster credentials", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.ClusterCredentials).NotTo(BeNil())
				Expect(c.ClusterCredentials.Server).To(Equal("127.0.0.1/api"))
				Expect(c.ClusterCredentials.Token).To(Equal("my-token"))
			})

			It("base 64 decodes ca data", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.ClusterCredentials.CAData).To(Equal([]byte("some random stuff")))
			})

			It("leaves decoded certifcates alone", func() {
				os.Setenv("CA_DATA", `  -----BEGIN CERTIFICATE-----
my cert data
-----END CERTIFICATE-----`)

				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.ClusterCredentials.CAData).To(Equal([]byte(`-----BEGIN CERTIFICATE-----
my cert data
-----END CERTIFICATE-----`)))
			})

			It("bubbles up error on bad cert", func() {
				os.Setenv("CA_DATA", "666F6F")

				_, err := Parse()
				Expect(err).NotTo(BeNil())
			})

		})

		Context("with registry config", func() {
			BeforeEach(func() {
				os.Setenv("REG_SERVER", "https://127.0.0.1")
				os.Setenv("REG_USER", "k8s")
				os.Setenv("REG_PASS", "xyz789")
				os.Setenv("REG_EMAIL", "k8s@example.com")
			})

			It("parses registry config", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.RegistryConfig).NotTo(BeNil())
				Expect(c.RegistryConfig.Server).To(Equal("https://127.0.0.1"))
				Expect(c.RegistryConfig.User).To(Equal("k8s"))
				Expect(c.RegistryConfig.Pass).To(Equal("xyz789"))
				Expect(c.RegistryConfig.Email).To(Equal("k8s@example.com"))
			})

			It("serializes registry config", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				j, err := c.RegistryConfig.GetDockerConfigJson()
				Expect(err).To(BeNil())

				unmarshalled := map[string]interface{}{}
				json.Unmarshal(j, &unmarshalled)

				Expect(unmarshalled).To(Equal(map[string]interface{}{
					"auths": map[string]interface{}{
						"https://127.0.0.1": map[string]interface{}{
							"username": "k8s",
							"password": "xyz789",
							"email":    "k8s@example.com",
						},
					},
				}))
			})
		})

		It("err on missing env values", func() {
			os.Clearenv()

			_, err := Parse()
			Expect(err).NotTo(BeNil())
		})

		Context("tiller config", func() {
			var tlsPath string

			BeforeEach(func() {
				var err error
				tlsPath, err = ioutil.TempDir("", "")
				Expect(err).To(BeNil())
				tlsFile := filepath.Join(tlsPath, "tls_key_file.txt")
				err = ioutil.WriteFile(tlsFile, []byte("foo key"), 0666)
				Expect(err).To(BeNil())

				tlsCertFile := filepath.Join(tlsPath, "tls_cert_file.txt")
				err = ioutil.WriteFile(tlsCertFile, []byte("foo cert"), 0666)
				Expect(err).To(BeNil())

				tlsCAFile := filepath.Join(tlsPath, "tls_ca_file.txt")
				err = ioutil.WriteFile(tlsCAFile, []byte("foo ca"), 0666)
				Expect(err).To(BeNil())

				os.Setenv("TILLER_TLS_KEY_FILE", tlsFile)
				os.Setenv("TILLER_CERT_FILE", tlsCertFile)
				os.Setenv("TILLER_TLS_CA_CERT_FILE", tlsCAFile)
			})

			AfterEach(func() {
				os.RemoveAll(tlsPath)
			})

			It("parse tls config", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.TillerTLSConfig.TLSKeyFile).NotTo(BeEmpty())
				Expect(c.TillerTLSConfig.TLSCertFile).NotTo(BeEmpty())
				Expect(c.TillerTLSConfig.TLSCaCertFile).NotTo(BeEmpty())
			})

			It("error when files don't exists", func() {
				os.RemoveAll(tlsPath)

				_, err := Parse()
				Expect(err).NotTo(BeNil())

				Expect(err.Error()).To(ContainSubstring("tls_"))
			})
		})
	})
})
