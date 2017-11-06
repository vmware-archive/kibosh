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
			os.Setenv("VCAP_SERVICES", valid_vcap_services)
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

			Expect(c.KuboODBVCAP).NotTo(BeNil())
			Expect(c.KuboODBVCAP.Name).To(Equal("my-kubernetes"))
			Expect(c.KuboODBVCAP.Credentials.KubeConfig.ApiVersion).To(Equal("v1"))
		})

		It("check for required password", func() {
			_, err := Parse()
			Expect(err).NotTo(BeNil())
		})
	})
})
