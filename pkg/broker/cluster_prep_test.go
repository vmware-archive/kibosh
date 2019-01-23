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

package broker_test

import (
	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/broker"
	my_config "github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

var _ = Describe("cluster preparation", func() {
	var fakeHelmClient helmfakes.FakeMyHelmClient
	var config *my_config.Config
	var fakeCluster k8sfakes.FakeCluster
	var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
	var operators []*my_helm.MyChart
	var logger *logrus.Logger
	var fakeClusterFactory k8sfakes.FakeClusterFactory
	var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
	var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
	var fakeInstaller helmfakes.FakeInstaller
	var fakeInstallerFactory my_helm.InstallerFactory

	BeforeEach(func() {
		fakeHelmClient = helmfakes.FakeMyHelmClient{}
		config = &my_config.Config{}
		fakeCluster = k8sfakes.FakeCluster{}
		fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
		fakeClusterFactory = k8sfakes.FakeClusterFactory{}
		fakeHelmClientFactory = helmfakes.FakeHelmClientFactory{}
		fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
		fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
		fakeInstaller = helmfakes.FakeInstaller{}
		fakeServiceAccountInstallerFactory = k8sfakes.FakeServiceAccountInstallerFactory{}
		fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
		fakeInstallerFactory = func(c *my_config.Config, cluster k8s.Cluster, client my_helm.MyHelmClient, logger *logrus.Logger) my_helm.Installer {
			return &fakeInstaller
		}

		logger = logrus.New()
		config = &my_config.Config{
			RegistryConfig: &my_config.RegistryConfig{
				Server: "127.0.0.1",
				User:   "k8s",
				Pass:   "monkey123",
				Email:  "k8s@example.com"},
			TillerNamespace: "my-kibosh-namespace",
			HelmTLSConfig:   &my_config.HelmTLSConfig{},
		}
		operators = []*my_helm.MyChart{
			{
				Chart: &hapi_chart.Chart{
					Metadata: &hapi_chart.Metadata{
						Name:        "spacebears",
						Description: "spacebears service and spacebears broker helm chart",
					},
				},
			},
			{
				Chart: &hapi_chart.Chart{
					Metadata: &hapi_chart.Metadata{
						Name:        "mysql",
						Description: "all your data are belong to us",
					},
				},
			},
		}
	})

	Context("basic functionality", func() {
		It("creates namespace", func() {
			err := broker.PrepareCluster(config, &fakeCluster, &fakeHelmClient, &fakeServiceAccountInstaller, fakeInstallerFactory, operators, logger)

			Expect(err).To(BeNil())

			Expect(fakeCluster.CreateNamespaceIfNotExistsCallCount()).To(Equal(1))
			createdNamespace := fakeCluster.CreateNamespaceIfNotExistsArgsForCall(0)
			Expect(createdNamespace.Name).To(Equal("my-kibosh-namespace"))
		})

		It("prepare cluster works", func() {
			err := broker.PrepareCluster(config, &fakeCluster, &fakeHelmClient, &fakeServiceAccountInstaller, fakeInstallerFactory, operators, logger)

			Expect(err).To(BeNil())
			Expect(fakeHelmClient.InstallOperatorCallCount()).To(Equal(2))
			Expect(fakeServiceAccountInstaller.InstallCallCount()).To(Equal(1))
		})

		It("prepare default cluster works", func() {
			err := broker.PrepareDefaultCluster(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, fakeInstallerFactory, logger, operators)

			Expect(err).To(BeNil())
			Expect(fakeHelmClient.InstallOperatorCallCount()).To(Equal(2))
			Expect(fakeServiceAccountInstaller.InstallCallCount()).To(Equal(1))
		})
	})
})
