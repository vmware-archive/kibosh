package broker_test

import (
	"code.cloudfoundry.org/lager"
	"io/ioutil"
	"os"
	"path/filepath"

	"errors"
	. "github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/helm/helmfakes"
	"github.com/cf-platform-eng/kibosh/k8s/k8sfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
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
	})
})
