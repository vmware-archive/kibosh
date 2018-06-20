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

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/config"
	my_helm "github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/helm"
	hapi_release "k8s.io/helm/pkg/proto/hapi/release"
)

const registrySecretName = "registry-secret"

// PksServiceBroker contains values passed in from configuration necessary for broker's work.
type PksServiceBroker struct {
	Logger         lager.Logger
	registryConfig *config.RegistryConfig

	cluster      k8s.Cluster
	myHelmClient my_helm.MyHelmClient
	chartsMap    map[string]*my_helm.MyChart
}

func NewPksServiceBroker(registryConfig *config.RegistryConfig, cluster k8s.Cluster, myHelmClient my_helm.MyHelmClient, charts []*my_helm.MyChart, logger lager.Logger) *PksServiceBroker {
	broker := &PksServiceBroker{
		Logger:         logger,
		registryConfig: registryConfig,

		cluster:      cluster,
		myHelmClient: myHelmClient,
	}
	broker.chartsMap = map[string]*my_helm.MyChart{}
	for _, chart := range charts {
		broker.chartsMap[broker.getServiceID(chart)] = chart
	}

	return broker
}

func (broker *PksServiceBroker) Services(ctx context.Context) []brokerapi.Service {
	serviceCatalog := []brokerapi.Service{}

	for _, chart := range broker.chartsMap {
		plans := []brokerapi.ServicePlan{}
		for _, plan := range chart.Plans {

			plans = append(plans, brokerapi.ServicePlan{
				ID:          broker.getServiceID(chart) + "-" + plan.Name,
				Name:        plan.Name,
				Description: plan.Description,
			})
		}

		serviceCatalog = append(serviceCatalog, brokerapi.Service{
			ID:          broker.getServiceID(chart),
			Name:        broker.getServiceName(chart),
			Description: chart.Metadata.Description,
			Bindable:    true,

			Plans: plans,
		})
	}

	return serviceCatalog
}

func (broker *PksServiceBroker) Provision(ctx context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, error) {
	if !asyncAllowed {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrAsyncRequired
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
	_, err := broker.cluster.CreateNamespace(&namespace)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	if broker.registryConfig.HasRegistryConfig() {
		privateRegistrySetup := k8s.NewPrivateRegistrySetup(namespaceName, "default", broker.cluster, broker.registryConfig)
		err := privateRegistrySetup.Setup()
		if err != nil {
			return brokerapi.ProvisionedServiceSpec{}, err
		}
	}

	planName := strings.TrimPrefix(details.PlanID, details.ServiceID+"-")

	chart := broker.chartsMap[details.ServiceID]
	if chart == nil {
		return brokerapi.ProvisionedServiceSpec{}, errors.New(fmt.Sprintf("Chart not found for [%s]", details.ServiceID))
	}
	_, err = broker.myHelmClient.InstallChart(chart, namespaceName, planName, helm.ReleaseName(namespaceName))
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, err
	}

	return brokerapi.ProvisionedServiceSpec{
		IsAsync:       true,
		OperationData: "provision",
	}, nil
}

func (broker *PksServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	_, err := broker.myHelmClient.DeleteRelease(broker.getNamespace(instanceID))
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	err = broker.cluster.DeleteNamespace(broker.getNamespace(instanceID), &meta_v1.DeleteOptions{})
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{
		IsAsync:       true,
		OperationData: "deprovision",
	}, nil
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
	}

	if brokerStatus != brokerapi.Succeeded {
		return brokerapi.LastOperation{
			State:       brokerStatus,
			Description: description,
		}, nil
	} else {
		servicesReady, err := broker.servicesReady(instanceID)
		if err != nil {
			return brokerapi.LastOperation{}, err
		}
		if !servicesReady {
			return brokerapi.LastOperation{
				State:       brokerapi.InProgress,
				Description: "service deployment load balancer in progress",
			}, nil
		}

		message, podsReady, err := broker.podsReady(instanceID)
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
}

func (broker *PksServiceBroker) servicesReady(instanceID string) (bool, error) {
	services, err := broker.cluster.ListServices(broker.getNamespace(instanceID), meta_v1.ListOptions{})
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

func (broker *PksServiceBroker) podsReady(instanceID string) (string, bool, error) {
	podList, err := broker.cluster.ListPods(broker.getNamespace(instanceID), meta_v1.ListOptions{})
	if err != nil {
		return "", false, err
	}

	podsReady := true
	message := ""
	for _, pod := range podList.Items {
		if pod.Status.Phase != "Running" {
			podsReady = false
			for _, condition := range pod.Status.Conditions {
				if condition.Message != "" {
					message = message + condition.Message + "\n"
				}
			}
		}
	}
	message = strings.TrimSpace(message)

	return message, podsReady, nil
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
