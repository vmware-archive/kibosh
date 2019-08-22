// kibosh
//
// Copyright (c) 2017-Present Pivotal Software, Inc. All Rights Reserved.
//
// This program and the accompanying materials are made available under the terms of the under the Apache License,
// Version 2.0 (the "License”); you may not use this file except in compliance with the License. You may
// obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the
// License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing permissions and
// limitations under the License.

package broker

import (
	"context"
	"crypto/md5"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/credstore"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/moreio"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/ghodss/yaml"
	"github.com/google/go-jsonnet"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sAPI "k8s.io/client-go/tools/clientcmd/api"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
	"os"
	"path"
	"strings"
)

const registrySecretName = "registry-secret"
const CredhubClientIdentifier = "kibosh"

type PksServiceBroker struct {
	config    *config.Config
	repo      repository.Repository
	credstore credstore.CredStore
	operators []*my_helm.MyChart

	clusterFactory                 k8s.ClusterFactory
	helmClientFactory              my_helm.HelmClientFactory
	serviceAccountInstallerFactory k8s.ServiceAccountInstallerFactory
	helmInstallerFactory           my_helm.InstallerFactory

	logger *logrus.Logger
}

func NewPksServiceBroker(
	config *config.Config, clusterFactory k8s.ClusterFactory, helmClientFactory my_helm.HelmClientFactory,
	serviceAccountInstallerFactory k8s.ServiceAccountInstallerFactory, helmInstallerFactory my_helm.InstallerFactory,
	repo repository.Repository, cs credstore.CredStore, operators []*my_helm.MyChart, logger *logrus.Logger,
) *PksServiceBroker {
	broker := &PksServiceBroker{
		config:    config,
		repo:      repo,
		credstore: cs,
		operators: operators,

		clusterFactory:                 clusterFactory,
		helmClientFactory:              helmClientFactory,
		serviceAccountInstallerFactory: serviceAccountInstallerFactory,
		helmInstallerFactory:           helmInstallerFactory,

		logger: logger,
	}

	return broker
}

func (broker *PksServiceBroker) FlushRepoChartCache() error {
	broker.logger.Info("Requested Repo Chart Cache Flush")
	return broker.repo.ClearCache()
}

func (broker *PksServiceBroker) GetChartsMap() (map[string]*my_helm.MyChart, error) {
	chartsMap := map[string]*my_helm.MyChart{}
	charts, err := broker.repo.GetCharts()
	if err != nil {
		return nil, err
	}
	for _, chart := range charts {
		chartsMap[broker.getServiceID(chart)] = chart
	}
	return chartsMap, nil
}

func (broker *PksServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	serviceCatalog := []brokerapi.Service{}

	charts, err := broker.GetChartsMap()
	if err != nil {
		return nil, err
	}
	for _, chart := range charts {
		plans := []brokerapi.ServicePlan{}
		for _, plan := range chart.Plans {
			plans = append(plans, brokerapi.ServicePlan{
				ID:          broker.getServiceID(chart) + "-" + plan.Name,
				Name:        plan.Name,
				Description: plan.Description,
				Metadata: &brokerapi.ServicePlanMetadata{
					DisplayName: plan.Name,
					Bullets: func() []string {
						if plan.Bullets == nil {
							return []string{
								plan.Description,
							}
						}
						return plan.Bullets
					}(),
				},
				Bindable: brokerapi.BindableValue(*plan.Bindable),
				Free:     brokerapi.FreeValue(*plan.Free),
			})
		}

		serviceCatalog = append(serviceCatalog, brokerapi.Service{
			ID:          broker.getServiceID(chart),
			Name:        broker.getServiceName(chart),
			Description: chart.Metadata.Description,
			Bindable:    true,
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName:      broker.getServiceName(chart),
				ImageUrl:         chart.Metadata.Icon,
				DocumentationUrl: chart.Metadata.Home,
			},

			Plans: plans,
		})
	}

	return serviceCatalog, nil
}

func clusterMapKey(instanceID string) string {
	return instanceID + "-instance-to-cluster"
}

type clusterConfigState struct {
	ClusterCredentials config.ClusterCredentials `json:"clusterCredentials"`
}

