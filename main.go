package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/pks-generic-broker/broker"
	"github.com/cf-platform-eng/pks-generic-broker/config"

	"github.com/pivotal-cf/brokerapi"
)

var logger *log.Logger

func main() {

	brokerLogger := lager.NewLogger("pks-generic-broker")
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	brokerLogger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	brokerLogger.Info("Starting PKS Generic Broker")

	c, err := config.Parse()
	if err != nil {
		brokerLogger.Fatal("Loading config file", err)
	}

	brokerCredentials := brokerapi.BrokerCredentials{
		Username: c.AdminUsername,
		Password: c.AdminPassword,
	}

	serviceBroker := &broker.PksServiceBroker{
		HelmChartDir: c.HelmChartDir,
		ServiceID:    c.ServiceID,
	}

	brokerAPI := brokerapi.New(serviceBroker, brokerLogger, brokerCredentials)

	http.Handle("/", brokerAPI)

	err = http.ListenAndServe(fmt.Sprintf(":%v", c.Port), nil)
	brokerLogger.Fatal("http-listen", err)

}
