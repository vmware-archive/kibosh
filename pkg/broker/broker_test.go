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
	"encoding/json"
	"errors"
	"strings"

	"code.cloudfoundry.org/lager"
	. "github.com/cf-platform-eng/kibosh/pkg/broker"
	my_config "github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	hapi_services "k8s.io/helm/pkg/proto/hapi/services"
)

type FakeClusterFactory struct {
	fakeCluster *k8sfakes.FakeCluster
}

func NewFakeClusterFactory() *FakeClusterFactory {
	return &FakeClusterFactory{&k8sfakes.FakeCluster{}}
}

func (cf FakeClusterFactory) DefaultCluster() (k8s.Cluster, error) {
	return cf.fakeCluster, nil
}

type FakeHelmClientFactory struct {
	fakeHelmClient *helmfakes.FakeMyHelmClient
}

func NewFakeHelmClientFactory() *FakeHelmClientFactory {
	return &FakeHelmClientFactory{&helmfakes.FakeMyHelmClient{}}
}

func (hcf FakeHelmClientFactory) HelmClient(cluster k8s.Cluster) my_helm.MyHelmClient {
	return hcf.fakeHelmClient
}

type FakeServiceAccountInstallerFactory struct {
	fakeServiceAccountInstaller *k8sfakes.FakeServiceAccountInstaller
}

func NewFakeServiceAccountInstallerFactory() *FakeServiceAccountInstallerFactory {
	return &FakeServiceAccountInstallerFactory{&k8sfakes.FakeServiceAccountInstaller{}}
}

func (fsaif FakeServiceAccountInstallerFactory) ServiceAccountInstaller(cluster k8s.Cluster) k8s.ServiceAccountInstaller {
	return fsaif.fakeServiceAccountInstaller
}

