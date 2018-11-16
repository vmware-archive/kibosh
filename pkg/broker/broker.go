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
	"fmt"
	"strings"

	"github.com/cf-platform-eng/kibosh/pkg/state"

	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/ghodss/yaml"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
)

const registrySecretName = "registry-secret"

type PksServiceBroker struct {
	config                         *config.Config
	clusterFactory                 k8s.ClusterFactory
	helmClientFactory              my_helm.HelmClientFactory
	serviceAccountInstallerFactory k8s.ServiceAccountInstallerFactory
	charts                         []*my_helm.MyChart
	mapInstanceToCluster           state.KeyValueStore
	operators                      []*my_helm.MyChart

	logger *logrus.Logger
}

func NewPksServiceBroker(config *config.Config, clusterFactory k8s.ClusterFactory, helmClientFactory my_helm.HelmClientFactory, serviceAccountInstallerFactory k8s.ServiceAccountInstallerFactory, charts []*my_helm.MyChart, operators []*my_helm.MyChart, mapInstanceToCluster state.KeyValueStore, logger *logrus.Logger) *PksServiceBroker {
	broker := &PksServiceBroker{
		logger:                         logger,
		config:                         config,
		clusterFactory:                 clusterFactory,
		helmClientFactory:              helmClientFactory,
		serviceAccountInstallerFactory: serviceAccountInstallerFactory,
		charts:                         charts,
		operators:                      operators,
		mapInstanceToCluster:           mapInstanceToCluster,
	}

	return broker
}

func (broker *PksServiceBroker) SetCharts(charts []*my_helm.MyChart) {
	broker.charts = charts
}

func (broker *PksServiceBroker) GetChartsMap() map[string]*my_helm.MyChart {
	chartsMap := map[string]*my_helm.MyChart{}
	for _, chart := range broker.charts {
		chartsMap[broker.getServiceID(chart)] = chart
	}
	return chartsMap
}

