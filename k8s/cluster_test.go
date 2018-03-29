package k8s_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cf-platform-eng/kibosh/config"
	. "github.com/cf-platform-eng/kibosh/k8s"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var creds *config.ClusterCredentials

	BeforeEach(func() {
		creds = &config.ClusterCredentials{
			CAData: "c29tZSByYW5kb20gc3R1ZmY=",
			Server: "127.0.0.1/api",
			Token:  "my-token",
		}
	})

	It("list pods", func() {
		var url string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			url = string(r.URL.Path)
		})
		testserver := httptest.NewServer(handler)
		creds.Server = testserver.URL

		cluster, err := NewCluster(creds)

		Expect(err).To(BeNil())

		cluster.ListPods()

		Expect(url).To(Equal("/api/v1/pods"))
	})
})
