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
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/cloudfoundry-community/go-cfclient"
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

	repo := repository.NewRepository(conf.HelmChartDir, conf.RegistryConfig.Server, true, brokerLogger)
	charts, err := repo.LoadCharts()
	if err != nil {
		brokerLogger.Fatal("Unable to load charts", err)
	}
	brokerLogger.Info(fmt.Sprintf("Brokering charts %s", charts))

	var cfAPIClient *cfclient.Client
	if conf.CFClientConfig.HasCFClientConfig() {
		cfAPIClient, err = cfclient.NewClient(&cfclient.Config{
			ApiAddress:        conf.CFClientConfig.ApiAddress,
			Username:          conf.CFClientConfig.Username,
			Password:          conf.CFClientConfig.Password,
			SkipSslValidation: conf.CFClientConfig.SkipSslValidation,
		})
		if err != nil {
			brokerLogger.Fatal("Unable to load charts", err)
		}
	}

	operatorRepo := repository.NewRepository(conf.OperatorDir, conf.RegistryConfig.Server, false, brokerLogger)
	operatorCharts, err := operatorRepo.LoadCharts()
	if err != nil {
		if !os.IsNotExist(err) {
			brokerLogger.Fatal("Unable to load operators", err)
		}
	}
	brokerLogger.Info(fmt.Sprintf("Loaded operator charts: %s", operatorCharts))

	clusterFactory := k8s.NewClusterFactory(*conf.ClusterCredentials)
	helmClientFactory := helm.NewHelmClientFactory(conf.HelmTLSConfig, brokerLogger)
	serviceAccountInstallerFactory := k8s.NewServiceAccountInstallerFactory(brokerLogger)

	serviceBroker := broker.NewPksServiceBroker(conf, clusterFactory, helmClientFactory, serviceAccountInstallerFactory, charts, operatorCharts, brokerLogger)
	brokerCredentials := brokerapi.BrokerCredentials{
		Username: conf.AdminUsername,
		Password: conf.AdminPassword,
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)
	http.Handle("/", brokerAPI)

	repositoryAPI := repository.NewAPI(serviceBroker, repo, cfAPIClient, conf, brokerLogger)
	authFilter := httphelpers.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)
	http.Handle("/reload_charts", authFilter.Filter(
		repositoryAPI.ReloadCharts(),
	))

	brokerLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	brokerLogger.Fatal("http-listen", err)
}
