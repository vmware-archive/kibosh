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
	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	. "github.com/cf-platform-eng/kibosh/pkg/operator"
	"github.com/cf-platform-eng/kibosh/pkg/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

var _ = Describe("Operator", func() {

	var logger lager.Logger
	var registryConfig config.RegistryConfig
	var cluster k8sfakes.FakeCluster
	var client helmfakes.FakeMyHelmClient
	var installer *PksOperator

	var spacebearsChart *my_helm.MyChart
	var mysqlChart *my_helm.MyChart
	var operatorChart []*my_helm.MyChart
	var operatorCharts []*my_helm.MyChart

	BeforeEach(func() {
		logger = lager.NewLogger("test")
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

		installer = NewInstaller(&registryConfig, &cluster, &client, logger)

		spacebearsChart = &my_helm.MyChart{
			Chart: &hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "spacebears",
					Description: "spacebears service and spacebears broker helm chart",
				},
			},
			Plans: map[string]my_helm.Plan{
				"small": {
					Name:        "small",
					Description: "default (small) plan for spacebears",
					File:        "small.yaml",
				},
				"medium": {
					Name:        "medium",
					Description: "medium plan for spacebears",
					File:        "medium.yaml",
				},
			},
		}
		mysqlChart = &my_helm.MyChart{
			Chart: &hapi_chart.Chart{
				Metadata: &hapi_chart.Metadata{
					Name:        "mysql",
					Description: "all your data are belong to us",
				},
			},
			Plans: map[string]my_helm.Plan{
				"small": {
					Name:        "tiny",
					Description: "tiny data",
					File:        "tiny.yaml",
				},
				"medium": {
					Name:        "big",
					Description: "big data",
					File:        "big.yaml",
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

	})

	It("installs one operator chart", func() {
		err := installer.InstallCharts(operatorChart)
		Expect(err).To(BeNil())
	})

	It("installs two operator charts", func() {
		err := installer.InstallCharts(operatorCharts)
		Expect(err).To(BeNil())
	})

	It("doesn't try to install an operator twice", func() {

	})
})
