package helm_test

//import (
//	. "github.com/onsi/ginkgo"
//	. "github.com/onsi/gomega"
//
//	"code.cloudfoundry.org/lager"
//	"errors"
//	. "github.com/cf-platform-eng/kibosh/helm"
//	"github.com/cf-platform-eng/kibosh/helm/helmfakes"
//	"github.com/cf-platform-eng/kibosh/k8s/k8sfakes"
//	"github.com/cf-platform-eng/kibosh/test"
//	api_errors "k8s.io/apimachinery/pkg/api/errors"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"time"
//)
//
//var _ = Describe("KubeConfig", func() {
//	var logger lager.Logger
//	var cluster k8sfakes.FakeCluster
//	var client helmfakes.FakeMyHelmClient
//	var installer Installer
//
//	BeforeEach(func() {
//		logger = lager.NewLogger("test")
//		k8sClient := test.FakeK8sInterface{}
//		cluster = k8sfakes.FakeCluster{}
//		cluster.GetClientReturns(&k8sClient)
//		client = helmfakes.FakeMyHelmClient{}
//
//		installer = NewInstaller(&cluster, &client, logger)
//	})
//
//	It("success", func() {
//		err := installer.Install()
//
//		Expect(err).To(BeNil())
//
//		Expect(client.InstallCallCount()).To(Equal(1))
//		Expect(client.UpgradeCallCount()).To(Equal(0))
//
//		opts := client.InstallArgsForCall(0)
//		Expect(opts.Namespace).To(Equal("kube-system"))
//		Expect(opts.ImageSpec).To(Equal("gcr.io/kubernetes-helm/tiller:v2.6.1"))
//	})
//
//	It("upgrade required", func() {
//		client.InstallReturns(api_errors.NewAlreadyExists(schema.GroupResource{}, ""))
//
//		err := installer.Install()
//
//		Expect(err).To(BeNil())
//		Expect(client.InstallCallCount()).To(Equal(1))
//		Expect(client.UpgradeCallCount()).To(Equal(1))
//	})
//
//	It("blocks on error", func() {
//		client.ListReleasesReturnsOnCall(0, nil, errors.New("broker"))
//		client.ListReleasesReturnsOnCall(1, nil, errors.New("broker"))
//		client.ListReleasesReturnsOnCall(2, nil, nil)
//		installer.SetMaxWait(1 * time.Millisecond)
//
//		err := installer.Install()
//
//		Expect(client.ListReleasesCallCount()).To(Equal(3))
//		Expect(err).To(BeNil())
//	})
//
//	It("returns error if helm doesn't become healthy", func() {
//		client.ListReleasesReturns(nil, errors.New("No helm for you"))
//		installer.SetMaxWait(1 * time.Millisecond)
//
//		err := installer.Install()
//
//		Expect(err).NotTo(BeNil())
//	})
//})
