package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cf-platform-eng/kibosh/config"
)

var _ = Describe("Config", func() {
	Context("config parsing", func() {
		BeforeEach(func() {
			os.Clearenv()
			os.Setenv("CA_DATA", "c29tZSByYW5kb20gc3R1ZmY=")
			os.Setenv("SERVER", "127.0.0.1/api")
			os.Setenv("TOKEN", "my-token")
		})

		It("parses config from environment", func() {
			os.Setenv("SECURITY_USER_NAME", "bob")
			os.Setenv("SECURITY_USER_PASSWORD", "abc123")
			os.Setenv("PORT", "9001")
			os.Setenv("HELM_CHART_DIR", "/home/somewhere")
			os.Setenv("SERVICE_ID", "123")

			c, err := Parse()
			Expect(err).To(BeNil())
			Expect(c.AdminUsername).To(Equal("bob"))
			Expect(c.AdminPassword).To(Equal("abc123"))
			Expect(c.HelmChartDir).To(Equal("/home/somewhere"))
			Expect(c.ServiceID).To(Equal("123"))
			Expect(c.Port).To(Equal(9001))

			Expect(c.ClusterCredentials.CAData).To(Equal("c29tZSByYW5kb20gc3R1ZmY="))
			Expect(c.ClusterCredentials.Server).To(Equal("127.0.0.1/api"))
			Expect(c.ClusterCredentials.Token).To(Equal("my-token"))
		})

		It("check for required password", func() {
			_, err := Parse()
			Expect(err).NotTo(BeNil())
		})
	})
})
