package credstore_test

import (
	"github.com/cf-platform-eng/kibosh/pkg/credstore"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Credhub Store", func() {
	var logger *logrus.Logger

	var uaaTestServer *httptest.Server
	var uaaRequest *http.Request

	var chTestServer *httptest.Server
	var chRequest *http.Request

	BeforeEach(func() {
		logger = logrus.New()

		uaaTestServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
			uaaRequest = r
		}))
		chTestServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"data": [{"foo": "bar"}]}`))
			chRequest = r
		}))
	})

	It("sanity check (class under test is mostly just a wrapper", func() {
		chStore, err := credstore.NewCredhubStore(
			chTestServer.URL, uaaTestServer.URL,
			"my-client", "my-scret",
			true, logger,
		)
		Expect(err).To(BeNil())
		cred, err := chStore.Get("/foo/bar/baz")

		Expect(err).To(BeNil())
		Expect(cred).NotTo(BeNil())

		Expect(uaaRequest).NotTo(BeNil())

		Expect(chRequest).NotTo(BeNil())
		chRequest.ParseForm()
		Expect(chRequest.Form["name"]).To(Equal([]string{"/foo/bar/baz"}))
	})
})