var _ = Describe("Broker", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"
	const mysqlServiceGUID = "c76ed0a4-9a04-5710-90c2-75e955697b08"

	var logger lager.Logger

	var spacebearsChart *my_helm.MyChart
	var mysqlChart *my_helm.MyChart
	var charts []*my_helm.MyChart

	var config *my_config.Config

	BeforeEach(func() {
		logger = lager.NewLogger("test")
		config = &my_config.Config{
			RegistryConfig: &my_config.RegistryConfig{
				Server: "127.0.0.1",
				User:   "k8s",
				Pass:   "monkey123",
				Email:  "k8s@example.com"},
			TillerTLSConfig: &my_config.TillerTLSConfig{},
		}
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

		charts = []*my_helm.MyChart{spacebearsChart, mysqlChart}
	})

	Context("catalog", func() {
		It("Provides a catalog with correct service", func() {
			serviceBroker := NewPksServiceBroker(config, nil, nil, nil, charts, logger)
			serviceCatalog, err := serviceBroker.Services(nil)
			Expect(err).To(BeNil())

			Expect(len(serviceCatalog)).To(Equal(2))

			var spacebearsService brokerapi.Service
			var mysqlService brokerapi.Service
			if serviceCatalog[0].ID == spacebearsServiceGUID {
				spacebearsService = serviceCatalog[0]
				mysqlService = serviceCatalog[1]
			} else if serviceCatalog[1].ID == spacebearsServiceGUID {
				spacebearsService = serviceCatalog[1]
				mysqlService = serviceCatalog[0]
			} else {
				panic("Spacebears service not found")
			}

			Expect(spacebearsService.Name).To(Equal("spacebears"))
			Expect(spacebearsService.Description).To(Equal("spacebears service and spacebears broker helm chart"))
			Expect(spacebearsService.Bindable).To(BeTrue())

			Expect(mysqlService.ID).To(Equal(mysqlServiceGUID))
			Expect(mysqlService.Name).To(Equal("mysql"))
			Expect(mysqlService.Description).To(Equal("all your data are belong to us"))
		})

		It("Provides a catalog with correct plans", func() {
			serviceBroker := NewPksServiceBroker(config, nil, nil, nil, charts, logger)
			serviceCatalog, err := serviceBroker.Services(nil)
			Expect(err).To(BeNil())

			expectedPlans := []brokerapi.ServicePlan{
				{
					ID:          "37b7acb6-6755-56fe-a17f-2307657023ef-small",
					Name:        "small",
					Description: "default (small) plan for spacebears",
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "small",
						Bullets: []string{
							"default (small) plan for spacebears",
						},
					},
				},
				{
					ID:          "37b7acb6-6755-56fe-a17f-2307657023ef-medium",
					Name:        "medium",
					Description: "medium plan for spacebears",
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "medium",
						Bullets: []string{
							"medium plan for spacebears",
						},
					},
				},
			}

			service1 := serviceCatalog[0]
			service2 := serviceCatalog[1]
			if service1.Name == "spacebears" {
				Expect(service1.Plans).Should(ConsistOf(expectedPlans))
			} else {
				Expect(service2.Plans).Should(ConsistOf(expectedPlans))
			}

		})
	})

	Context("provision", func() {
		var details brokerapi.ProvisionDetails
		var fakeHelmClientFactory *FakeHelmClientFactory
		var fakeClusterFactory *FakeClusterFactory
		var fakeServiceInstallerAccountInstallerFactory *FakeServiceAccountInstallerFactory
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClientFactory = NewFakeHelmClientFactory()
			fakeClusterFactory = NewFakeClusterFactory()
			fakeServiceInstallerAccountInstallerFactory = NewFakeServiceAccountInstallerFactory()
			details = brokerapi.ProvisionDetails{
				ServiceID: spacebearsServiceGUID,
			}

			broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)
		})

		It("requires async", func() {
			_, err := broker.Provision(nil, "my-instance-guid", details, false)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("async"))

		})

		It("responds correctly", func() {
			resp, err := broker.Provision(nil, "my-instance-guid", details, true)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
			Expect(resp.OperationData).To(Equal("provision"))
		})

		Context("namespace", func() {
			It("creates a new namespace", func() {
				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())

				Expect(fakeClusterFactory.fakeCluster.CreateNamespaceCallCount()).To(Equal(1))

				namespace := fakeClusterFactory.fakeCluster.CreateNamespaceArgsForCall(0)
				Expect(namespace.Name).To(Equal("kibosh-my-instance-guid"))
			})

			It("returns error on namespace creation failure", func() {
				errorMessage := "namespace already taken or something"
				fakeClusterFactory.fakeCluster.CreateNamespaceReturns(nil, errors.New(errorMessage))

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring(errorMessage))
			})
		})

		Context("service account", func() {
			It("creates service account", func() {
				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())

				Expect(fakeServiceInstallerAccountInstallerFactory.fakeServiceAccountInstaller.InstallCallCount()).To(Equal(1))
			})

			It("returns error on service account installation failure", func() {
				errorMessage := "could not create service account"
				fakeServiceInstallerAccountInstallerFactory.fakeServiceAccountInstaller.InstallReturns(errors.New(errorMessage))

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring(errorMessage))
			})
		})

		Context("registry secrets", func() {
			It("doesn't mess with secrets when not configured", func() {
				config = &my_config.Config{RegistryConfig: &my_config.RegistryConfig{}, TillerTLSConfig: &my_config.TillerTLSConfig{}}
				broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())

				Expect(fakeClusterFactory.fakeCluster.UpdateSecretCallCount()).To(Equal(0))
				Expect(fakeClusterFactory.fakeCluster.PatchCallCount()).To(Equal(0))
			})
		})

		Context("chart", func() {
			It("creates helm chart", func() {
				planID := spacebearsServiceGUID + "-small"
				_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{
					ServiceID: spacebearsServiceGUID,
					PlanID:    planID,
				}, true)

				Expect(err).To(BeNil())

				Expect(fakeHelmClientFactory.fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				chart, namespaceName, plan, opts := fakeHelmClientFactory.fakeHelmClient.InstallChartArgsForCall(0)
				Expect(chart).To(Equal(spacebearsChart))
				Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
				Expect(plan).To(Equal("small"))
				Expect(opts).To(BeNil())
			})

			It("returns error on helm chart creation failure", func() {
				errorMessage := "no helm for you"
				fakeHelmClientFactory.fakeHelmClient.InstallChartReturns(nil, errors.New(errorMessage))

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring(errorMessage))
			})

			It("provisions correct chart", func() {
				_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{
					ServiceID: mysqlServiceGUID,
					PlanID:    mysqlServiceGUID + "-tiny",
				}, true)

				Expect(err).To(BeNil())

				Expect(fakeHelmClientFactory.fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				chart, _, _, _ := fakeHelmClientFactory.fakeHelmClient.InstallChartArgsForCall(0)
				Expect(chart).To(Equal(mysqlChart))
			})

			It("creates helm chart with values", func() {
				planID := spacebearsServiceGUID + "-small"
				raw := json.RawMessage(`{"foo":"bar"}`)

				_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{
					ServiceID:     spacebearsServiceGUID,
					PlanID:        planID,
					RawParameters: raw,
				}, true)

				Expect(err).To(BeNil())

				Expect(fakeHelmClientFactory.fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				chart, namespaceName, plan, opts := fakeHelmClientFactory.fakeHelmClient.InstallChartArgsForCall(0)
				Expect(chart).To(Equal(spacebearsChart))
				Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
				Expect(plan).To(Equal("small"))
				Expect(strings.TrimSpace(string(opts))).To(Equal("foo: bar"))
			})
		})
	})

	Context("last operation", func() {
		var fakeHelmClientFactory *FakeHelmClientFactory
		var fakeClusterFactory *FakeClusterFactory
		var fakeServiceInstallerAccountInstallerFactory *FakeServiceAccountInstallerFactory
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClientFactory = NewFakeHelmClientFactory()
			fakeClusterFactory = NewFakeClusterFactory()
			fakeServiceInstallerAccountInstallerFactory = NewFakeServiceAccountInstallerFactory()

			broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)

			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{
							LoadBalancer: api_v1.LoadBalancerStatus{
								Ingress: []api_v1.LoadBalancerIngress{
									{IP: "127.0.0.1"},
								},
							},
						},
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)
			podList := api_v1.PodList{
				Items: []api_v1.Pod{},
			}
			fakeClusterFactory.fakeCluster.ListPodsReturns(&podList, nil)

		})

		It("elevates error from helm", func() {
			errMessage := "helm communication failure or something"
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(nil, errors.New(errMessage))

			_, err := broker.LastOperation(nil, "my-instance-guid", "???")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(errMessage))
		})

		It("returns success if deployed", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("succeeded"))
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
		})

		It("returns pending install", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_PENDING_INSTALL,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("in progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("returns success if updated", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "update")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("updated"))
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
		})

		It("returns pending upgrade", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_PENDING_UPGRADE,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("in progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("returns ok when instance is gone ", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DELETED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "deprovision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("gone"))
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
		})

		It("returns error when instance is gone when trying to create", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DELETED,
					},
				},
			}, nil)

			resp, _ := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(resp.State).To(Equal(brokerapi.Failed))
		})

		It("returns delete in progress", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DELETING,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "deprovision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("in progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("returns failed", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_FAILED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "deprovision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("failed"))
			Expect(resp.State).To(Equal(brokerapi.Failed))
		})

		It("waits until load balancer servers have ingress", func() {
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "LoadBalancer",
						},
						Status: api_v1.ServiceStatus{},
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("waits until pods are running", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			podList := api_v1.PodList{
				Items: []api_v1.Pod{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "pod1"},
						Spec:       api_v1.PodSpec{},
						Status: api_v1.PodStatus{
							Phase: "Pending",
							Conditions: []api_v1.PodCondition{
								{
									Status:  "False",
									Type:    "PodScheduled",
									Reason:  "Unschedulable",
									Message: "0/1 nodes are available: 1 Insufficient memory",
								},
							},
						},
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListPodsReturns(&podList, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.State).To(Equal(brokerapi.InProgress))
			Expect(resp.Description).To(ContainSubstring("0/1 nodes are available: 1 Insufficient memory"))
		})

		It("returns error when unable to list pods", func() {
			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			fakeClusterFactory.fakeCluster.ListPodsReturns(nil, errors.New("nope"))

			_, err := broker.LastOperation(nil, "my-instance-guid", "provision")
			Expect(err).NotTo(BeNil())
		})

		It("bubbles up error on list service failure", func() {
			fakeClusterFactory.fakeCluster.ListServicesReturns(&api_v1.ServiceList{}, errors.New("no services for you"))

			fakeHelmClientFactory.fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			_, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).NotTo(BeNil())
		})
	})

	Context("bind", func() {
		var fakeHelmClientFactory *FakeHelmClientFactory
		var fakeClusterFactory *FakeClusterFactory
		var fakeServiceInstallerAccountInstallerFactory *FakeServiceAccountInstallerFactory
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClientFactory = NewFakeHelmClientFactory()
			fakeClusterFactory = NewFakeClusterFactory()
			fakeServiceInstallerAccountInstallerFactory = NewFakeServiceAccountInstallerFactory()

			broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)
		})

		It("bind returns cluster secrets", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)

			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListSecretsReturns(&secretsList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeClusterFactory.fakeCluster.ListSecretsCallCount()).To(Equal(1))

			Expect(binding).NotTo(BeNil())

			creds := binding.Credentials
			secrets := creds.(map[string]interface{})["secrets"]
			secretsJson, err := json.Marshal(secrets)
			Expect(string(secretsJson)).To(Equal(`[{"data":{"db-password":"abc123"},"name":"passwords"}]`))
		})

		It("bind filters to only opaque secrets", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)

			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					}, {
						ObjectMeta: meta_v1.ObjectMeta{Name: "default-token-xyz"},
						Data:       map[string][]byte{"token": []byte("my-token")},
						Type:       api_v1.SecretTypeServiceAccountToken,
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListSecretsReturns(&secretsList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeClusterFactory.fakeCluster.ListSecretsCallCount()).To(Equal(1))

			Expect(binding).NotTo(BeNil())

			creds := binding.Credentials
			secrets := creds.(map[string]interface{})["secrets"]
			secretsJson, err := json.Marshal(secrets)
			Expect(string(secretsJson)).To(Equal(`[{"data":{"db-password":"abc123"},"name":"passwords"}]`))
		})

		It("bubbles up list secrets errors", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeClusterFactory.fakeCluster.ListSecretsReturns(nil, errors.New("foo failed"))
			_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).NotTo(BeNil())
			Expect(fakeClusterFactory.fakeCluster.ListSecretsCallCount()).To(Equal(1))
		})

		It("returns services (load balancer stuff)", func() {
			secretList := api_v1.SecretList{Items: []api_v1.Secret{}}
			fakeClusterFactory.fakeCluster.ListSecretsReturns(&secretList, nil)

			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{
								{
									Name:     "mysql",
									NodePort: 30092,
									Port:     3306,
									Protocol: "TCP",
								},
							},
						},
						Status: api_v1.ServiceStatus{
							LoadBalancer: api_v1.LoadBalancerStatus{
								Ingress: []api_v1.LoadBalancerIngress{
									{IP: "127.0.0.1"},
								},
							},
						},
					},
				},
			}
			fakeClusterFactory.fakeCluster.ListServicesReturns(&serviceList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeClusterFactory.fakeCluster.ListServicesCallCount()).To(Equal(1))

			Expect(binding).NotTo(BeNil())

			creds := binding.Credentials
			services := creds.(map[string]interface{})["services"]
			name := services.([]map[string]interface{})[0]["name"]
			Expect(name).To(Equal("kibosh-my-mysql-db-instance"))

			spec := services.([]map[string]interface{})[0]["spec"]
			specJson, _ := json.Marshal(spec)
			Expect(string(specJson)).To(Equal(`{"ports":[{"name":"mysql","protocol":"TCP","port":3306,"targetPort":0,"nodePort":30092}]}`))

			status := services.([]map[string]interface{})[0]["status"]
			statusJson, _ := json.Marshal(status)
			Expect(string(statusJson)).To(Equal(`{"loadBalancer":{"ingress":[{"ip":"127.0.0.1"}]}}`))
		})

		It("bubbles up list services errors", func() {
			secretList := api_v1.SecretList{Items: []api_v1.Secret{}}
			fakeClusterFactory.fakeCluster.ListSecretsReturns(&secretList, nil)

			fakeClusterFactory.fakeCluster.ListServicesReturns(nil, errors.New("no services for you"))

			_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).NotTo(BeNil())
		})
	})

	Context("delete", func() {
		var fakeHelmClientFactory *FakeHelmClientFactory
		var fakeClusterFactory *FakeClusterFactory
		var fakeServiceInstallerAccountInstallerFactory *FakeServiceAccountInstallerFactory
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClientFactory = NewFakeHelmClientFactory()
			fakeClusterFactory = NewFakeClusterFactory()
			fakeServiceInstallerAccountInstallerFactory = NewFakeServiceAccountInstallerFactory()

			broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)
		})

		It("bubbles up delete chart errors", func() {
			fakeHelmClientFactory.fakeHelmClient.DeleteReleaseReturns(nil, errors.New("Failed"))

			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeHelmClientFactory.fakeHelmClient.DeleteReleaseCallCount()).To(Equal(1))

		})

		It("bubbles up delete namespace errors", func() {
			fakeClusterFactory.fakeCluster.DeleteNamespaceReturns(errors.New("nope"))

			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeClusterFactory.fakeCluster.DeleteNamespaceCallCount()).To(Equal(1))

		})

		It("correctly calls deletion", func() {
			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			response, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			namespace, _ := fakeClusterFactory.fakeCluster.DeleteNamespaceArgsForCall(0)
			Expect(namespace).To(Equal("kibosh-my-instance-guid"))

			releaseName, _ := fakeHelmClientFactory.fakeHelmClient.DeleteReleaseArgsForCall(0)
			Expect(releaseName).To(Equal("kibosh-my-instance-guid"))

			Expect(err).To(BeNil())
			Expect(response.IsAsync).To(BeTrue())
			Expect(response.OperationData).To(Equal("deprovision"))
		})
	})

	Context("update", func() {
		var fakeHelmClientFactory *FakeHelmClientFactory
		var fakeClusterFactory *FakeClusterFactory
		var fakeServiceInstallerAccountInstallerFactory *FakeServiceAccountInstallerFactory
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClientFactory = NewFakeHelmClientFactory()
			fakeClusterFactory = NewFakeClusterFactory()
			fakeServiceInstallerAccountInstallerFactory = NewFakeServiceAccountInstallerFactory()

			broker = NewPksServiceBroker(config, fakeClusterFactory, fakeHelmClientFactory, fakeServiceInstallerAccountInstallerFactory, charts, logger)
		})

		It("requires async", func() {
			resp, err := broker.Update(nil, "my-instance-guid", brokerapi.UpdateDetails{}, true)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
			Expect(resp.OperationData).To(Equal("update"))
		})

		It("responds correctly", func() {
			raw := json.RawMessage(`{"foo":"bar"}`)

			details := brokerapi.UpdateDetails{
				PlanID:        "my-plan-id",
				ServiceID:     spacebearsServiceGUID,
				RawParameters: raw,
			}

			resp, err := broker.Update(nil, "my-instance-guid", details, true)

			chart, namespaceName, plan, opts := fakeHelmClientFactory.fakeHelmClient.UpdateChartArgsForCall(0)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
			Expect(resp.OperationData).To(Equal("update"))
			Expect(fakeHelmClientFactory.fakeHelmClient.UpdateChartCallCount()).To(Equal(1))

			Expect(chart).To(Equal(spacebearsChart))
			Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
			Expect(plan).To(Equal("my-plan-id"))
			Expect(strings.TrimSpace(string(opts))).To(Equal("foo: bar"))

		})
	})
})