func (broker *PksServiceBroker) Provision(ctx context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
	}

	planName := strings.TrimPrefix(details.PlanID, details.ServiceID+"-")
	charts, err := broker.GetChartsMap()
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}
	chart := charts[details.ServiceID]
	if chart == nil {
		return brokerapi.ProvisionedServiceSpec{}, errors.New(fmt.Sprintf("Chart not found for [%s]", details.ServiceID))
	}

	var installValues []byte
	if details.GetRawParameters() != nil {
		installValues, err = yaml.JSONToYAML(details.GetRawParameters())
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	var cluster k8s.Cluster
	// todo: load cluster, plans, etc

	if strings.HasSuffix(chart.ChartPath, "tgz") {
		chartReader, err := os.Open(chart.ChartPath)
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
		moreio.Untar(chartReader, path.Dir(chart.ChartPath))
		err = chartReader.Close()
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}
	if chart.ChartPath == "" {
		return brokerapi.ProvisionedServiceSpec{}, errors.New("chart chartPath should not be empty")
	}
	plansDir := path.Join(path.Dir(chart.ChartPath), chart.Chart.Metadata.Name, "plans")

	planDetails, err := chart.LoadPlans(plansDir, chart.Plans)
	if err != nil {
		panic("todo: fix me")
	}

	planHasCluster := chart.Plans[planName].HasCluster()
	if planHasCluster {
		if broker.credstore != nil {
			creds, err := broker.credstore.Get(fmt.Sprintf("/c/%s/%s/%s/cluster-creds", CredhubClientIdentifier, chart.Metadata.Name, planName))
			if err != nil {
				return brokerapi.ProvisionedServiceSpec{}, errors.Wrap(err, "Unable to get creds from credstore")
			}
			clusterConfig := &k8sAPI.Config{}
			err = yaml.Unmarshal([]byte(creds), clusterConfig)
			if err != nil {
				return brokerapi.ProvisionedServiceSpec{}, errors.Wrap(err, "Unable to get unmarshall creds retreived from credstore")
			}
			cluster, err = broker.clusterFactory.GetClusterFromK8sConfig(clusterConfig)
		} else {
			cluster, err = broker.clusterFactory.GetClusterFromK8sConfig(planDetails[planName].ClusterConfig)
		}
	} else {
		cluster, err = broker.clusterFactory.DefaultCluster()
	}
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, errors.Wrap(err, "Unable to get cluster")
	}

	myHelmClient := broker.helmClientFactory.HelmClient(cluster)
	if planHasCluster {
		planClusterServiceAccountInstaller := broker.serviceAccountInstallerFactory.ServiceAccountInstaller(cluster)
		err = PrepareCluster(broker.config, cluster, myHelmClient, planClusterServiceAccountInstaller, broker.helmInstallerFactory, broker.operators, broker.logger)
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	namespaceName := broker.getNamespace(instanceID)
	namespace := api_v1.Namespace{
		Spec: api_v1.NamespaceSpec{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"serviceID":        details.ServiceID,
				"planID":           details.PlanID,
				"organizationGUID": details.OrganizationGUID,
				"spaceGUID":        details.SpaceGUID,
				"instanceID":       instanceID,
			},
		},
	}

	var planBytes []byte
	if broker.credstore != nil {
		creds, err := broker.credstore.Get(fmt.Sprintf("/c/%s/%s/%s/values", CredhubClientIdentifier, chart.Metadata.Name, planName))
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
		planBytes = []byte(creds)
	} else {
		planBytes = planDetails[planName].Values
	}

	_, err = myHelmClient.InstallChart(broker.config.RegistryConfig, namespace, broker.getReleaseName(instanceID), chart, planBytes, installValues)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: "provision",
	}, nil
}

func (broker *PksServiceBroker) GetInstance(context context.Context, instanceID string) (brokerapi.GetInstanceDetailsSpec, error) {
	return brokerapi.GetInstanceDetailsSpec{}, errors.New("this optional operation isn't supported yet")
}

func (broker *PksServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	planID := details.PlanID
	serviceID := details.ServiceID
	cluster, err := broker.getCluster(planID, serviceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	go func() {
		_, err = helmClient.DeleteRelease(broker.getReleaseName(instanceID))
		if err != nil {
			broker.logger.Error(
				"Delete Release failed for planID=", planID, " serviceID=", serviceID, " instanceID=", instanceID, " ", err,
			)
		}

		err = cluster.DeleteNamespace(broker.getNamespace(instanceID), &meta_v1.DeleteOptions{})
		if err != nil {
			broker.logger.Error(
				"Delete Namespace failed for planID=", planID, " serviceID=", serviceID, " instanceID=", instanceID, " ", err,
			)
		}
	}()

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: "deprovision",
	}, nil
}

