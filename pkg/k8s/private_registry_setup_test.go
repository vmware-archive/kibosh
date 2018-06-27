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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	api_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Private Registry Setup", func() {
	var fakeCluster *k8sfakes.FakeCluster
	var registryConfig *config.RegistryConfig

	var setup k8s.PrivateRegistrySetup

	BeforeEach(func() {
		fakeCluster = &k8sfakes.FakeCluster{}
		registryConfig = &config.RegistryConfig{
			Server: "registry.example.com",
			User:   "registry-user",
			Pass:   "registry-pass",
			Email:  "bob@example.com",
		}

		setup = k8s.NewPrivateRegistrySetup(
			"my-namespace", "my-service-account", fakeCluster, registryConfig,
		)
	})

	It("creates a new docker registry secret when configured", func() {
		fakeCluster.GetSecretReturns(nil, &api_errors.StatusError{
			ErrStatus: meta_v1.Status{
				Reason: meta_v1.StatusReasonNotFound,
			},
		})

		err := setup.Setup()

		Expect(err).To(BeNil())

		Expect(fakeCluster.CreateSecretCallCount()).To(Equal(1))
		Expect(fakeCluster.UpdateSecretCallCount()).To(Equal(0))

		namespace, secret := fakeCluster.CreateSecretArgsForCall(0)
		Expect(namespace).To(Equal("my-namespace"))

		Expect(secret.Type).To(Equal(api_v1.SecretTypeDockerConfigJson))

		expectedConfig, _ := registryConfig.GetDockerConfigJson()
		Expect(secret.Data).To(Equal(map[string][]byte{
			".dockerconfigjson": expectedConfig,
		}))
	})

	It("creates returns some error that isn't a status error", func() {
		fakeCluster.GetSecretReturns(nil, errors.New("my funky error"))

		err := setup.Setup()

		Expect(err).NotTo(BeNil())

		Expect(fakeCluster.CreateSecretCallCount()).To(Equal(0))
		Expect(fakeCluster.UpdateSecretCallCount()).To(Equal(0))

		Expect(err.Error()).To(Equal("my funky error"))
	})

	It("patches service account", func() {
		err := setup.Setup()

		Expect(err).To(BeNil())

		Expect(fakeCluster.PatchCallCount()).To(Equal(1))

		namespace, name, pathType, data, _ := fakeCluster.PatchArgsForCall(0)
		Expect(namespace).To(Equal("my-namespace"))
		Expect(name).To(Equal("my-service-account"))
		Expect(string(pathType)).To(Equal("application/merge-patch+json"))
		Expect(data).To(Equal([]byte(`{"imagePullSecrets":[{"name":"registry-secret"}]}`)))
	})

	It("return error if patching cluster fails", func() {
		fakeCluster.PatchReturns(nil, errors.New("no patch for you"))

		err := setup.Setup()

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("no patch for you"))
	})

	It("update a docker registry secret when configured", func() {
		fakeCluster.GetSecretReturns(nil, nil)
		err := setup.Setup()

		Expect(err).To(BeNil())

		Expect(fakeCluster.CreateSecretCallCount()).To(Equal(0))
		Expect(fakeCluster.UpdateSecretCallCount()).To(Equal(1))

		namespace, secret := fakeCluster.UpdateSecretArgsForCall(0)
		Expect(namespace).To(Equal("my-namespace"))

		Expect(secret.Type).To(Equal(api_v1.SecretTypeDockerConfigJson))

		expectedConfig, _ := registryConfig.GetDockerConfigJson()
		Expect(secret.Data).To(Equal(map[string][]byte{
			".dockerconfigjson": expectedConfig,
		}))
	})

})
