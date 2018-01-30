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

	cluster, err := k8s.NewCluster(conf.KuboODBVCAP)
	if err != nil {
		brokerLogger.Fatal("Error setting up k8s cluster", err)
	}
	myHelmClient := helm.NewMyHelmClient(cluster, brokerLogger)
	//myHelmClient := helm.NewMyHelmClient(nil, brokerLogger)
	println(myHelmClient)
	serviceBroker := broker.NewPksServiceBroker(
		//conf.HelmChartDir, conf.ServiceID, cluster, myHelmClient,
		conf.HelmChartDir, conf.ServiceID, nil, nil,
	)
	brokerCredentials := brokerapi.BrokerCredentials{
		Username: conf.AdminUsername,
		Password: conf.AdminPassword,
	}

	//helmInstaller := helm.NewInstaller(cluster, myHelmClient, brokerLogger)
	//err = helmInstaller.Install()
	//if err != nil {
	//	brokerLogger.Fatal("Error setting installing helm", err)
	//}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)

	http.Handle("/", brokerAPI)

	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	brokerLogger.Fatal("http-listen", err)
}
