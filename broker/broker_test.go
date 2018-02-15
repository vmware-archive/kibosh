package broker_test

import (
	"code.cloudfoundry.org/lager"
	"io/ioutil"
	"os"
	"path/filepath"

	"encoding/json"
	"errors"
	. "github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/k8s/k8sfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	hapi_services "k8s.io/helm/pkg/proto/hapi/services"
)

var _ = Describe("Broker", func() {
	var logger lager.Logger

	chartYamlContent := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	BeforeEach(func() {
		logger = lager.NewLogger("test")
	})

	Context("catalog", func() {
		It("Provides a catalog", func() {
			// Create a temporary directory (defer removing it until after method)
			dir, err := ioutil.TempDir("", "chart-")
			Expect(err).To(BeNil())
			defer os.RemoveAll(dir)

			// Create a temporary file in that directory with chart yaml.
			tmpfn := filepath.Join(dir, "Chart.yaml")
			err = ioutil.WriteFile(tmpfn, chartYamlContent, 0666)
			Expect(err).To(BeNil())

			// Create the service catalog using that yaml.
			serviceBroker := NewPksServiceBroker(dir, "service-id", nil, nil, logger)
			serviceCatalog := serviceBroker.Services(nil)

			// Evalute correctness of service catalog against expected.
			expectedPlan := []brokerapi.ServicePlan{{
				ID:          "service-id-default",
				Name:        "default",
				Description: "spacebears service and spacebears broker helm chart",
			}}
			expectedCatalog := []brokerapi.Service{{
				ID:          "service-id",
				Name:        "spacebears",
				Description: "spacebears service and spacebears broker helm chart",
				Bindable:    true,
				Plans:       expectedPlan,
			}}
			Expect(expectedCatalog).Should(Equal(serviceCatalog))

		})

		It("Throws an appropriate error", func() {
			serviceBroker := NewPksServiceBroker("unknown", "service-id", nil, nil, logger)

			Expect(func() {
				serviceBroker.Services(nil)
			}).To(Panic())

		})
	})

	Context("provision", func() {
		var fakeMyHelmClient *helmfakes.FakeMyHelmClient
		var fakeCluster *k8sfakes.FakeCluster
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeMyHelmClient = &helmfakes.FakeMyHelmClient{}
			fakeCluster = &k8sfakes.FakeCluster{}

			broker = NewPksServiceBroker("/my/chart/dir", "service-id", fakeCluster, fakeMyHelmClient, logger)
		})

		It("requires async", func() {
			_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, false)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("async"))
		})

		It("creates a new namespace", func() {
			_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, true)

			Expect(err).To(BeNil())

			Expect(fakeCluster.CreateNamespaceCallCount()).To(Equal(1))

			namespace := fakeCluster.CreateNamespaceArgsForCall(0)
			Expect(namespace.Name).To(Equal("kibosh-my-instance-guid"))
		})

		It("returns error on namespace creation failure", func() {
			errorMessage := "namespace already taken or something"
			fakeCluster.CreateNamespaceReturns(nil, errors.New(errorMessage))

			_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, true)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})

		It("creates helm chart", func() {
			_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, true)

			Expect(err).To(BeNil())

			Expect(fakeMyHelmClient.InstallReleaseFromDirCallCount()).To(Equal(1))

			Expect(fakeMyHelmClient.InstallReleaseFromDirCallCount()).To(Equal(1))
			chartDir, namespaceName, opts := fakeMyHelmClient.InstallReleaseFromDirArgsForCall(0)
			Expect(chartDir).To(Equal("/my/chart/dir"))
			Expect(namespaceName).To(Equal("kibosh-my-instance-guid"))
			Expect(opts).To(HaveLen(1))
		})

		It("returns error on helm chart creation failure", func() {
			errorMessage := "no helm for you"
			fakeMyHelmClient.InstallReleaseFromDirReturns(nil, errors.New(errorMessage))

			_, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, true)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(errorMessage))
		})

		It("responds correctly", func() {
			resp, err := broker.Provision(nil, "my-instance-guid", brokerapi.ProvisionDetails{}, true)

			Expect(err).To(BeNil())
			Expect(resp.IsAsync).To(BeTrue())
		})
	})

	Context("last operation", func() {
		var fakeMyHelmClient *helmfakes.FakeMyHelmClient
		var fakeCluster *k8sfakes.FakeCluster
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeMyHelmClient = &helmfakes.FakeMyHelmClient{}
			fakeCluster = &k8sfakes.FakeCluster{}

			broker = NewPksServiceBroker("/my/chart/dir", "service-id", fakeCluster, fakeMyHelmClient, logger)

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
		})

		It("elevates error from helm", func() {
			errMessage := "helm communication failure or something"
			fakeMyHelmClient.ReleaseStatusReturns(nil, errors.New(errMessage))

			_, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring(errMessage))
		})

		It("returns success if deployed", func() {
			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("succeeded"))
			Expect(resp.State).To(Equal(brokerapi.Succeeded))
		})

		It("returns pending install", func() {
			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_PENDING_INSTALL,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("in progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("returns pending upgrade", func() {
			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_PENDING_UPGRADE,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("in progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("returns failed", func() {
			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_FAILED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-inststance-guid", "???")

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

			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			resp, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).To(BeNil())
			Expect(resp.Description).To(ContainSubstring("progress"))
			Expect(resp.State).To(Equal(brokerapi.InProgress))
		})

		It("bubbles up error on list service failure", func() {
			fakeCluster.ListServicesReturns(&api_v1.ServiceList{}, errors.New("no services for you"))

			fakeMyHelmClient.ReleaseStatusReturns(&hapi_services.GetReleaseStatusResponse{
				Info: &hapi_release.Info{
					Status: &hapi_release.Status{
						Code: hapi_release.Status_DEPLOYED,
					},
				},
			}, nil)

			_, err := broker.LastOperation(nil, "my-inststance-guid", "???")

			Expect(err).NotTo(BeNil())
		})
	})

	Context("bind", func() {
		var fakeMyHelmClient *helmfakes.FakeMyHelmClient
		var fakeCluster *k8sfakes.FakeCluster
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeMyHelmClient = &helmfakes.FakeMyHelmClient{}
			fakeCluster = &k8sfakes.FakeCluster{}

			broker = NewPksServiceBroker("/my/chart/dir", "service-id", fakeCluster, fakeMyHelmClient, logger)
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
	})
	Context("delete", func() {
		var fakeMyHelmClient *helmfakes.FakeMyHelmClient
		var fakeCluster *k8sfakes.FakeCluster
		var broker *PksServiceBroker

		BeforeEach(func() {
			fakeMyHelmClient = &helmfakes.FakeMyHelmClient{}
			fakeCluster = &k8sfakes.FakeCluster{}

			broker = NewPksServiceBroker("/my/chart/dir", "service-id", fakeCluster, fakeMyHelmClient, logger)
		})

		It("bubbles up delete chart errors", func() {
			fakeMyHelmClient.DeleteReleaseReturns(nil, errors.New("Failed"))

			details := brokerapi.DeprovisionDetails {
				PlanID:"my-plan-id",
				ServiceID:"my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeMyHelmClient.DeleteReleaseCallCount()).To(Equal(1))

		})

		It("bubbles up delete namespace errors", func() {
			fakeCluster.DeleteNamespaceReturns(errors.New("nope"))

			details := brokerapi.DeprovisionDetails {
				PlanID:"my-plan-id",
				ServiceID:"my-service-id",
			}
			_, err := broker.Deprovision(nil, "my-instance-guid", details, true)

			Expect(err).NotTo(BeNil())
			Expect(fakeCluster.DeleteNamespaceCallCount()).To(Equal(1))

		})

		It("correctly calls deletion", func() {
			details := brokerapi.DeprovisionDetails {
				PlanID:"my-plan-id",
				ServiceID:"my-service-id",
			}
			broker.Deprovision(nil, "my-instance-guid", details, true)

			namespace, _ := fakeCluster.DeleteNamespaceArgsForCall(0)
			Expect(namespace).To(Equal("kibosh-my-instance-guid"))

			releaseName, _ := fakeMyHelmClient.DeleteReleaseArgsForCall(0)
			Expect(releaseName).To(Equal("kibosh-my-instance-guid"))
		})

	})
})
