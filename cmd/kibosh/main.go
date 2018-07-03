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

package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/auth"
	"github.com/cf-platform-eng/kibosh/pkg/broker"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/operator"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	brokerLogger := lager.NewLogger("kibosh")
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	brokerLogger.Info("Starting PKS Generic Broker")

	conf, err := config.Parse()
	if err != nil {
		brokerLogger.Fatal("Loading config file", err)
	}

	cluster, err := k8s.NewCluster(conf.ClusterCredentials)
	if err != nil {
		brokerLogger.Fatal("Error setting up k8s cluster", err)
	}

	repo := repository.NewRepository(conf.HelmChartDir, conf.RegistryConfig.Server, brokerLogger)
	charts, err := repo.LoadCharts()
	if err != nil {
		brokerLogger.Fatal("Unable to load charts", err)
	}
	brokerLogger.Info(fmt.Sprintf("Brokering charts %s", charts))

	operatorRepo := repository.NewRepository(conf.OperatorDir, conf.RegistryConfig.Server, brokerLogger)
	operatorCharts, err := operatorRepo.LoadCharts()
	if err != nil {
		brokerLogger.Fatal("Unable to load operators", err)
	}
	brokerLogger.Info(fmt.Sprintf("Brokering operators %s", operatorCharts))

	myHelmClient := helm.NewMyHelmClient(cluster, brokerLogger)
	serviceBroker := broker.NewPksServiceBroker(conf.RegistryConfig, cluster, myHelmClient, charts, brokerLogger)
	brokerCredentials := brokerapi.BrokerCredentials{
		Username: conf.AdminUsername,
		Password: conf.AdminPassword,
	}

	myServiceAccountInstaller := k8s.NewServiceAccountInstaller(cluster, brokerLogger)
	err = myServiceAccountInstaller.Install()
	if err != nil {
		brokerLogger.Fatal("Error creating service account", err)
	}

	helmInstaller := helm.NewInstaller(conf.RegistryConfig, cluster, myHelmClient, brokerLogger)
	err = helmInstaller.Install()
	if err != nil {
		brokerLogger.Fatal("Error installing helm", err)
	}

	// Install each operator chart.
	operatorInstaller := operator.NewInstaller(conf.RegistryConfig, cluster, myHelmClient, brokerLogger)
	err = operatorInstaller.InstallCharts(operatorCharts)
	if err != nil {
		brokerLogger.Fatal("Error installing operator", err)
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)
	http.Handle("/", brokerAPI)

	repositoryAPI := repository.NewAPI(serviceBroker, repo, brokerLogger)
	authFilter := auth.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)
	http.Handle("/reload_charts", authFilter.Filter(
		repositoryAPI.ReloadCharts(),
	))

	brokerLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	brokerLogger.Fatal("http-listen", err)
}
