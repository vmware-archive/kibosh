package logger_test

import (
	"bufio"
	"bytes"

	"github.com/cf-platform-eng/kibosh/pkg/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("out err splitter", func() {
	var stdOutBuffer bytes.Buffer
	var stdOut *bufio.Writer

	var errOutBuffer bytes.Buffer
	var errOut *bufio.Writer

	BeforeEach(func() {
		stdOutBuffer = bytes.Buffer{}
		stdOut = bufio.NewWriter(&stdOutBuffer)

		errOutBuffer = bytes.Buffer{}
		errOut = bufio.NewWriter(&errOutBuffer)
	})

	It("splits logs", func() {
		l := logger.NewSplitLogger(stdOut, errOut)
		l.Error("some_error")
		l.Debug("some_debug")

		stdOut.Flush()
		Expect(stdOutBuffer.String()).To(ContainSubstring("level=debug"))
		Expect(stdOutBuffer.String()).To(ContainSubstring("msg=some_debug"))

		errOut.Flush()
		Expect(errOutBuffer.String()).To(ContainSubstring("level=error"))
		Expect(errOutBuffer.String()).To(ContainSubstring("msg=some_error"))
	})
})
