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

	"errors"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	. "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	"github.com/cf-platform-eng/kibosh/pkg/test"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	v1_beta1 "k8s.io/api/extensions/v1beta1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"
)

const helmVersion = "v2.13.1"

var _ = Describe("KubeConfig", func() {
	var logger *logrus.Logger
	var cluster k8sfakes.FakeCluster
	var client helmfakes.FakeMyHelmClient
	var conf *config.Config
	var installer Installer

	BeforeEach(func() {
		logger = logrus.New()
		conf = &config.Config{
			RegistryConfig:  &config.RegistryConfig{},
			TillerNamespace: "my-kibosh-namespace",
			HelmTLSConfig:   &config.HelmTLSConfig{},
		}

		k8sClient := test.FakeK8sInterface{}
		cluster = k8sfakes.FakeCluster{}
		cluster.GetClientReturns(&k8sClient)
		client = helmfakes.FakeMyHelmClient{}

		installer = NewInstaller(conf, &cluster, &client, logger)
	})

	Context("with private registry configured", func() {
		BeforeEach(func() {
			conf = &config.Config{
				RegistryConfig: &config.RegistryConfig{
					Server: "registry.example.com",
					User:   "k8s",
					Pass:   "monkey123",
					Email:  "k8s@example.com",
				},
				HelmTLSConfig:   &config.HelmTLSConfig{},
				TillerNamespace: "my-kibosh-namespace",
			}

			installer = NewInstaller(conf, &cluster, &client, logger)
		})

		It("adds private registry to secret to default service account", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(cluster.PatchCallCount()).To(Equal(1))

			namespace, name, _, data, _ := cluster.PatchArgsForCall(0)
			Expect(namespace).To(Equal("my-kibosh-namespace"))
			Expect(name).To(Equal("kibosh-tiller"))
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
			Expect(opts.Namespace).To(Equal("my-kibosh-namespace"))
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

	Context("insecure", func() {
		It("success", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(client.InstallCallCount()).To(Equal(1))
			Expect(client.UpgradeCallCount()).To(Equal(0))

			opts := client.InstallArgsForCall(0)
			Expect(opts.Namespace).To(Equal("my-kibosh-namespace"))
			Expect(opts.ImageSpec).To(Equal("gcr.io/kubernetes-helm/tiller:" + helmVersion))

			Expect(client.UninstallCallCount()).To(Equal(0))
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
	})

	Context("secure", func() {
		BeforeEach(func() {
			conf = &config.Config{
				RegistryConfig:  &config.RegistryConfig{},
				TillerNamespace: "my-kibosh-namespace",
				HelmTLSConfig: &config.HelmTLSConfig{
					TLSCaCertFile:     "foo",
					TillerTLSKeyFile:  "bar",
					TillerTLSCertFile: "baz",
				},
			}
			installer = NewInstaller(conf, &cluster, &client, logger)
		})

		It("success", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(client.InstallCallCount()).To(Equal(1))
			Expect(client.UpgradeCallCount()).To(Equal(0))

			opts := client.InstallArgsForCall(0)
			Expect(opts.VerifyTLS).To(BeTrue())
			Expect(opts.TLSCaCertFile).To(Equal("foo"))
			Expect(opts.TLSKeyFile).To(Equal("bar"))
			Expect(opts.TLSCertFile).To(Equal("baz"))
			Expect(opts.Namespace).To(Equal("my-kibosh-namespace"))
			Expect(opts.ImageSpec).To(Equal("gcr.io/kubernetes-helm/tiller:" + helmVersion))
		})

		It("uninstalls before installing on tls issue", func() {
			client.HasDifferentTLSConfigReturns(true)
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(client.UninstallCallCount()).To(Equal(1))
		})

		It("only uninstalls before installing on tls issue", func() {
			err := installer.Install()

			Expect(err).To(BeNil())

			Expect(client.UninstallCallCount()).To(Equal(0))
		})

		It("uninstall fail for previous tiller ", func() {
			client.HasDifferentTLSConfigReturns(true)
			client.UninstallReturns(errors.New("internal server error"))

			err := installer.Install()

			Expect(client.UninstallCallCount()).To(Equal(1))
			Expect(err).NotTo(BeNil())
		})
	})
})
