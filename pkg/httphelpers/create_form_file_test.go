package httphelpers_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"io/ioutil"
)

var _ = Describe("Save charts", func() {
	var testRequest *http.Request
	var testServer *httptest.Server

	BeforeEach(func() {
	})

	AfterEach(func() {
		testServer.Close()
	})

	It("correctly adds file to request", func() {
		file, err := ioutil.TempFile("", "")
		Expect(err).To(BeNil())
		_, err = file.Write([]byte("some random content stuff"))
		Expect(err).To(BeNil())

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.ParseMultipartForm(4096)
			testRequest = r
		})
		testServer = httptest.NewServer(handler)

		req, err := httphelpers.CreateFormRequest(testServer.URL, "my_file", file.Name())
		Expect(err).To(BeNil())

		res, err := http.DefaultClient.Do(req)
		Expect(err).To(BeNil())
		Expect(res.StatusCode).To(Equal(200))

		Expect(testRequest.Method).To(Equal("POST"))

		formFile, _, err := testRequest.FormFile("my_file")
		Expect(err).To(BeNil())

		fileContents, err := ioutil.ReadAll(formFile)
		Expect(err).To(BeNil())

		Expect(fileContents).To(Equal([]byte("some random content stuff")))
	})

	It("returns error on non-existant file", func() {
		_, err := httphelpers.CreateFormRequest(testServer.URL, "my_file", "")
		Expect(err).NotTo(BeNil())
	})
})
