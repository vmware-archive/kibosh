package broker

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/pivotal-cf/brokerapi"
	yaml "gopkg.in/yaml.v2"
)

// PksServiceBroker contains values passed in from configuration necessary for broker's work.
type PksServiceBroker struct {
	HelmChartDir string
	ServiceID    string
}

// HelmChart contains heml chart data useful for Broker Catalog
type HelmChart struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// GetConf parses the chart yaml file.
func GetConf(yamlReader io.Reader) *HelmChart {

	yamlFile, err := ioutil.ReadAll(yamlReader)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	c := &HelmChart{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

// Services reads Chart.yaml and uses the name and description in the service catalog.
func (pksServiceBroker *PksServiceBroker) Services(ctx context.Context) []brokerapi.Service {

	// Get the helm chart Chart.yaml data
	file, err := os.Open(path.Join(pksServiceBroker.HelmChartDir, "Chart.yaml"))
	if err != nil {
		log.Fatal("Unable to read Chart.yaml", err)
	}
	defer file.Close()
	helmChart := GetConf(file)

	// Create a default plan.
	plan := []brokerapi.ServicePlan{{
		ID:          pksServiceBroker.ServiceID + "-Default",
		Name:        "Default",
		Description: helmChart.Description,
	}}

	serviceCatalog := []brokerapi.Service{{
		ID:          pksServiceBroker.ServiceID,
		Name:        helmChart.Name,
		Description: helmChart.Description,

		Plans: plan,
	}}

	return serviceCatalog
}

// Provision uses the cloud controller service id to create a namespace in k8s,
// does the Helm install into that namespace, and
// converts the cf create-instance json block into Helm key-value pairs.
func (pksServiceBroker *PksServiceBroker) Provision(ctx context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {

	// TODO

	return brokerapi.ProvisionedServiceSpec{}, nil
}

// Deprovision deletes the namespace (and everything in it) created by provision.
func (pksServiceBroker *PksServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {

	// TODO

	return brokerapi.DeprovisionServiceSpec{}, nil
}

// Bind fetches the secrets and services from the k8s namespace.
// It may use "kubectl expose" and/or "kubectl show services".
// It creates environment variables in the scope of the cf app by populating the brokerapi.Binding object.
func (pksServiceBroker *PksServiceBroker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {

	// TODO

	return brokerapi.Binding{}, nil
}

// Unbind reverses bind
func (pksServiceBroker *PksServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {

	// noop

	return nil
}

// Update is perhaps not needed for MVP.
// Its purpose may be for changing plans, so if we only have a single default plan
// it is out of scope.
func (pksServiceBroker *PksServiceBroker) Update(ctx context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, nil
}

// LastOperation is for async
func (pksServiceBroker *PksServiceBroker) LastOperation(ctx context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, nil
}
