package httphelpers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHttphelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Httphelpers Suite")
}
