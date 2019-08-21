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

package operator_test

import (
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	. "github.com/cf-platform-eng/kibosh/pkg/operator"
	"github.com/cf-platform-eng/kibosh/pkg/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmpkg "k8s.io/helm/pkg/helm"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

var _ = Describe("Operator", func() {

	var logger *logrus.Logger
	var registryConfig config.RegistryConfig
	var cluster k8sfakes.FakeCluster
	var client helmfakes.FakeMyHelmClient
	var installer *PksOperator

	var spacebearsChart *my_helm.MyChart
	var mysqlChart *my_helm.MyChart
	var operatorChart []*my_helm.MyChart
	var operatorCharts []*my_helm.MyChart

	BeforeEach(func() {
		logger = logrus.New()
		registryConfig = config.RegistryConfig{
			Server: "127.0.0.1",
			User:   "k8s",
			Pass:   "monkey123",
			Email:  "k8s@example.com",
		}

		k8sClient := test.FakeK8sInterface{}
		cluster = k8sfakes.FakeCluster{}
		cluster.GetClientReturns(&k8sClient)
		client = helmfakes.FakeMyHelmClient{}

		knownNamespaces := []*api_v1.Namespace{}
		cluster.CreateNamespaceStub = func(namespace *api_v1.Namespace) (*api_v1.Namespace, error) {
			knownNamespaces = append(knownNamespaces, namespace)
			return namespace, nil
		}

		cluster.GetNamespaceStub = func(name string, options *meta_v1.GetOptions) (*api_v1.Namespace, error) {
			for _, knownNamespace := range knownNamespaces {
				if name == knownNamespace.ObjectMeta.Name {
					return knownNamespace, nil
				}
			}
			return nil, &api_errors.StatusError{
				ErrStatus: meta_v1.Status{
					Reason: meta_v1.StatusReasonNotFound,
				},
			}
		}

		installedReleases := []*hapi_release.Release{}
		client.ListReleasesStub = func(opts ...helmpkg.ReleaseListOption) (*rls.ListReleasesResponse, error) {
			return &rls.ListReleasesResponse{
				Releases: installedReleases,
			}, nil
		}

		client.InstallOperatorStub = func(chart *my_helm.MyChart, namespace string) (*rls.InstallReleaseResponse, error) {
			installedReleases = append(installedReleases, &hapi_release.Release{
				Name: namespace,
			})
			return &rls.InstallReleaseResponse{}, nil
		}

		installer = NewInstaller(&registryConfig, &cluster, &client, logger)

		spacebearsChart = &my_helm.MyChart{
			Chart: hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "spacebears",
					Description: "spacebears service and spacebears broker helm chart",
				},
			},
			Plans: map[string]my_helm.Plan{
				"small": {
					Name:        "small",
					Description: "default (small) plan for spacebears",
					ValuesFile:  "small.yaml",
				},
				"medium": {
					Name:        "medium",
					Description: "medium plan for spacebears",
					ValuesFile:  "medium.yaml",
				},
			},
		}
		mysqlChart = &my_helm.MyChart{
			Chart: hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "mysql",
					Description: "all your data are belong to us",
				},
			},
			Plans: map[string]my_helm.Plan{
				"small": {
					Name:        "tiny",
					Description: "tiny data",
					ValuesFile:  "tiny.yaml",
				},
				"medium": {
					Name:        "big",
					Description: "big data",
					ValuesFile:  "big.yaml",
				},
			},
		}

		operatorChart = []*my_helm.MyChart{spacebearsChart}
		operatorCharts = []*my_helm.MyChart{spacebearsChart, mysqlChart}
	})

	It("doesn't throw error when there are no operators", func() {
		err := installer.InstallCharts(nil)
		Expect(err).To(BeNil())
	})

	It("installs the namespace", func() {
		err := installer.InstallCharts(operatorChart)
		Expect(err).To(BeNil())
		Expect(cluster.CreateNamespaceCallCount()).To(BeEquivalentTo(1))
	})

	It("doesn't install the namespace twice", func() {
		err := installer.InstallCharts(operatorChart)
		err = installer.InstallCharts(operatorChart)
		Expect(err).To(BeNil())
		Expect(cluster.CreateNamespaceCallCount()).To(BeEquivalentTo(1))
	})

	It("installs one operator chart", func() {
		err := installer.InstallCharts(operatorChart)
		Expect(err).To(BeNil())
		Expect(client.InstallOperatorCallCount()).To(BeEquivalentTo(1))
	})

	It("installs multiple operator charts", func() {
		err := installer.InstallCharts(operatorCharts)
		Expect(err).To(BeNil())
		Expect(client.InstallOperatorCallCount()).To(BeEquivalentTo(len(operatorCharts)))
	})

	It("doesn't try to install an operator twice", func() {
		err := installer.InstallCharts(operatorChart)
		err = installer.InstallCharts(operatorChart)
		Expect(err).To(BeNil())
		Expect(client.InstallOperatorCallCount()).To(BeEquivalentTo(1))
	})
})
