package logger_test

import (
	"bufio"
	"bytes"
	"code.cloudfoundry.org/lager"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("lager adapter", func() {
	var outBytes bytes.Buffer
	var out *bufio.Writer

	BeforeEach(func() {
		outBytes = bytes.Buffer{}
		out = bufio.NewWriter(&outBytes)
	})

	It("sends debug logs", func() {
		logrusLogger := logrus.New()
		logrusLogger.Out = out
		logrusLogger.Level = logrus.DebugLevel
		sink := logger.NewLogrusSink(logrusLogger)

		lagerLogger := lager.NewLogger("test")
		lagerLogger.RegisterSink(sink)

		lagerLogger.Debug("my_debug_message")

		out.Flush()

		Expect(outBytes.String()).To(ContainSubstring("level=debug"))
		Expect(outBytes.String()).To(ContainSubstring("my_debug_message"))
	})

	It("sends error logs", func() {
		logrusLogger := logrus.New()
		logrusLogger.Out = out
		logrusLogger.Level = logrus.DebugLevel
		sink := logger.NewLogrusSink(logrusLogger)

		lagerLogger := lager.NewLogger("test")
		lagerLogger.RegisterSink(sink)

		lagerLogger.Error("my_error_message", errors.New("my error"))

		out.Flush()

		Expect(outBytes.String()).To(ContainSubstring("level=error"))
		Expect(outBytes.String()).To(ContainSubstring("my_error_message"))
	})
})