func (broker *PksServiceBroker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	planID := details.PlanID
	serviceID := details.ServiceID
	cluster, err := broker.getCluster(planID, serviceID)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	chartsMap, err := broker.GetChartsMap()
	if err != nil {
		return brokerapi.Binding{}, err
	}

	chart, ok := chartsMap[serviceID]
	if !ok {
		return brokerapi.Binding{}, errors.New(fmt.Sprintf("service %s not found ", serviceID))
	}

	template := chart.BindTemplate
	credentials, err := broker.getCredentials(cluster, instanceID, template)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	if broker.credstore != nil {
		credentialName := broker.getCredentialName(broker.getServiceName(chart), bindingID)

		_, err := broker.credstore.Put(credentialName, credentials)
		if err != nil {
			return brokerapi.Binding{}, err
		}
		credentials = map[string]interface{}{
			"credhub-ref": credentialName,
		}

		_, err = broker.credstore.AddPermission(credentialName, "mtls-app:"+details.AppGUID, []string{"read"})
		if err != nil {
			return brokerapi.Binding{}, err
		}
	}

	return brokerapi.Binding{
		Credentials: credentials,
	}, nil
}

func (broker *PksServiceBroker) getCluster(planID, serviceID string) (k8s.Cluster, error) {
	planName := strings.TrimPrefix(planID, serviceID+"-")
	charts, err := broker.GetChartsMap()
	if err != nil {
		return nil, err
	}
	chart := charts[serviceID]

	if chart != nil {
		plan, planFound := chart.Plans[planName]
		if planFound && plan.ClusterConfig != nil {
			cluster, err := broker.clusterFactory.GetClusterFromK8sConfig(chart.Plans[planName].ClusterConfig)
			if err != nil {
				return nil, err
			}
			if cluster != nil {
				return cluster, err
			}
		}
	}
	return broker.clusterFactory.DefaultCluster()
}

func (broker *PksServiceBroker) getCredentials(cluster k8s.Cluster, instanceID string, bindTemplate string) (map[string]interface{}, error) {
	servicesAndSecrets, err := cluster.GetSecretsAndServices(broker.getNamespace(instanceID))
	if err != nil {
		return nil, err
	}

	if bindTemplate != "" {
		renderedTemplate, err := my_helm.RenderJsonnetTemplate(bindTemplate, servicesAndSecrets)
		if err != nil {
			return nil, err
		}
		var bindCredentials map[string]interface{}
		err = json.Unmarshal([]byte(renderedTemplate), &bindCredentials)
		if err != nil {
			return nil, err
		}
		return bindCredentials, nil
	} else {
		return servicesAndSecrets, nil
	}
}

func (broker *PksServiceBroker) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("this broker does not support async binding")
}

func (broker *PksServiceBroker) GetBinding(ctx context.Context, instanceID, bindingID string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, errors.New("this optional operation isn't supported yet")
}

