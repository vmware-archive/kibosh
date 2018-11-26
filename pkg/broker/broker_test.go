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

	"github.com/cf-platform-eng/kibosh/pkg/state"

	"github.com/Sirupsen/logrus"
	. "github.com/cf-platform-eng/kibosh/pkg/broker"
	my_config "github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/pkg/k8s/k8sfakes"
	"github.com/cf-platform-eng/kibosh/pkg/state/statefakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sAPI "k8s.io/client-go/tools/clientcmd/api"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	hapi_services "k8s.io/helm/pkg/proto/hapi/services"
)

var _ = Describe("Broker", func() {
	const spacebearsServiceGUID = "37b7acb6-6755-56fe-a17f-2307657023ef"
	const mysqlServiceGUID = "c76ed0a4-9a04-5710-90c2-75e955697b08"

	var logger *logrus.Logger

	var spacebearsChart *my_helm.MyChart
	var mysqlChart *my_helm.MyChart
	var charts []*my_helm.MyChart

	var config *my_config.Config

	BeforeEach(func() {
		logger = logrus.New()
		config = &my_config.Config{
			RegistryConfig: &my_config.RegistryConfig{
				Server: "127.0.0.1",
				User:   "k8s",
				Pass:   "monkey123",
				Email:  "k8s@example.com"},
			HelmTLSConfig: &my_config.HelmTLSConfig{},
		}
		bullets := []string{"bullet1 for plan for spacebears", "bullet2 for plan for spacebears"}
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
					Free:        brokerapi.FreeValue(true),
					Bindable:    brokerapi.BindableValue(true),
				},
				"medium": {
					Name:        "medium",
					Description: "medium plan for spacebears",
					File:        "medium.yaml",
					Bullets:     bullets,
					Free:        brokerapi.FreeValue(false),
					Bindable:    brokerapi.BindableValue(true),
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
					Free:        brokerapi.FreeValue(true),
					Bindable:    brokerapi.BindableValue(true),
				},
				"medium": {
					Name:        "big",
					Description: "big data",
					File:        "big.yaml",
					Free:        brokerapi.FreeValue(false),
					Bindable:    brokerapi.BindableValue(false),
				},
			},
		}

		charts = []*my_helm.MyChart{spacebearsChart, mysqlChart}
	})

	Context("catalog", func() {
		It("Provides a catalog with correct service", func() {
			serviceBroker := NewPksServiceBroker(config, nil, nil, nil, charts, nil, nil, logger)
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
			serviceBroker := NewPksServiceBroker(config, nil, nil, nil, charts, nil, nil, logger)
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
					Free:     brokerapi.FreeValue(true),
					Bindable: brokerapi.BindableValue(true),
				},
				{
					ID:          "37b7acb6-6755-56fe-a17f-2307657023ef-medium",
					Name:        "medium",
					Description: "medium plan for spacebears",
					Metadata: &brokerapi.ServicePlanMetadata{
						DisplayName: "medium",
						Bullets: []string{
							"bullet1 for plan for spacebears",
							"bullet2 for plan for spacebears",
						},
					},
					Free:     brokerapi.FreeValue(false),
					Bindable: brokerapi.BindableValue(true),
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
		var fakeHelmClient helmfakes.FakeMyHelmClient
		var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
		var fakeCluster k8sfakes.FakeCluster
		var fakeClusterFactory k8sfakes.FakeClusterFactory
		var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
		var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
		var fakeBrokerState statefakes.FakeKeyValueStore
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClient = helmfakes.FakeMyHelmClient{}
			fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
			fakeClusterFactory = k8sfakes.FakeClusterFactory{}
			fakeCluster = k8sfakes.FakeCluster{}
			fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
			fakeClusterFactory.GetClusterReturns(&fakeCluster, nil)
			fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
			fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
			fakeBrokerState = statefakes.FakeKeyValueStore{}
			details = brokerapi.ProvisionDetails{
				ServiceID: spacebearsServiceGUID,
			}

			broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)
			Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
			Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(0))

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

		It("uses the default cluster", func() {
			_, err := broker.Provision(nil, "my-instance-guid", details, true)

			Expect(err).To(BeNil())
			Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(1))
			Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(0))
		})

		Context("cluster targeting", func() {
			BeforeEach(func() {
				details.RawParameters = []byte(`{"clusterConfig" : {"server": "server url", "token":"token data", "certificateAuthorityData":"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURKRENDQWd5Z0F3SUJBZ0lVVHltSk1uWEU0aXp5QlBvalM1enpXU2tzNmVrd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0RURUxNQWtHQTFVRUF4TUNZMkV3SGhjTk1UZ3dOVEE0TVRneU16SXdXaGNOTVRrd05UQTRNVGd5TXpJdwpXakFOTVFzd0NRWURWUVFERXdKallUQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCCkFLNGd3ZnFpeG1KT1JjSmtERjZVdmNVQWl5TEdlRks1Z0JnaEU3MVFzMnZ4UU4vT1AxekVHMkVSWHZheFIzbUYKVFdxTkNlTTl5d1Fpcm9FTmtFODd2LzFBZXZudkJQMDVZczdmaU5pS0ZNZTRYV091UWRlNXR0S3JpdlpJRWtCawpTT2psdXlQR0g4d3JTY0J1alZQelQvMGxLR3FKUW1iTGFTVm1qczczK0NINFpEZ2ZILzN0c0tpQTRaWU54Z2JGCnY0WWRnTFJHSkdTZjN5NlhyaWoxaVpMaUdhYjVWbDVLUSs2T0ZqR0wxbEEybyt4SGV2d2J0T2hQdlB1emNYbHkKd1RJay8rSEw0ckRMOG9Oemg5QTFldlFCaDJHRzhUSmFQMXVldVVXSXlHZENyb2FqTUZMN1ZaZkd1aUZLK2UyUwplODNYNHdzakJSc0t6RFlKL21IcjE5a0NBd0VBQWFOOE1Ib3dIUVlEVlIwT0JCWUVGSDdsRlJWSFZSeXFYQkhJClQ1cGdJVmRtM0JmcU1FZ0dBMVVkSXdSQk1EK0FGSDdsRlJWSFZSeXFYQkhJVDVwZ0lWZG0zQmZxb1JHa0R6QU4KTVFzd0NRWURWUVFERXdKallZSVVUeW1KTW5YRTRpenlCUG9qUzV6eldTa3M2ZWt3RHdZRFZSMFRBUUgvQkFVdwpBd0VCL3pBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUs2SUtJdGJOMVFaR2pMWUZsTU1KbDJrcVFCNG9KekgyClZLczhxTCtMZDJHVHNIWDVxKzFhV2ZLcVRha0V1QVdwTGZYWUZEUXc3TXYyak9rZGQ0WEV6MXEwZ3k1QWw1MVYKZnlYaXBJalBMdFYwK21DdGVkc2hFNHJZVjZvQUp4RFE2MzJ2b3JpWVJpR3A3SHVqL254VjFMbmUwQzlQdmI0UAp1NVYrZGxQRHZSR3J2Y1dtNjk4bC9PQncyNk9GcHFCQytBUExteW5SMDBXL2xQQURHOWpaT0ZiblNlRGFPMkhqClcwQzhzb3QrYkZianFsaU01T2hBU0RwOFI2VHBqU1hWNEFqZzE5blMxM1M0bVZVSGFtOXJOTkw4aWVhdVdVMUUKdUVaUFBNb0hHcGlZQ29CelEwYmdqL0xaVDR1YzVlZ1Mrb29XdUJTKzM0Mk1KcVFFa2NVRWFBPT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo="}}`)
			})

			It("targets the right cluster", func() {
				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())
				Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
				Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(1))
				clusterConfig := fakeClusterFactory.GetClusterArgsForCall(0)
				Expect(clusterConfig.Server).To(Equal("server url"))
				Expect(fakeBrokerState.PutJsonCallCount()).To(Equal(1))
				Expect(fakeBrokerState.PutJsonCallCount()).To(Equal(1))
				putKey, _ := fakeBrokerState.PutJsonArgsForCall(0)
				Expect(putKey).To(Equal("my-instance-guid-instance-to-cluster"))
			})

			It("creates service account", func() {
				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())

				Expect(fakeServiceAccountInstaller.InstallCallCount()).To(Equal(1))
			})

			It("returns error on service account installation failure", func() {
				errorMessage := "could not create service account"
				fakeServiceAccountInstaller.InstallReturns(errors.New(errorMessage))

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring(errorMessage))
			})

		})

		Context("Cluster config in plan", func() {
			It("happy path", func() {
				k8sConfig := &k8sAPI.Config{
					Clusters: map[string]*k8sAPI.Cluster{
						"cluster1": {
							CertificateAuthorityData: []byte("my cat"),
							Server:                   "myserver",
						},
						"cluster2": {
							CertificateAuthorityData: []byte("my cat"),
							Server:                   "myserver",
						},
					},
					CurrentContext: "context2",
					Contexts: map[string]*k8sAPI.Context{
						"context1": {
							Cluster:  "cluster1",
							AuthInfo: "auth1",
						},
						"context2": {
							Cluster:  "cluster2",
							AuthInfo: "auth2",
						},
					},
					AuthInfos: map[string]*k8sAPI.AuthInfo{
						"auth1": {
							Token: "myencoded token",
						},
						"auth2": {
							Token: "myencoded 2nd token",
						},
					},
				}

				plan := spacebearsChart.Plans["small"]
				plan.ClusterConfig = k8sConfig
				spacebearsChart.Plans["small"] = plan

				details = brokerapi.ProvisionDetails{
					ServiceID: spacebearsServiceGUID,
					PlanID:    spacebearsServiceGUID + "-small",
				}

				broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())
				clusterUsed := fakeHelmClientFactory.HelmClientArgsForCall(0)

				Expect(clusterUsed.GetClientConfig().BearerToken).To(Equal("myencoded 2nd token"))
			})

		})

		Context("registry secrets", func() {
			It("doesn't mess with secrets when not configured", func() {
				config = &my_config.Config{RegistryConfig: &my_config.RegistryConfig{}, HelmTLSConfig: &my_config.HelmTLSConfig{}}
				broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, nil, logger)

				_, err := broker.Provision(nil, "my-instance-guid", details, true)

				Expect(err).To(BeNil())

				Expect(fakeCluster.UpdateSecretCallCount()).To(Equal(0))
				Expect(fakeCluster.PatchCallCount()).To(Equal(0))
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

				Expect(fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				_, namespace, chart, plan, opts := fakeHelmClient.InstallChartArgsForCall(0)
				Expect(chart).To(Equal(spacebearsChart))
				Expect(namespace.Name).To(Equal("kibosh-my-instance-guid"))
				Expect(plan).To(Equal("small"))
				Expect(opts).To(BeNil())
			})

			It("returns error on helm chart creation failure", func() {
				errorMessage := "no helm for you"
				fakeHelmClient.InstallChartReturns(nil, errors.New(errorMessage))

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

				Expect(fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				_, _, chart, _, _ := fakeHelmClient.InstallChartArgsForCall(0)
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

				Expect(fakeHelmClient.InstallChartCallCount()).To(Equal(1))
				_, namespace, chart, plan, opts := fakeHelmClient.InstallChartArgsForCall(0)
				Expect(chart).To(Equal(spacebearsChart))
				Expect(namespace.Name).To(Equal("kibosh-my-instance-guid"))
				Expect(plan).To(Equal("small"))
				Expect(strings.TrimSpace(string(opts))).To(Equal("foo: bar"))
			})
		})
	})

	Context("last operation", func() {
		var fakeHelmClient helmfakes.FakeMyHelmClient
		var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
		var fakeCluster k8sfakes.FakeCluster
		var fakeClusterFactory k8sfakes.FakeClusterFactory
		var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
		var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
		var fakeBrokerState statefakes.FakeKeyValueStore
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClient = helmfakes.FakeMyHelmClient{}
			fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
			fakeCluster = k8sfakes.FakeCluster{}
			fakeClusterFactory = k8sfakes.FakeClusterFactory{}
			fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
			fakeClusterFactory.GetClusterReturns(&fakeCluster, nil)
			fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
			fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
			fakeBrokerState = statefakes.FakeKeyValueStore{}
			fakeBrokerState.GetJsonReturns(state.KeyNotFoundError)

			broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)

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
			fakeCluster.ListServicesReturns(&serviceList, nil)
			podList := api_v1.PodList{
				Items: []api_v1.Pod{},
			}
			fakeCluster.ListPodsReturns(&podList, nil)

		})

		It("elevates error from helm", func() {
			errMessage := "helm communication failure or something"
			fakeHelmClient.ReleaseStatusReturns(nil, errors.New(errMessage))

			_, err := broker.LastOperation(nil, "my-instance-guid", "???")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(errMessage))
		})

		It("returns success if deployed", func() {
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			Expect(fakeClusterFactory.DefaultClusterCallCount()).ShouldNot(Equal(0))
			Expect(fakeBrokerState.GetJsonCallCount()).To(Equal(fakeClusterFactory.DefaultClusterCallCount()))
		})

		It("returns pending install", func() {
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
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
			fakeCluster.ListPodsReturns(&podList, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.State).To(Equal(brokerapi.InProgress))
			Expect(resp.Description).To(ContainSubstring("0/1 nodes are available: 1 Insufficient memory"))
		})

		It("considers a pod status of Completed as meaning the pod succeeded", func() {
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			podList := api_v1.PodList{
				Items: []api_v1.Pod{
					{
						ObjectMeta: meta_v1.ObjectMeta{
							Name: "pod1",
							Labels: map[string]string{
								"job-name": "test",
							},
						},
						Spec: api_v1.PodSpec{},
						Status: api_v1.PodStatus{
							Phase: "Succeeded",
						},
					},
				},
			}
			fakeCluster.ListPodsReturns(&podList, nil)

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
		})

		It("returns error when unable to list pods", func() {
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			fakeCluster.ListPodsReturns(nil, errors.New("nope"))

			_, err := broker.LastOperation(nil, "my-instance-guid", "provision")
			Expect(err).NotTo(BeNil())
		})

		It("bubbles up error on list service failure", func() {
			fakeCluster.ListServicesReturns(&api_v1.ServiceList{}, errors.New("no services for you"))

			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			_, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).NotTo(BeNil())
		})

		It("targets the proper cluster", func() {
			fakeHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)
			fakeBrokerState.GetJsonStub = func(key string, value interface{}) error {
				if key == "my-instance-guid-instance-to-cluster" {
					json.Unmarshal([]byte(`{"clusterCredentials":{"caDataRaw":"some data","server":"some server", "token":"some token"}}`), value)
					return nil
				}
				return state.KeyNotFoundError
			}

			resp, err := broker.LastOperation(nil, "my-instance-guid", "provision")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("succeeded"))
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
			Expect(fakeClusterFactory.GetClusterCallCount()).ShouldNot(Equal(0))
			clusterCreds := fakeClusterFactory.GetClusterArgsForCall(0)
			Expect(clusterCreds.Server).To(Equal("some server"))
			Expect(clusterCreds.Token).To(Equal("some token"))
			Expect(clusterCreds.CADataRaw).To(Equal("some data"))
			Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
		})
	})

	Context("bind", func() {
		var fakeHelmClient helmfakes.FakeMyHelmClient
		var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
		var fakeCluster k8sfakes.FakeCluster
		var fakeClusterFactory k8sfakes.FakeClusterFactory
		var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
		var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
		var fakeBrokerState statefakes.FakeKeyValueStore
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClient = helmfakes.FakeMyHelmClient{}
			fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
			fakeCluster = k8sfakes.FakeCluster{}
			fakeClusterFactory = k8sfakes.FakeClusterFactory{}
			fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
			fakeClusterFactory.GetClusterReturns(&fakeCluster, nil)
			fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
			fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
			fakeBrokerState = statefakes.FakeKeyValueStore{}
			fakeBrokerState.GetJsonReturns(state.KeyNotFoundError)

			broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)
		})

		It("bind returns cluster secrets", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					},
				},
			}
			fakeCluster.ListSecretsReturns(&secretsList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeCluster.ListSecretsCallCount()).To(Equal(1))

			Expect(binding).NotTo(BeNil())

			creds := binding.Credentials
			secrets := creds.(map[string]interface{})["secrets"]
			secretsJson, err := json.Marshal(secrets)
			Expect(string(secretsJson)).To(Equal(`[{"data":{"db-password":"abc123"},"name":"passwords"}]`))
		})

		// NodePort Test
		It("bind returns externalIPs field when Service Type NodePort is used", func() {
			nodeList := api_v1.NodeList{
				Items: []api_v1.Node{
					{
						ObjectMeta: meta_v1.ObjectMeta{
							Labels: map[string]string{
								"spec.ip": "1.1.1.1",
							},
						},
					},
				},
			}
			secretsList := api_v1.SecretList{
				Items: []api_v1.Secret{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "passwords"},
						Data:       map[string][]byte{"db-password": []byte("abc123")},
						Type:       api_v1.SecretTypeOpaque,
					},
				},
			}
			serviceList := api_v1.ServiceList{
				Items: []api_v1.Service{
					{
						ObjectMeta: meta_v1.ObjectMeta{Name: "kibosh-my-mysql-db-instance"},
						Spec: api_v1.ServiceSpec{
							Ports: []api_v1.ServicePort{},
							Type:  "NodePort",
						},
					},
				},
			}
			fakeCluster.ListNodesReturns(&nodeList, nil)
			fakeCluster.ListServicesReturns(&serviceList, nil)
			fakeCluster.ListSecretsReturns(&secretsList, nil)
			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})
			Expect(err).To(BeNil())
			creds := binding.Credentials

			services := creds.(map[string]interface{})["services"]
			spec := services.([]map[string]interface{})[0]["spec"]
			externalIPs := spec.(api_v1.ServiceSpec).ExternalIPs
			Expect(externalIPs[0]).To(Equal("1.1.1.1"))
		})
		//
		It("bind filters to only opaque secrets", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeCluster.ListServicesReturns(&serviceList, nil)

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
			fakeCluster.ListSecretsReturns(&secretsList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeCluster.ListSecretsCallCount()).To(Equal(1))

			Expect(binding).NotTo(BeNil())

			creds := binding.Credentials
			secrets := creds.(map[string]interface{})["secrets"]
			secretsJson, err := json.Marshal(secrets)
			Expect(string(secretsJson)).To(Equal(`[{"data":{"db-password":"abc123"},"name":"passwords"}]`))
		})

		It("bubbles up list secrets errors", func() {
			serviceList := api_v1.ServiceList{Items: []api_v1.Service{}}
			fakeCluster.ListServicesReturns(&serviceList, nil)

			fakeCluster.ListSecretsReturns(nil, errors.New("foo failed"))
			_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).NotTo(BeNil())
			Expect(fakeCluster.ListSecretsCallCount()).To(Equal(1))
		})

		It("returns services (load balancer stuff)", func() {
			secretList := api_v1.SecretList{Items: []api_v1.Secret{}}
			fakeCluster.ListSecretsReturns(&secretList, nil)

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
			fakeCluster.ListServicesReturns(&serviceList, nil)

			binding, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).To(BeNil())
			Expect(fakeCluster.ListServicesCallCount()).To(Equal(1))

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
			fakeCluster.ListSecretsReturns(&secretList, nil)

			fakeCluster.ListServicesReturns(nil, errors.New("no services for you"))

			_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

			Expect(err).NotTo(BeNil())
		})

		Describe("uses proper cluster", func() {
			var secretList api_v1.SecretList
			var serviceList api_v1.ServiceList

			BeforeEach(func() {
				secretList = api_v1.SecretList{Items: []api_v1.Secret{}}
				fakeCluster.ListSecretsReturns(&secretList, nil)

				serviceList = api_v1.ServiceList{
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
				fakeCluster.ListServicesReturns(&serviceList, nil)
			})

			It("connects to default cluster", func() {
				_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

				Expect(err).To(BeNil())
				Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(0))
				Expect(fakeClusterFactory.DefaultClusterCallCount()).ShouldNot(Equal(0))
			})

			It("connects to alternate cluster", func() {
				fakeBrokerState.GetJsonStub = func(key string, value interface{}) error {
					if key == "my-instance-id-instance-to-cluster" {
						json.Unmarshal([]byte(`{"clusterCredentials":{"caDataRaw":"some data","server":"some server", "token":"some token"}}`), value)
						return nil
					}
					return state.KeyNotFoundError
				}

				_, err := broker.Bind(nil, "my-instance-id", "my-binding-id", brokerapi.BindDetails{})

				Expect(err).To(BeNil())
				Expect(fakeClusterFactory.GetClusterCallCount()).ShouldNot(Equal(0))
				clusterCreds := fakeClusterFactory.GetClusterArgsForCall(0)
				Expect(clusterCreds.Server).To(Equal("some server"))
				Expect(clusterCreds.Token).To(Equal("some token"))
				Expect(clusterCreds.CADataRaw).To(Equal("some data"))
				Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
			})
		})
	})

	Context("delete", func() {
		var fakeHelmClient helmfakes.FakeMyHelmClient
		var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
		var fakeCluster k8sfakes.FakeCluster
		var fakeClusterFactory k8sfakes.FakeClusterFactory
		var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
		var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
		var fakeBrokerState statefakes.FakeKeyValueStore
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClient = helmfakes.FakeMyHelmClient{}
			fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
			fakeCluster = k8sfakes.FakeCluster{}
			fakeClusterFactory = k8sfakes.FakeClusterFactory{}
			fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
			fakeClusterFactory.GetClusterReturns(&fakeCluster, nil)
			fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
			fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
			fakeBrokerState = statefakes.FakeKeyValueStore{}
			fakeBrokerState.GetJsonReturns(state.KeyNotFoundError)

			broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)
		})

		It("bubbles up delete chart errors", func() {
			fakeHelmClient.DeleteReleaseReturns(nil, errors.New("Failed"))

			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeHelmClient.DeleteReleaseCallCount()).To(Equal(1))

		})

		It("bubbles up delete namespace errors", func() {
			fakeCluster.DeleteNamespaceReturns(errors.New("nope"))

			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeCluster.DeleteNamespaceCallCount()).To(Equal(1))

		})

		It("correctly calls deletion", func() {
			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			response, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			namespace, _ := fakeCluster.DeleteNamespaceArgsForCall(0)
			Expect(namespace).To(Equal("kibosh-my-instance-guid"))

			releaseName, _ := fakeHelmClient.DeleteReleaseArgsForCall(0)
			Expect(releaseName).To(Equal("kibosh-my-instance-guid"))

			Expect(err).To(BeNil())
			Expect(response.IsAsync).To(BeTrue())
			Expect(response.OperationData).To(Equal("deprovision"))
			Expect(fakeClusterFactory.DefaultClusterCallCount()).ShouldNot(Equal(0))
			Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(0))
		})

		It("targets proper cluster", func() {
			details := brokerapi.DeprovisionDetails{
				PlanID:    "my-plan-id",
				ServiceID: "my-service-id",
			}
			fakeBrokerState.GetJsonStub = func(key string, value interface{}) error {
				if key == "my-instance-guid-instance-to-cluster" {
					json.Unmarshal([]byte(`{"clusterCredentials":{"caDataRaw":"some data","server":"some server", "token":"some token"}}`), value)
					return nil
				}
				return state.KeyNotFoundError
			}

			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).To(BeNil())
			Expect(fakeClusterFactory.GetClusterCallCount()).ShouldNot(Equal(0))
			clusterCreds := fakeClusterFactory.GetClusterArgsForCall(0)
			Expect(clusterCreds.Server).To(Equal("some server"))
			Expect(clusterCreds.Token).To(Equal("some token"))
			Expect(clusterCreds.CADataRaw).To(Equal("some data"))
			Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
		})
	})

	Context("update", func() {
		var fakeHelmClient helmfakes.FakeMyHelmClient
		var fakeHelmClientFactory helmfakes.FakeHelmClientFactory
		var fakeCluster k8sfakes.FakeCluster
		var fakeClusterFactory k8sfakes.FakeClusterFactory
		var fakeServiceAccountInstaller k8sfakes.FakeServiceAccountInstaller
		var fakeServiceAccountInstallerFactory k8sfakes.FakeServiceAccountInstallerFactory
		var fakeBrokerState statefakes.FakeKeyValueStore
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeHelmClient = helmfakes.FakeMyHelmClient{}
			fakeHelmClientFactory.HelmClientReturns(&fakeHelmClient)
			fakeCluster = k8sfakes.FakeCluster{}
			fakeClusterFactory = k8sfakes.FakeClusterFactory{}
			fakeClusterFactory.DefaultClusterReturns(&fakeCluster, nil)
			fakeClusterFactory.GetClusterReturns(&fakeCluster, nil)
			fakeServiceAccountInstaller = k8sfakes.FakeServiceAccountInstaller{}
			fakeServiceAccountInstallerFactory.ServiceAccountInstallerReturns(&fakeServiceAccountInstaller)
			fakeBrokerState = statefakes.FakeKeyValueStore{}
			fakeBrokerState.GetJsonReturns(state.KeyNotFoundError)

			broker = NewPksServiceBroker(config, &fakeClusterFactory, &fakeHelmClientFactory, &fakeServiceAccountInstallerFactory, charts, nil, &fakeBrokerState, logger)
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

			chart, namespaceName, plan, opts := fakeHelmClient.UpdateChartArgsForCall(0)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
			Expect(resp.OperationData).To(Equal("update"))
			Expect(fakeHelmClient.UpdateChartCallCount()).To(Equal(1))

			Expect(chart).To(Equal(spacebearsChart))
			Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
			Expect(plan).To(Equal("my-plan-id"))
			Expect(strings.TrimSpace(string(opts))).To(Equal("foo: bar"))
			Expect(fakeClusterFactory.DefaultClusterCallCount()).ShouldNot(Equal(0))
			Expect(fakeClusterFactory.GetClusterCallCount()).To(Equal(0))
		})

		It("targets proper cluster", func() {
			raw := json.RawMessage(`{"foo":"bar"}`)

			details := brokerapi.UpdateDetails{
				PlanID:        "my-plan-id",
				ServiceID:     spacebearsServiceGUID,
				RawParameters: raw,
			}

			fakeBrokerState.GetJsonStub = func(key string, value interface{}) error {
				if key == "my-instance-guid-instance-to-cluster" {
					json.Unmarshal([]byte(`{"clusterCredentials":{"caDataRaw":"some data","server":"some server", "token":"some token"}}`), value)
					return nil
				}
				return state.KeyNotFoundError
			}

			resp, err := broker.Update(nil, "my-instance-guid", details, true)

			chart, namespaceName, plan, opts := fakeHelmClient.UpdateChartArgsForCall(0)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
			Expect(resp.OperationData).To(Equal("update"))
			Expect(fakeHelmClient.UpdateChartCallCount()).To(Equal(1))

			Expect(chart).To(Equal(spacebearsChart))
			Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
			Expect(plan).To(Equal("my-plan-id"))
			Expect(strings.TrimSpace(string(opts))).To(Equal("foo: bar"))
			Expect(fakeClusterFactory.GetClusterCallCount()).ShouldNot(Equal(0))
			clusterCreds := fakeClusterFactory.GetClusterArgsForCall(0)
			Expect(clusterCreds.Server).To(Equal("some server"))
			Expect(clusterCreds.Token).To(Equal("some token"))
			Expect(clusterCreds.CADataRaw).To(Equal("some data"))
			Expect(fakeClusterFactory.DefaultClusterCallCount()).To(Equal(0))
		})
	})
})
