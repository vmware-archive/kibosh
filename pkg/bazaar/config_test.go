package bazaar_test

import (
	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	BeforeEach(func() {
		os.Clearenv()

		os.Setenv("SECURITY_USER_NAME", "bob")
		os.Setenv("SECURITY_USER_PASSWORD", "abc123")
		os.Setenv("PORT", "9001")
		os.Setenv("HELM_CHART_DIR", "/home/somewhere")
		os.Setenv("KIBOSH_SERVER", "mykibosh.com")
		os.Setenv("KIBOSH_USER_NAME", "kevin")
		os.Setenv("KIBOSH_USER_PASSWORD", "monkey123")
	})

	It("parses config from environment", func() {
		c, err := bazaar.ParseConfig()
		Expect(err).To(BeNil())
		Expect(c.AdminUsername).To(Equal("bob"))
		Expect(c.AdminPassword).To(Equal("abc123"))
		Expect(c.HelmChartDir).To(Equal("/home/somewhere"))
		Expect(c.Port).To(Equal(9001))
		Expect(c.KiboshConfig.Server).To(Equal("mykibosh.com"))
		Expect(c.KiboshConfig.User).To(Equal("kevin"))
		Expect(c.KiboshConfig.Pass).To(Equal("monkey123"))

	})

})
