package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/auth"
	"github.com/cf-platform-eng/kibosh/bazaar"
	"github.com/cf-platform-eng/kibosh/repository"
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
	bazaarAPI := bazaar.NewAPI(repo, conf.KiboshConfig, bazaarLogger)
	authFilter := auth.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)
	http.Handle("/charts/", authFilter.Filter(
		bazaarAPI.ListCharts(),
	))
	if true {
		panic("foo")
	}

	//todo: broken
	//post request type
	//http.Handle("/charts/", authFilter.Filter(
	//	bazaarAPI.SaveChart(),
	//))
	//todo: broken
	//delete request type
	//http.Handle("/charts/", authFilter.Filter(
	//	bazaarAPI.DeleteChart(),
	//))
	bazaarLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	bazaarLogger.Fatal("http-listen", err)
}
