package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/config"
	"github.com/cf-platform-eng/kibosh/helm"
	"github.com/cf-platform-eng/kibosh/k8s"
	"github.com/pivotal-cf/brokerapi"
)

func main() {
	brokerLogger := lager.NewLogger("kibosh")
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	brokerLogger.Info("Starting PKS Generic Broker")

	conf, err := config.Parse()
	if err != nil {
		brokerLogger.Fatal("Loading config file", err)
	}

	cluster, err := k8s.NewCluster(conf.ClusterCredentials)
	if err != nil {
		brokerLogger.Fatal("Error setting up k8s cluster", err)
	}

	myChart, err := helm.NewChart(conf.HelmChartDir, conf.RegistryConfig.Server)
	if err != nil {
		brokerLogger.Fatal("Helm chart failed to load", err)
	}

	myHelmClient := helm.NewMyHelmClient(myChart, cluster, brokerLogger)
	serviceBroker := broker.NewPksServiceBroker(
		conf.ServiceID, conf.ServiceName, conf.RegistryConfig, cluster, myHelmClient, myChart, brokerLogger,
	)
	brokerCredentials := brokerapi.BrokerCredentials{
		Username: conf.AdminUsername,
		Password: conf.AdminPassword,
	}

	myServiceAccountInstaller := k8s.NewServiceAccountInstaller(cluster, brokerLogger)
	err = myServiceAccountInstaller.Install()
	if err != nil {
		brokerLogger.Fatal("Error creating service account", err)
	}

	helmInstaller := helm.NewInstaller(cluster, myHelmClient, brokerLogger)
	err = helmInstaller.Install()
	if err != nil {
		brokerLogger.Fatal("Error installing helm", err)
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)

	http.Handle("/", brokerAPI)

	brokerLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	brokerLogger.Fatal("http-listen", err)
}
