package main

import (
	"code.cloudfoundry.org/lager"
	"fmt"
	"github.com/cf-platform-eng/kibosh/auth"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/cf-platform-eng/kibosh/repository"
	"net/http"
	"os"
)

func main() {

	bazaarLogger := lager.NewLogger("bazaar")
	bazaarLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	bazaarLogger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))
	bazaarLogger.Info("Starting PKS Bazaar")

	conf, err := bazaar.ParseConfig()
	if err != nil {
		bazaarLogger.Fatal("Loading config file", err)
	}

	repo := repository.NewRepository(conf.HelmChartDir, conf.RegistryConfig.Server, bazaarLogger)
	bazaarAPI := bazaar.NewAPI(repo, bazaarLogger)
	authFilter := auth.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)
	http.Handle("/charts/", authFilter.Filter(
		bazaarAPI.ListCharts(),
	))

	http.Handle("/charts/create", authFilter.Filter(
		bazaarAPI.CreateChart(),
	))
	bazaarLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	bazaarLogger.Fatal("http-listen", err)
}
