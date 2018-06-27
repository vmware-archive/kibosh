package cli_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cf-platform-eng/kibosh/auth"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/cf-platform-eng/kibosh/bazaar/cli"
)

var _ = Describe("List charts", func() {
	var b bytes.Buffer
	var out *bufio.Writer

	var bazaarAPIRequest *http.Request
	var bazaarAPITestServer *httptest.Server

	BeforeEach(func() {
		b = bytes.Buffer{}
		out = bufio.NewWriter(&b)

	})
	AfterEach(func() {
		bazaarAPITestServer.Close()
	})

	It("calls list charts", func() {
		charts := []bazaar.DisplayChart{
			{
				Name:    "mysql",
				Version: "0.1",
				Plans:   []string{"small", "medium"},
			},
			{
				Name:    "spacebears",
				Version: "0.2",
				Plans:   []string{"tiny", "big"},
			},
		}
		responseBody, _ := json.Marshal(charts)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(responseBody)
			bazaarAPIRequest = r
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c := cli.NewChartsListCmd(out)
		c.Flags().Set("target", bazaarAPITestServer.URL)
		c.Flags().Set("user", "bob")
		c.Flags().Set("password", "monkey123")

		err := c.RunE(c, []string{})
		out.Flush()

		Expect(err).To(BeNil())

		Expect(string(b.Bytes())).To(ContainSubstring("mysql"))
		Expect(string(b.Bytes())).To(ContainSubstring("spacebears"))
	})

	It("correctly auths request", func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("[]"))
			bazaarAPIRequest = r
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c := cli.NewChartsListCmd(out)
		c.Flags().Set("target", bazaarAPITestServer.URL)
		c.Flags().Set("user", "bob")
		c.Flags().Set("password", "monkey123")

		err := c.RunE(c, []string{})
		out.Flush()

		Expect(err).To(BeNil())
		Expect(bazaarAPIRequest.Header.Get("Authorization")).To(
			Equal(auth.BasicAuthorizationHeaderVal("bob", "monkey123")),
		)
	})

	It("calls list charts", func() {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		})
		bazaarAPITestServer = httptest.NewServer(handler)

		c := cli.NewChartsListCmd(out)
		c.Flags().Set("target", bazaarAPITestServer.URL)
		c.Flags().Set("user", "bob")
		c.Flags().Set("password", "monkey123")

		err := c.RunE(c, []string{})
		out.Flush()

		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("401"))

	})
})
