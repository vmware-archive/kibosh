package broker_test

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/cf-platform-eng/pks-generic-broker/broker"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/brokerapi"
)

var _ = Describe("Broker", func() {

	chartYamlContent := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	Context("Broker", func() {
		BeforeEach(func() {
			os.Clearenv()
		})

		It("Provides a catalog", func() {

			// Create a temporary directory (defer removing it until after method)
			dir, err := ioutil.TempDir("", "chart-")
			if err != nil {
				log.Fatal(err)
			}
			defer os.RemoveAll(dir)

			// Create a temporary file in that directory with chart yaml.
			tmpfn := filepath.Join(dir, "Chart.yaml")
			if err := ioutil.WriteFile(tmpfn, chartYamlContent, 0666); err != nil {
				log.Fatal(err)
			}

			// Create the service catalog using that yaml.
			serviceBroker := &broker.PksServiceBroker{
				HelmChartDir: dir,
				ServiceID:    "service-id",
			}
			serviceCatalog := serviceBroker.Services(nil)

			// Evalute correctness of service catalog against expected.
			expectedPlan := []brokerapi.ServicePlan{{
				ID:          "service-id-Default",
				Name:        "Default",
				Description: "spacebears service and spacebears broker helm chart",
			}}
			expectedCatalog := []brokerapi.Service{{
				ID:          "service-id",
				Name:        "spacebears",
				Description: "spacebears service and spacebears broker helm chart",
				Plans:       expectedPlan,
			}}
			Expect(expectedCatalog).Should(Equal(serviceCatalog))

		})

	})
})
