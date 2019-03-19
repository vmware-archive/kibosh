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
	"github.com/cf-platform-eng/kibosh/pkg/broker"
	"github.com/cf-platform-eng/kibosh/pkg/config"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/cf-platform-eng/kibosh/pkg/k8s"
	"github.com/cf-platform-eng/kibosh/pkg/logger"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	kiboshLogger := logger.NewSplitLogger(os.Stdout, os.Stderr)
	kiboshLogger.Info("Starting PKS Generic Broker")

	conf, err := config.Parse()
	if err != nil {
		kiboshLogger.Fatal("Loading config file", err)
	}

	repo := repository.NewRepository(conf.HelmChartDir, conf.RegistryConfig.Server, kiboshLogger)
	charts, err := repo.GetCharts()
	if err != nil {
		kiboshLogger.Fatal("Unable to load charts", err)
	}
	kiboshLogger.Info(fmt.Sprintf("Brokering charts %s", charts))

	var cfAPIClient *cfclient.Client
	if conf.CFClientConfig.HasCFClientConfig() {
		cfAPIClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress:        conf.CFClientConfig.ApiAddress,
			Username:          conf.CFClientConfig.Username,
			Password:          conf.CFClientConfig.Password,
			SkipSslValidation: conf.CFClientConfig.SkipSslValidation,
		})
		if err != nil {
			kiboshLogger.Fatal("Unable to build cf client", err)
		}
	}

	operatorRepo := repository.NewRepository(conf.OperatorDir, conf.RegistryConfig.Server, kiboshLogger)
	operatorCharts, err := operatorRepo.GetCharts()
	if err != nil {
		if !os.IsNotExist(err) {
			kiboshLogger.Fatal("Unable to load operators", err)
		}
	}
	kiboshLogger.Info(fmt.Sprintf("Loaded operator charts: %s", operatorCharts))

	clusterFactory := k8s.NewClusterFactory(*conf.ClusterCredentials)
	helmClientFactory := helm.NewHelmClientFactory(conf.HelmTLSConfig, conf.TillerNamespace, kiboshLogger)
	serviceAccountInstallerFactory := k8s.NewServiceAccountInstallerFactory(conf.TillerNamespace, kiboshLogger)

	err = broker.PrepareDefaultCluster(conf, clusterFactory, helmClientFactory, serviceAccountInstallerFactory, helm.InstallerFactoryDefault, kiboshLogger, operatorCharts)

	if err != nil {
		kiboshLogger.Fatal("Unable to prepare default cluster", err)
	}

	serviceBroker := broker.NewPksServiceBroker(conf, clusterFactory, helmClientFactory, serviceAccountInstallerFactory, helm.InstallerFactoryDefault, repo, operatorCharts, kiboshLogger)
	brokerCredentials := brokerapi.BrokerCredentials{
		Username: conf.AdminUsername,
		Password: conf.AdminPassword,
	}

	brokerLogger := lager.NewLogger("broker")
	brokerLogger.RegisterSink(logger.NewLogrusSink(kiboshLogger))

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)
	http.Handle("/", brokerAPI)

	repositoryAPI := repository.NewAPI(repo, cfAPIClient, conf, kiboshLogger)
	authFilter := httphelpers.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)
	http.Handle("/reload_charts", authFilter.Filter(
		repositoryAPI.ReloadCharts(),
	))

	kiboshLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	kiboshLogger.Fatal("http-listen", err)
}
