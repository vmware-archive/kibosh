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
	"github.com/Sirupsen/logrus"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	my_helm "github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/operator"
)

func PrepareDefaultCluster(
	config *config.Config,
	clusterFactory k8s.ClusterFactory,
	helmClientFactory my_helm.HelmClientFactory,
	serviceAccountInstallerFactory k8s.ServiceAccountInstallerFactory,
	installerFactory my_helm.InstallerFactory,
	logger *logrus.Logger,
	operators []*my_helm.MyChart,
) error {
	cluster, err := clusterFactory.DefaultCluster()

	if err == nil {
		helmClient := helmClientFactory.HelmClient(cluster)
		serviceAccountInstaller := serviceAccountInstallerFactory.ServiceAccountInstaller(cluster)

		return PrepareCluster(config, cluster, helmClient, serviceAccountInstaller, installerFactory, operators, logger)
	}

	return err
}

func PrepareCluster(config *config.Config, cluster k8s.Cluster, helmClient my_helm.MyHelmClient, serviceAccountInstaller k8s.ServiceAccountInstaller, installerFactory my_helm.InstallerFactory, operators []*my_helm.MyChart, logger *logrus.Logger) error {
	err := serviceAccountInstaller.Install()
	if err != nil {
		logger.WithError(err).Error("failed installing service account")
		return err
	}

	helmInstaller := installerFactory(config, cluster, helmClient, logger)
	err = helmInstaller.Install()
	if err != nil {
		logger.WithError(err).Error("failed installing helm")
		return err
	}

	// Install each operator chart.
	operatorInstaller := operator.NewInstaller(config.RegistryConfig, cluster, helmClient, logger)
	err = operatorInstaller.InstallCharts(operators)
	if err != nil {
		logger.WithError(err).Error("failed installing operator charts")
		return err
	}

	return nil
}
