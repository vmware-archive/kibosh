package cli_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"github.com/cf-platform-eng/kibosh/pkg/bazaar/cli"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/spf13/cobra"
)

var _ = Describe("Delete charts", func() {
	var b bytes.Buffer
	var out *bufio.Writer

	var bazaarAPIRequest *http.Request
	var bazaarAPITestServer *httptest.Server
	var c *cobra.Command

	BeforeEach(func() {
		b = bytes.Buffer{}
		out = bufio.NewWriter(&b)
		c = cli.NewChartsDeleteCmd(out)

		c.Flags().Set("user", "bob")
		c.Flags().Set("password", "monkey123")

	})
	AfterEach(func() {
		bazaarAPITestServer.Close()
	})

	It("calls delete chart", func() {
		msgFromServer := bazaar.DisplayResponse{
			Message: "Yay",
		}
		responseBody, _ := json.Marshal(msgFromServer)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(responseBody)
			bazaarAPIRequest = r
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c.Flags().Set("target", bazaarAPITestServer.URL)

		err := c.RunE(c, []string{
			"casandra",
		})
		out.Flush()

		Expect(err).To(BeNil())

		Expect(string(b.Bytes())).To(ContainSubstring("Yay"))
		Expect(bazaarAPIRequest.URL.Path).To(Equal("/charts/casandra"))

	})

	It("correctly auths request", func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{}"))
			bazaarAPIRequest = r
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c.Flags().Set("target", bazaarAPITestServer.URL)

		err := c.RunE(c, []string{
			"cassandra",
		})
		out.Flush()

		Expect(err).To(BeNil())
		Expect(bazaarAPIRequest.Header.Get("Authorization")).To(
			Equal(httphelpers.BasicAuthHeaderVal("bob", "monkey123")),
		)
	})

	It("error when chart name not supplied", func() {
		c.Flags().Set("target", bazaarAPITestServer.URL)

		err := c.RunE(c, []string{})
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("chart"))
	})

	It("auth failure delete chart", func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c.Flags().Set("target", bazaarAPITestServer.URL)

		err := c.RunE(c, []string{
			"cassandra",
		})
		out.Flush()

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("401"))
	})
})
