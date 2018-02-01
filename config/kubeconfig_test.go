package config_test

import (
	"github.com/cf-platform-eng/kibosh/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

var _ = Describe("KubeConfig", func() {
	BeforeEach(func() {
		os.Clearenv()
		os.Setenv("VCAP_SERVICES", valid_vcap_services)
	})

	It("parses service block", func() {
		kuboODBService, err := config.ParseVCAPServices("user-provided")

		Expect(err).To(BeNil())

		Expect(kuboODBService.Name).To(Equal("my-kubernetes"))
		Expect(kuboODBService.Credentials.KubeConfig.ApiVersion).To(Equal("v1"))

		Expect(kuboODBService.Credentials.KubeConfig.Clusters).To(HaveLen(1))
		Expect(kuboODBService.Credentials.KubeConfig.Clusters[0].ClusterInfo.Server).To(Equal("https://example.com:33071"))
		Expect(kuboODBService.Credentials.KubeConfig.Clusters[0].ClusterInfo.CAData).To(Equal("bXktZmFrZWNlcnQ="))

		Expect(kuboODBService.Credentials.KubeConfig.Users).To(HaveLen(1))
		Expect(kuboODBService.Credentials.KubeConfig.Users[0].Name).To(Equal("7dd1424e-c44c-4090-ae0d-3c92a8abe52d"))
		Expect(kuboODBService.Credentials.KubeConfig.Users[0].UserCredentials.Token).To(Equal("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ"))
	})

	It("decode ca data", func() {
		kuboODBService, err := config.ParseVCAPServices("user-provided")

		Expect(err).To(BeNil())

		data, err := kuboODBService.Credentials.KubeConfig.Clusters[0].ClusterInfo.DecodeCAData()

		Expect(err).To(BeNil())

		Expect(data).To(Equal([]byte("my-fakecert")))
	})

	It("returns error on bad json", func() {
		os.Setenv("VCAP_SERVICES", "{!,")
		_, err := config.ParseVCAPServices("kubo-odb")

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("VCAP_SERVICES"))
	})

	It("returns error on bad cd data", func() {
		kuboODBService, err := config.ParseVCAPServices("user-provided")

		Expect(err).To(BeNil())

		kuboODBService.Credentials.KubeConfig.Clusters[0].ClusterInfo.CAData = "not base 64:!//"

		_, err = kuboODBService.Credentials.KubeConfig.Clusters[0].ClusterInfo.DecodeCAData()

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("decode"))
	})
})
