package k8s_test

import (
	"github.com/cf-platform-eng/kibosh/config"
	. "github.com/cf-platform-eng/kibosh/k8s"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Config", func() {
	var kuboConfig *config.KuboODBVCAP

	BeforeEach(func() {
		kuboConfig = &config.KuboODBVCAP{
			Credentials: config.Credentials{
				KubeConfig: config.KubeConfig{
					Users: []config.User{
						{
							UserCredentials: config.UserCredentials{
								Token: "my-token",
							},
						},
					},
					Clusters: []config.Cluster{
						{
							ClusterInfo: config.ClusterInfo{
								Server: "http://example.com:333",
								CAData: "bXktZmFrZWNlcnQ=",
							},
						},
					},
				},
			},
		}
	})

	It("list pods", func() {
		var url string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			url = string(r.URL.Path)
		})
		testserver := httptest.NewServer(handler)
		kuboConfig.Credentials.KubeConfig.Clusters[0].ClusterInfo.Server = testserver.URL

		cluster, err := NewCluster(kuboConfig)

		Expect(err).To(BeNil())

		cluster.ListPods()

		Expect(url).To(Equal("/api/v1/pods"))
	})
})