func (broker *PksServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	serviceCatalog := []brokerapi.Service{}

	for _, chart := range broker.GetChartsMap() {
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

	var cluster k8s.Cluster
	var err error
	clusterCreds, configPresent := ExtractClusterConfig(details.GetRawParameters())

	if configPresent {
		cluster, err = broker.clusterFactory.GetCluster(&clusterCreds)
	} else {
		cluster, err = broker.clusterFactory.DefaultCluster()
	}

	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	planName := strings.TrimPrefix(details.PlanID, details.ServiceID+"-")
	var installValues []byte // = nil
	if details.GetRawParameters() != nil {
		installValues, err = yaml.JSONToYAML(details.GetRawParameters())
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	myHelmClient := broker.helmClientFactory.HelmClient(cluster)

	myServiceAccountInstaller := broker.serviceAccountInstallerFactory.ServiceAccountInstaller(cluster)

	if configPresent {
		err = PrepareCluster(broker.config, cluster, myHelmClient, myServiceAccountInstaller, broker.logger, broker.operators)

		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	chart := broker.GetChartsMap()[details.ServiceID]
	if chart == nil {
		return brokerapi.ProvisionedServiceSpec{}, errors.New(fmt.Sprintf("Chart not found for [%s]", details.ServiceID))
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
			},
		},
	}

	_, err = myHelmClient.InstallChart(broker.config.RegistryConfig, namespace, chart, planName, installValues)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	if configPresent {
		broker.mapInstanceToCluster.PutJson(clusterMapKey(instanceID), clusterConfigState{ClusterCredentials: clusterCreds})
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: "provision",
	}, nil
}

func (broker *PksServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	var cluster k8s.Cluster
	var clusterConfigForInstance clusterConfigState

	err := broker.mapInstanceToCluster.GetJson(clusterMapKey(instanceID), &clusterConfigForInstance)

	if err == nil {
		cluster, err = broker.clusterFactory.GetCluster(&clusterConfigForInstance.ClusterCredentials)
	} else if err == state.KeyNotFoundError {
		cluster, err = broker.clusterFactory.DefaultCluster()
		if err != nil {
			return brokerapi.DeprovisionServiceSpec{}, err
		}
	} else {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	_, err = helmClient.DeleteRelease(broker.getNamespace(instanceID))
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	err = cluster.DeleteNamespace(broker.getNamespace(instanceID), &meta_v1.DeleteOptions{})
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: "deprovision",
	}, nil
}

func (broker *PksServiceBroker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, error) {
	var cluster k8s.Cluster
	var clusterConfigForInstance clusterConfigState

	err := broker.mapInstanceToCluster.GetJson(clusterMapKey(instanceID), &clusterConfigForInstance)

	if err == nil {
		cluster, err = broker.clusterFactory.GetCluster(&clusterConfigForInstance.ClusterCredentials)
	} else if err == state.KeyNotFoundError {
		cluster, err = broker.clusterFactory.DefaultCluster()
		if err != nil {
			return brokerapi.Binding{}, err
		}
	} else {
		return brokerapi.Binding{}, err
	}

	secrets, err := cluster.ListSecrets(broker.getNamespace(instanceID), meta_v1.ListOptions{})
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

	services, err := cluster.ListServices(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return brokerapi.Binding{}, err
	}

	servicesMap := []map[string]interface{}{}
	for _, service := range services.Items {
		if service.Spec.Type == "NodePort" {
			nodes, _ := cluster.ListNodes(meta_v1.ListOptions{})
			for _, node := range nodes.Items {
				service.Spec.ExternalIPs = append(service.Spec.ExternalIPs, node.ObjectMeta.Labels["spec.ip"])
			}
		}
		credentialService := map[string]interface{}{
			"name":   service.ObjectMeta.Name,
			"spec":   service.Spec,
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

func (broker *PksServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails) error {
	// noop

	return nil
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

	chart := broker.GetChartsMap()[details.ServiceID]
	if chart == nil {
		return brokerapi.UpdateServiceSpec{}, errors.New(fmt.Sprintf("Chart not found for [%s]", details.ServiceID))
	}

	planName := strings.TrimPrefix(details.PlanID, details.ServiceID+"-")

	var cluster k8s.Cluster
	var clusterConfigForInstance clusterConfigState

	err = broker.mapInstanceToCluster.GetJson(clusterMapKey(instanceID), &clusterConfigForInstance)

	if err == nil {
		cluster, err = broker.clusterFactory.GetCluster(&clusterConfigForInstance.ClusterCredentials)
	} else if err == state.KeyNotFoundError {
		cluster, err = broker.clusterFactory.DefaultCluster()
		if err != nil {
			return brokerapi.UpdateServiceSpec{}, err
		}
	} else {
		return brokerapi.UpdateServiceSpec{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	_, err = helmClient.UpdateChart(chart, broker.getNamespace(instanceID), planName, updateValues)
	if err != nil {
		broker.logger.Debug(fmt.Sprintf("Update failed on update release= %v", err))
		return brokerapi.UpdateServiceSpec{}, err
	}

	return brokerapi.UpdateServiceSpec{
		IsAsync:       true,
		OperationData: "update",
	}, nil
}

// LastOperation is for async
func (broker *PksServiceBroker) LastOperation(ctx context.Context, instanceID, operationData string) (brokerapi.LastOperation, error) {
	var brokerStatus brokerapi.LastOperationState
	var description string

	var cluster k8s.Cluster
	var clusterConfigForInstance clusterConfigState

	err := broker.mapInstanceToCluster.GetJson(clusterMapKey(instanceID), &clusterConfigForInstance)

	if err == nil {
		cluster, err = broker.clusterFactory.GetCluster(&clusterConfigForInstance.ClusterCredentials)
	} else if err == state.KeyNotFoundError {
		cluster, err = broker.clusterFactory.DefaultCluster()
		if err != nil {
			return brokerapi.LastOperation{}, err
		}
	} else {
		return brokerapi.LastOperation{}, err
	}

	helmClient := broker.helmClientFactory.HelmClient(cluster)

	response, err := helmClient.ReleaseStatus(broker.getNamespace(instanceID))
	if err != nil {
		return brokerapi.LastOperation{}, err
	}

	code := response.Info.Status.Code
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

	servicesReady, err := broker.servicesReady(instanceID, cluster)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}
	if !servicesReady {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "service deployment load balancer in progress",
		}, nil
	}

	message, podsReady, err := broker.podsReady(instanceID, cluster)
	if err != nil {
		return brokerapi.LastOperation{}, err
	}
	if !podsReady {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: message,
		}, nil
	}

	return brokerapi.LastOperation{
		State:       brokerStatus,
		Description: description,
	}, nil
}

func (broker *PksServiceBroker) servicesReady(instanceID string, cluster k8s.Cluster) (bool, error) {
	services, err := cluster.ListServices(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return false, err
	}

	servicesReady := true
	for _, service := range services.Items {
		if service.Spec.Type == "LoadBalancer" {
			if len(service.Status.LoadBalancer.Ingress) < 1 {
				servicesReady = false
			}
		}
	}
	return servicesReady, nil
}

func (broker *PksServiceBroker) podsReady(instanceID string, cluster k8s.Cluster) (string, bool, error) {
	podList, err := cluster.ListPods(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return "", false, err
	}

	podsReady := true
	message := ""
	for _, pod := range podList.Items {

		if broker.podIsJob(pod) {
			if pod.Status.Phase != "Succeeded" {
				podsReady = false
				for _, condition := range pod.Status.Conditions {
					if condition.Message != "" {
						message = message + condition.Message + "\n"
					}
				}
			}
		} else {
			if pod.Status.Phase != "Running" {
				podsReady = false
				for _, condition := range pod.Status.Conditions {
					if condition.Message != "" {
						message = message + condition.Message + "\n"
					}
				}
			}
		}
	}

	message = strings.TrimSpace(message)

	return message, podsReady, nil
}

func (broker *PksServiceBroker) podIsJob(pod api_v1.Pod) bool {
	for key := range pod.ObjectMeta.Labels {
		if key == "job-name" {
			return true
		}
	}
	return false
}

func (broker *PksServiceBroker) getNamespace(instanceID string) string {
	return "kibosh-" + instanceID
}

func (broker *PksServiceBroker) getServiceName(chart *my_helm.MyChart) string {
	return chart.Metadata.Name
}

func (broker *PksServiceBroker) getServiceID(chart *my_helm.MyChart) string {
	return uuid.NewSHA1(uuid.NameSpace_OID, []byte(broker.getServiceName(chart))).String()
}
