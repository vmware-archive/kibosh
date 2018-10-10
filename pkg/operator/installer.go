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

package operator

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	api_v1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

type PksOperator struct {
	Logger         *logrus.Logger
	registryConfig *config.RegistryConfig

	cluster      k8s.Cluster
	myHelmClient my_helm.MyHelmClient
	operatorsMap map[string]*my_helm.MyChart
}

func NewInstaller(registryConfig *config.RegistryConfig, cluster k8s.Cluster, myHelmClient my_helm.MyHelmClient, logger *logrus.Logger) *PksOperator {
	operator := &PksOperator{
		Logger:         logger,
		registryConfig: registryConfig,

		cluster:      cluster,
		myHelmClient: myHelmClient,
	}

	return operator
}

func (operator *PksOperator) InstallCharts(operatorCharts []*my_helm.MyChart) error {
	for _, operatorChart := range operatorCharts {
		err := operator.Install(operatorChart)
		if err != nil {
			return err
		}
	}
	return nil
}

func (operator *PksOperator) Install(chart *my_helm.MyChart) error {

	namespaceName := chart.String() + "-kibosh-operator"

	releases, err := operator.myHelmClient.ListReleases()
	if err != nil {
		return err
	}
	if releases != nil && exists(namespaceName, releases) {
		operator.Logger.Info(fmt.Sprintf("Operator " + chart.String() + " is already installed. Not installing."))
		return nil
	}

	operator.Logger.Info(fmt.Sprintf("Installing operator chart: " + chart.String()))

	_, err = operator.cluster.GetNamespace(namespaceName, &meta_v1.GetOptions{})
	if err != nil {
		statusErr, ok := err.(*api_errors.StatusError)
		if !ok {
			return err
		}
		if statusErr.ErrStatus.Reason == meta_v1.StatusReasonNotFound {
			namespace := api_v1.Namespace{
				Spec: api_v1.NamespaceSpec{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name: namespaceName,
					Labels: map[string]string{
						"kibosh": "installed",
					},
				},
			}
			_, err := operator.cluster.CreateNamespace(&namespace)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if operator.registryConfig.HasRegistryConfig() {
		privateRegistrySetup := k8s.NewPrivateRegistrySetup(namespaceName, "default", operator.cluster, operator.registryConfig)
		err := privateRegistrySetup.Setup()
		if err != nil {
			return err
		}
	}

	_, err = operator.myHelmClient.InstallOperator(chart, namespaceName)
	if err != nil {
		return err
	}

	return nil
}

func exists(name string, releases *rls.ListReleasesResponse) bool {
	for _, release := range releases.Releases {
		if release.Name == name {
			return true
		}
	}
	return false
}
