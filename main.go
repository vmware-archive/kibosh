package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/broker"
	"github.com/cf-platform-eng/kibosh/config"

	"github.com/pivotal-cf/brokerapi"
)

func main() {

	brokerLogger := lager.NewLogger("kibosh")
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
