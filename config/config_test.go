package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	. "github.com/cf-platform-eng/kibosh/config"
)

var _ = Describe("Config", func() {
	Context("config parsing", func() {
		BeforeEach(func() {
			os.Clearenv()
			os.Setenv("CA_DATA", "c29tZSByYW5kb20gc3R1ZmY=")
			os.Setenv("SERVER", "127.0.0.1/api")
			os.Setenv("TOKEN", "my-token")

			os.Setenv("SECURITY_USER_NAME", "bob")
			os.Setenv("SECURITY_USER_PASSWORD", "abc123")
			os.Setenv("PORT", "9001")
			os.Setenv("HELM_CHART_DIR", "/home/somewhere")
			os.Setenv("SERVICE_ID", "123")
		})

		It("parses config from environment", func() {
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

		It("parses cluster credentials", func() {
			c, err := Parse()
			Expect(err).To(BeNil())

			Expect(c.ClusterCredentials).NotTo(BeNil())
			Expect(c.ClusterCredentials.CAData).To(Equal("c29tZSByYW5kb20gc3R1ZmY="))
			Expect(c.ClusterCredentials.Server).To(Equal("127.0.0.1/api"))
			Expect(c.ClusterCredentials.Token).To(Equal("my-token"))
		})

		It("has registry config", func() {
			c, err := Parse()
			Expect(err).To(BeNil())

			Expect(c.RegistryConfig.HasRegistryConfig()).To(Equal(false))
		})

		It("errors trying to serialize reg config when not present", func() {
			c, err := Parse()
			Expect(err).To(BeNil())

			_, err = c.RegistryConfig.GetDockerConfigJson()
			Expect(err).NotTo(BeNil())
		})

		Context("with registry config", func() {
			BeforeEach(func() {
				os.Setenv("REG_SERVER", "https://127.0.0.1")
				os.Setenv("REG_USER", "k8s")
				os.Setenv("REG_PASS", "xyz789")
				os.Setenv("REG_EMAIL", "k8s@example.com")
			})

			It("parses registry config", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				Expect(c.RegistryConfig).NotTo(BeNil())
				Expect(c.RegistryConfig.Server).To(Equal("https://127.0.0.1"))
				Expect(c.RegistryConfig.User).To(Equal("k8s"))
				Expect(c.RegistryConfig.Pass).To(Equal("xyz789"))
				Expect(c.RegistryConfig.Email).To(Equal("k8s@example.com"))
			})

			It("serializes registry config", func() {
				c, err := Parse()
				Expect(err).To(BeNil())

				j, err := c.RegistryConfig.GetDockerConfigJson()
				Expect(err).To(BeNil())

				unmarshalled := map[string]interface{}{}
				json.Unmarshal(j, &unmarshalled)

				Expect(unmarshalled).To(Equal(map[string]interface{}{
					"auths": map[string]interface{}{
						"https://127.0.0.1": map[string]interface{}{
							"username": "k8s",
							"password": "xyz789",
							"email":    "k8s@example.com",
							//"auth": "?????? maybe....",
						},
					},
				}))
			})
		})

		It("err on missing env values", func() {
			os.Clearenv()

			_, err := Parse()
			Expect(err).NotTo(BeNil())
		})
	})
})
