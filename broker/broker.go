package broker

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"

	"code.cloudfoundry.org/lager"
	//my_helm "github.com/cf-platform-eng/kibosh/helm"
	//"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	//hapi_release "k8s.io/helm/pkg/proto/hapi/release"
)

// PksServiceBroker contains values passed in from configuration necessary for broker's work.
type PksServiceBroker struct {
	Logger       lager.Logger
	HelmChartDir string
	ServiceID    string
	//cluster      k8s.Cluster
	//myHelmClient my_helm.MyHelmClient
}

// todo: now that we're including helm, it probably make sense to defer to the helm library's parsing?
// HelmChart contains heml chart data useful for Broker Catalog
type HelmChart struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

//func NewPksServiceBroker(helmChartDir string, serviceID string, cluster k8s.Cluster, myHelmClient my_helm.MyHelmClient) *PksServiceBroker {
func NewPksServiceBroker(helmChartDir string, serviceID string, cluster interface{}, myHelmClient interface{}) *PksServiceBroker {
	return &PksServiceBroker{
		HelmChartDir: helmChartDir,
		ServiceID:    serviceID,
		//cluster:      cluster,
		//myHelmClient: myHelmClient,
	}
}

// GetConf parses the chart yaml file.
func GetConf(yamlReader io.Reader) (*HelmChart, error) {

	yamlFile, err := ioutil.ReadAll(yamlReader)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read Chart.yaml")
	}

	c := &HelmChart{}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to unmarshal Chart.yaml")
	}

	return c, nil
}

// Services reads Chart.yaml and uses the name and description in the service catalog.
func (pksServiceBroker *PksServiceBroker) Services(ctx context.Context) []brokerapi.Service {

	// Get the helm chart Chart.yaml data
	file, err := os.Open(path.Join(pksServiceBroker.HelmChartDir, "Chart.yaml"))
	if err != nil {
		pksServiceBroker.Logger.Fatal("Unable to read Chart.yaml", err)
	}
	defer file.Close()
	helmChart, err := GetConf(file)
	if err != nil {
		pksServiceBroker.Logger.Fatal("Unable to parse Chart.yaml", err)
	}

	// Create a default plan.
	plan := []brokerapi.ServicePlan{{
		ID:          pksServiceBroker.ServiceID + "-default",
		Name:        "default",
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
	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	/*
	namespaceName := instanceID
	namespace := api_v1.Namespace{
		Spec: api_v1.NamespaceSpec{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"serviceID":        details.ServiceID,
				"planID":           details.PlanID,
				"organizationGUID": details.OrganizationGUID,
				"spaceGUID":        details.SpaceGUID,
			},
		},
	}

	_, err := pksServiceBroker.cluster.CreateNamespace(&namespace)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	_, err = pksServiceBroker.myHelmClient.InstallReleaseFromDir(
		pksServiceBroker.HelmChartDir, namespaceName, helm.ReleaseName(instanceID),
	)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}*/

	return brokerapi.ProvisionedServiceSpec{
		IsAsync: true,
	}, nil
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
	/*
	var brokerStatus brokerapi.LastOperationState
	var description string
	response, err := pksServiceBroker.myHelmClient.ReleaseStatus(instanceID)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	code := response.Info.Status.Code
	switch code {
	case hapi_release.Status_DEPLOYED:
		//todo: treat Status_DELETED as succeeded
		brokerStatus = brokerapi.Succeeded
		description = "service deployment succeeded"
	case hapi_release.Status_PENDING_INSTALL:
		fallthrough
	case hapi_release.Status_PENDING_UPGRADE:
		brokerStatus = brokerapi.InProgress
		description = "service deployment in progress"
	default:
		brokerStatus = brokerapi.Failed
		description = fmt.Sprintf("service deployment failed %v", code)
	}

	return brokerapi.LastOperation{
		State:       brokerStatus,
		Description: description,
	}, nil
	*/
	return brokerapi.LastOperation{}, nil
}
