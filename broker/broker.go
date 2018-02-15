package broker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"code.cloudfoundry.org/lager"
	my_helm "github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
)

// PksServiceBroker contains values passed in from configuration necessary for broker's work.
type PksServiceBroker struct {
	Logger       lager.Logger
	HelmChartDir string
	ServiceID    string
	cluster      k8s.Cluster
	myHelmClient my_helm.MyHelmClient
}

// todo: now that we're including helm, it probably make sense to defer to the helm library's parsing?
// HelmChart contains heml chart data useful for Broker Catalog
type HelmChart struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func NewPksServiceBroker(helmChartDir string, serviceID string, cluster k8s.Cluster, myHelmClient my_helm.MyHelmClient, logger lager.Logger) *PksServiceBroker {
	return &PksServiceBroker{
		HelmChartDir: helmChartDir,
		ServiceID:    serviceID,
		cluster:      cluster,
		myHelmClient: myHelmClient,
		Logger:       logger,
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
func (broker *PksServiceBroker) Services(ctx context.Context) []brokerapi.Service {
	// Get the helm chart Chart.yaml data
	file, err := os.Open(path.Join(broker.HelmChartDir, "Chart.yaml"))
	if err != nil {
		broker.Logger.Fatal("Unable to read Chart.yaml", err)
	}
	defer file.Close()
	helmChart, err := GetConf(file)
	if err != nil {
		broker.Logger.Fatal("Unable to parse Chart.yaml", err)
	}

	// Create a default plan.
	plan := []brokerapi.ServicePlan{{
		ID:          broker.ServiceID + "-default",
		Name:        "default",
		Description: helmChart.Description,
	}}

	serviceCatalog := []brokerapi.Service{{
		ID:          broker.ServiceID,
		Name:        helmChart.Name,
		Description: helmChart.Description,
		Bindable:    true,

		Plans: plan,
	}}

	return serviceCatalog
}

// Provision uses the cloud controller service id to create a namespace in k8s,
// does the Helm install into that namespace, and
// converts the cf create-instance json block into Helm key-value pairs.
func (broker *PksServiceBroker) Provision(ctx context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	namespace := api_v1.Namespace{
		Spec: api_v1.NamespaceSpec{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: broker.getNamespace(instanceID),
			Labels: map[string]string{
				"serviceID":        details.ServiceID,
				"planID":           details.PlanID,
				"organizationGUID": details.OrganizationGUID,
				"spaceGUID":        details.SpaceGUID,
			},
		},
	}
	_, err := broker.cluster.CreateNamespace(&namespace)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	_, err = broker.myHelmClient.InstallReleaseFromDir(
		broker.HelmChartDir, broker.getNamespace(instanceID), helm.ReleaseName(broker.getNamespace(instanceID)),
	)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync: true,
	}, nil
}

// Deprovision deletes the namespace (and everything in it) created by provision.
func (broker *PksServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	_, err := broker.myHelmClient.DeleteRelease(broker.getNamespace(instanceID), helm.DeletePurge(true))
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	err = broker.cluster.DeleteNamespace(broker.getNamespace(instanceID), &meta_v1.DeleteOptions{})
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (broker *PksServiceBroker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	secrets, err := broker.cluster.ListSecrets(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return brokerapi.Binding{}, err
	}

	secretsMap := []map[string]interface{}{}
	for _, secret := range secrets.Items {
		if secret.Type == api_v1.SecretTypeOpaque {
			credentialSecrets := map[string]string{}
			for key, val := range secret.Data {
				credentialSecrets[key] = string(val)
			}
			credential := map[string]interface{}{
				"name": secret.Name,
				"data": credentialSecrets,
			}
			secretsMap = append(secretsMap, credential)
		}
	}

	services, err := broker.cluster.ListServices(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return brokerapi.Binding{}, err
	}

	servicesMap := []map[string]interface{}{}
	for _, service := range services.Items {
		credentialService := map[string]interface{}{
			"name": service.ObjectMeta.Name,
			"spec": service.Spec,
			"status": service.Status,
		}
		servicesMap = append(servicesMap, credentialService)
	}

	return brokerapi.Binding{
		Credentials: map[string]interface{}{
			"secrets":  secretsMap,
			"services": servicesMap,
		},
	}, nil
}



// Unbind reverses bind
func (broker *PksServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {

	// noop

	return nil
}

// Update is perhaps not needed for MVP.
// Its purpose may be for changing plans, so if we only have a single default plan
// it is out of scope.
func (broker *PksServiceBroker) Update(ctx context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, nil
}

// LastOperation is for async
func (broker *PksServiceBroker) LastOperation(ctx context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	var brokerStatus brokerapi.LastOperationState
	var description string
	response, err := broker.myHelmClient.ReleaseStatus(broker.getNamespace(instanceID))
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

	services, err := broker.cluster.ListServices(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	serviceReady := true
	for _, service := range services.Items {
		if service.Spec.Type == "LoadBalancer" {
			if len(service.Status.LoadBalancer.Ingress) < 1 {
				serviceReady = false
			}
		}
	}
	if brokerStatus == brokerapi.Succeeded && !serviceReady {
		brokerStatus = brokerapi.InProgress
		description = "service deployment in progress"
	}

	return brokerapi.LastOperation{
		State:       brokerStatus,
		Description: description,
	}, nil
}

func (broker *PksServiceBroker) getNamespace(instanceID string) string {
	return "kibosh-" + instanceID
}
