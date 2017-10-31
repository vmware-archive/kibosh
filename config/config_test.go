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
		})

		It("parses config from environment", func() {
			os.Setenv("ADMIN_USERNAME", "bob")
			os.Setenv("ADMIN_PASSWORD", "abc123")
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
		})

		It("check for required password", func() {
			_, err := Parse()
			Expect(err).NotTo(BeNil())
		})
	})
})
