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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager"
	"errors"
	"github.com/cf-platform-eng/kibosh/config"
	. "github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/k8s/k8sfakes"
	"github.com/cf-platform-eng/kibosh/test"
	"k8s.io/api/core/v1"
	v1_beta1 "k8s.io/api/extensions/v1beta1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"
)

const helmVersion = "v2.8.2"

var _ = Describe("KubeConfig", func() {
	var logger lager.Logger
	var registryConfig config.RegistryConfig
	var cluster k8sfakes.FakeCluster
	var client helmfakes.FakeMyHelmClient
	var installer Installer

	BeforeEach(func() {
		logger = lager.NewLogger("test")
		registryConfig = config.RegistryConfig{}

		k8sClient := test.FakeK8sInterface{}
		cluster = k8sfakes.FakeCluster{}
		cluster.GetClientReturns(&k8sClient)
		client = helmfakes.FakeMyHelmClient{}

		installer = NewInstaller(&registryConfig, &cluster, &client, logger)
	})

	It("success", func() {
		err := installer.Install()

		Expect(err).To(BeNil())

		Expect(client.InstallCallCount()).To(Equal(1))
		Expect(client.UpgradeCallCount()).To(Equal(0))

		opts := client.InstallArgsForCall(0)
		Expect(opts.Namespace).To(Equal("kube-system"))
		Expect(opts.ImageSpec).To(Equal("gcr.io/kubernetes-helm/tiller:" + helmVersion))
	})

	It("upgrade required", func() {
		client.InstallReturns(api_errors.NewAlreadyExists(schema.GroupResource{}, ""))
		cluster.GetDeploymentReturns(
			&v1_beta1.Deployment{
				Spec: v1_beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Image: "gcr.io/kubernetes-helm/tiller:v2.7.0"},
							}},
					},
				},
			}, nil,
		)

		err := installer.Install()

		Expect(err).To(BeNil())
		Expect(client.InstallCallCount()).To(Equal(1))
		Expect(client.UpgradeCallCount()).To(Equal(1))
	})

	It("installed but upgrade not required (same version)", func() {
		client.InstallReturns(api_errors.NewAlreadyExists(schema.GroupResource{}, ""))
		cluster.GetDeploymentReturns(
			&v1_beta1.Deployment{
				Spec: v1_beta1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Image: "gcr.io/kubernetes-helm/tiller:" + helmVersion},
							}},
					},
				},
			}, nil,
		)

		err := installer.Install()

		Expect(err).To(BeNil())
		Expect(client.InstallCallCount()).To(Equal(1))
		Expect(client.UpgradeCallCount()).To(Equal(0))
	})

	It("blocks on error", func() {
		client.ListReleasesReturnsOnCall(0, nil, errors.New("broker"))
		client.ListReleasesReturnsOnCall(1, nil, errors.New("broker"))
		client.ListReleasesReturnsOnCall(2, nil, nil)
		installer.SetMaxWait(1 * time.Millisecond)

		err := installer.Install()

		Expect(client.ListReleasesCallCount()).To(Equal(3))
		Expect(err).To(BeNil())
	})

	It("returns error if helm doesn't become healthy", func() {
		client.ListReleasesReturns(nil, errors.New("No helm for you"))
		installer.SetMaxWait(1 * time.Millisecond)

		err := installer.Install()

		Expect(err).NotTo(BeNil())
	})

	Context("with private registry configured", func() {
		BeforeEach(func() {
			registryConfig = config.RegistryConfig{
				Server: "registry.example.com",
				User:   "k8s",
				Pass:   "monkey123",
				Email:  "k8s@example.com",
			}

			installer = NewInstaller(&registryConfig, &cluster, &client, logger)
		})

		It("adds private registry to secret to default service account", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(cluster.PatchCallCount()).To(Equal(1))

			namespace, name, _, data, _ := cluster.PatchArgsForCall(0)
			Expect(namespace).To(Equal("kube-system"))
			Expect(name).To(Equal("tiller"))
			Expect(data).To(Equal([]byte(`{"imagePullSecrets":[{"name":"registry-secret"}]}`)))
		})

		It("if private registry setup fails, error bubbles up", func() {
			cluster.PatchReturns(nil, errors.New("no patch for you"))

			err := installer.Install()

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("no patch for you"))
		})

		It("should use private registry image for install when configured", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(client.InstallCallCount()).To(Equal(1))
			Expect(client.UpgradeCallCount()).To(Equal(0))

			opts := client.InstallArgsForCall(0)
			Expect(opts.Namespace).To(Equal("kube-system"))
			Expect(opts.ImageSpec).To(Equal("registry.example.com/tiller:" + helmVersion))
		})

		It("upgrade required", func() {
			client.InstallReturns(api_errors.NewAlreadyExists(schema.GroupResource{}, ""))
			cluster.GetDeploymentReturns(
				&v1_beta1.Deployment{
					Spec: v1_beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{Image: "gcr.io/kubernetes-helm/tiller:" + helmVersion},
								}},
						},
					},
				}, nil,
			)

			err := installer.Install()

			Expect(err).To(BeNil())
			Expect(client.InstallCallCount()).To(Equal(1))
			Expect(client.UpgradeCallCount()).To(Equal(0))
		})
	})
})
