package broker_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Broker", func() {
	Context("Broker", func() {
		BeforeEach(func() {
			os.Clearenv()
		})

		It("Provides a catalog", func() {
			// TODO: Create some tests
			Expect("").NotTo(BeNil())
		})
	})
})
