package bazaar_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBazaar(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bazaar Suite")
}
