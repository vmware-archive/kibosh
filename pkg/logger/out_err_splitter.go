package logger

import (
	"bytes"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// See this, once merged, this can go away https://github.com/sirupsen/logrus/pull/671
type OutputSplitter struct {
	stdOut io.Writer
	stdErr io.Writer
}

func NewOutputSplitter(stdOut io.Writer, stdErr io.Writer) *OutputSplitter {
	return &OutputSplitter{
		stdOut: stdOut,
		stdErr: stdErr,
	}
}

func (splitter *OutputSplitter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, []byte("level=error")) || bytes.Contains(p, []byte("level=fatal")) {
		return splitter.stdErr.Write(p)
	}
	return splitter.stdOut.Write(p)
}

func NewSplitLogger(stdOut io.Writer, errOut io.Writer) *logrus.Logger {
	formatter := new(logrus.TextFormatter)
	formatter.ForceColors = checkIfTerminal(stdOut) && checkIfTerminal(errOut)

	out := NewOutputSplitter(stdOut, errOut)
	return &logrus.Logger{
		Out:       out,
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
}

//copied from Logrus, needed because their terminal detection doesn't work with custom Writer
func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}
