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

	"github.com/cf-platform-eng/kibosh/pkg/bazaar"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/sirupsen/logrus"
)

func main() {
	bazaarLogger := logrus.New()
	bazaarLogger.SetLevel(logrus.DebugLevel)
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
