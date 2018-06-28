package main

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
)

func main() {
	bazaarLogger := lager.NewLogger("bazaar")
	bazaarLogger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	bazaarLogger.Info("Starting PKS Bazaar")

	conf, err := bazaar.ParseConfig()
	if err != nil {
		bazaarLogger.Fatal("Loading config file", err)
	}

	repo := repository.NewRepository(conf.HelmChartDir, conf.RegistryConfig.Server, bazaarLogger)
	bazaarAPI := bazaar.NewAPI(repo, conf.KiboshConfig, bazaarLogger)
	authFilter := httphelpers.NewAuthFilter(conf.AdminUsername, conf.AdminPassword)

	// When registering *only* the trailing slash, for the non-trailing slash url,
	// ServeMux returns a 301 (not 307), so client flips to GET
	http.Handle("/charts", authFilter.Filter(
		bazaarAPI.Charts(),
	))
	http.Handle("/charts/", authFilter.Filter(
		bazaarAPI.Charts(),
	))

	bazaarLogger.Info(fmt.Sprintf("Listening on %v", conf.Port))
	err = http.ListenAndServe(fmt.Sprintf(":%v", conf.Port), nil)
	bazaarLogger.Fatal("http-listen", err)
}