func (broker *PksServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {
	chartsMap, err := broker.GetChartsMap()
	if err != nil {
		return brokerapi.UnbindSpec{}, err
	}

	chart, ok := chartsMap[details.ServiceID]
	if !ok {
		return brokerapi.UnbindSpec{}, errors.New(fmt.Sprintf("service %s not found ", details.ServiceID))
	}

	if broker.credstore != nil {
		credentialName := broker.getCredentialName(broker.getServiceName(chart), bindingID)

		err = broker.credstore.DeletePermission(credentialName)
		if err != nil {
			broker.logger.Error(fmt.Sprintf("fail to delete permissions on the key %s", credentialName), err)
		}

		err := broker.credstore.Delete(credentialName)
		if err != nil {
			return brokerapi.UnbindSpec{}, err
		}
	}

	return brokerapi.UnbindSpec{
		IsAsync: false,
	}, nil
}

func (broker *PksServiceBroker) Update(ctx context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	var updateValues []byte
	var err error

	if details.GetRawParameters() == nil {
		return brokerapi.UpdateServiceSpec{
			IsAsync:       true,
			OperationData: "update",
		}, nil
	}

	if details.GetRawParameters() != nil {
		updateValues, err = yaml.JSONToYAML(details.GetRawParameters())
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, err
		}
	}

	charts, err := broker.GetChartsMap()
	if err != nil {
		return brokerapi.UpdateServiceSpec{}, err
	}
	chart := charts[details.ServiceID]
	if chart == nil {
		return brokerapi.UpdateServiceSpec{}, errors.New(fmt.Sprintf("Chart not found for [%s]", details.ServiceID))
	}

	planName := strings.TrimPrefix(details.PlanID, details.ServiceID+"-")

	planID := details.PlanID
	serviceID := details.ServiceID
	cluster, err := broker.getCluster(planID, serviceID)
	if err != nil {
		return brokerapi.UpdateServiceSpec{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	_, err = helmClient.UpdateChart(chart, broker.getReleaseName(instanceID), planName, updateValues)
	if err != nil {
		broker.logger.Debug(fmt.Sprintf("Update failed on update release= %v", err))
		return brokerapi.UpdateServiceSpec{}, err
	}

	return brokerapi.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: "update",
	}, nil
}

func (broker *PksServiceBroker) LastOperation(ctx context.Context, instanceID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	var brokerStatus brokerapi.LastOperationState
	var description string

	planID := details.PlanID
	serviceID := details.ServiceID
	cluster, err := broker.getCluster(planID, serviceID)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	response, err := helmClient.ReleaseStatus(broker.getReleaseName(instanceID))
	if err != nil {
		//This err potentially should result in 410 / ok response, in the case where the release is no-found
		//Will require some changes if we want to support release purging or other flows
		return brokerapi.LastOperation{}, err
	}

	code := response.Info.Status.Code
	operationData := details.OperationData
	if operationData == "provision" {
		switch code {
		case hapi_release.Status_DEPLOYED:
			brokerStatus = brokerapi.Succeeded
			description = "service deployment succeeded"
		case hapi_release.Status_PENDING_INSTALL:
			fallthrough
		case hapi_release.Status_PENDING_UPGRADE:
			brokerStatus = brokerapi.InProgress
			description = "deploy in progress"
		default:
			brokerStatus = brokerapi.Failed
			description = fmt.Sprintf("provision failed %v", code)
		}
	} else if operationData == "deprovision" {
		switch code {
		case hapi_release.Status_DELETED:
			brokerStatus = brokerapi.Succeeded
			description = "gone"
		case hapi_release.Status_DEPLOYED:
			fallthrough
		case hapi_release.Status_DELETING:
			brokerStatus = brokerapi.InProgress
			description = "delete in progress"
		default:
			brokerStatus = brokerapi.Failed
			description = fmt.Sprintf("deprovision failed %v", code)
		}
	} else if operationData == "update" {
		switch code {
		case hapi_release.Status_DEPLOYED:
			brokerStatus = brokerapi.Succeeded
			description = "updated"
		default:
			brokerStatus = brokerapi.Failed
			description = fmt.Sprintf("update failed %v", code)
		}
	}

	if brokerStatus != brokerapi.Succeeded {
		return brokerapi.LastOperation{
			State:       brokerStatus,
			Description: description,
		}, nil
	}

	var message *string
	if operationData != "deprovision" {
		message, code, err = helmClient.ResourceReadiness(broker.getNamespace(instanceID), cluster)
		if err != nil || code == hapi_release.Status_UNKNOWN {
			return brokerapi.LastOperation{}, err
		}
		if message == nil {
			message = &description
		}
		if code == hapi_release.Status_PENDING_INSTALL {
			return brokerapi.LastOperation{
				State:       brokerapi.InProgress,
				Description: *message,
			}, nil
		}
	} else {
		message = &description
	}

	return brokerapi.LastOperation{
		State:       brokerapi.Succeeded,
		Description: *message,
	}, nil
}

func (broker *PksServiceBroker) getNamespace(instanceID string) string {
	return "kibosh-" + instanceID
}

func (broker *PksServiceBroker) getReleaseName(instanceID string) string {
	hashed := md5.Sum([]byte(instanceID))
	encoded := base32.StdEncoding.EncodeToString(hashed[:])
	return fmt.Sprintf("k-%s", strings.ToLower(string(encoded[0:8])))
}

func (broker *PksServiceBroker) getServiceName(chart *my_helm.MyChart) string {
	return chart.Metadata.Name
}

func (broker *PksServiceBroker) getServiceID(chart *my_helm.MyChart) string {
	return uuid.NewSHA1(uuid.NameSpace_OID, []byte(broker.getServiceName(chart))).String()
}

func (broker *PksServiceBroker) getCredentialName(serviceName, bindingID string) string {
	return fmt.Sprintf("/c/%s/%s/%s/secrets-and-services", CredhubClientIdentifier, serviceName, bindingID)
}

func (broker *PksServiceBroker) getRenderedTemplate(bindTemplate string, servicesAndSecrets map[string]interface{}) (string, error) {
	ssTemplateBytes, err := json.Marshal(servicesAndSecrets)
	if err != nil {
		return "", err
	}

	ssTemplate := string(ssTemplateBytes)
	i := strings.LastIndex(ssTemplate, "}")
	fullTemplate := ssTemplate[0:i] + `,"template": ` + bindTemplate + "}"
	vm := jsonnet.MakeVM()
	renderedTemplate, err := vm.EvaluateSnippetMulti("", fullTemplate)
	if err != nil {
		return "", err
	}

	return renderedTemplate["template"], nil
}
