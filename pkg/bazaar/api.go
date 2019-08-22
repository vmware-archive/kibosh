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

package bazaar

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cf-platform-eng/kibosh/pkg/broker"
	"github.com/cf-platform-eng/kibosh/pkg/credstore"
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"gopkg.in/yaml.v2"

	"strings"

	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type API interface {
	Charts() http.Handler
	ListCharts(w http.ResponseWriter, r *http.Request) error
	SaveChart(w http.ResponseWriter, r *http.Request) error
	DeleteChart(w http.ResponseWriter, r *http.Request) error
}

type api struct {
	repo         repository.Repository
	kiboshConfig *KiboshConfig
	credStore    credstore.CredStore
	logger       *logrus.Logger
}

func NewAPI(repo repository.Repository, kiboshConfig *KiboshConfig, credStore credstore.CredStore, logger *logrus.Logger) API {
	return &api{
		repo:         repo,
		kiboshConfig: kiboshConfig,
		credStore:    credStore,
		logger:       logger,
	}
}

type DisplayChart struct {
	Name    string   `json:"name"`
	Plans   []string `json:"plans"`
	Version string   `json:"version"`
}

type DisplayResponse struct {
	Message string `json:"message"`
}

func (api *api) Charts() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		switch r.Method {
		case "GET":
			err = api.ListCharts(w, r)
			break
		case "POST":
			err = api.SaveChart(w, r)
			break
		case "DELETE":
			err = api.DeleteChart(w, r)
			break
		default:
			w.WriteHeader(405)
			w.Header().Set("Allow", "GET, POST, DELETE")
		}

		if err != nil {
			api.logger.WithError(err).Error("Error writing response")
		}
	})
}

func (api *api) ListCharts(w http.ResponseWriter, r *http.Request) error {
	charts, err := api.repo.GetCharts()
	if err != nil {
		api.logger.WithError(err).Error("Unable to load charts")
		api.ServerError(500, errors.Wrap(err, "Unable to load charts").Error(), w)
	} else {
		var displayCharts []DisplayChart
		for _, chart := range charts {
			var plans []string
			for _, plan := range chart.Plans {
				plans = append(plans, plan.Name)
			}
			displayCharts = append(displayCharts, DisplayChart{
				Name:    chart.Metadata.Name,
				Version: chart.Metadata.Version,
				Plans:   plans,
			})
		}
		return api.WriteJSONResponse(w, displayCharts)
	}
	return nil
}

func (api *api) SaveChart(w http.ResponseWriter, r *http.Request) error {
	err := api.saveChartToRepository(r)
	if err != nil {
		api.ServerError(500, errors.Wrap(err, "Unable to save charts").Error(), w)
		return nil
	}

	err = api.triggerKiboshReload()
	if err != nil {
		//todo: retry? rollback? what's on disk now doesn't match Kibosh
		api.ServerError(500, errors.Wrap(err, "Chart persisted, but Kibosh reload failed").Error(), w)
		return nil
	}
	return api.WriteJSONResponse(w, DisplayResponse{Message: "Chart saved"})
}

func (api *api) DeleteChart(w http.ResponseWriter, r *http.Request) error {
	chartName, err := getUrlPart(1, r)
	if err != nil {
		api.ServerError(500, errors.Wrap(err, "Unable to parse url path parameters").Error(), w)
		return nil
	}

	charts, err := api.repo.GetCharts()
	if err != nil {
		api.ServerError(500, errors.Wrap(err, "Unable to delete chart").Error(), w)
		return nil
	}
	if len(charts) == 1 {
		if charts[0].Metadata.Name == chartName {
			api.ServerError(400, "Cannot remove last chart (this would result in invalid catalog)", w)
			return nil
		}
	}

	err = api.repo.DeleteChart(chartName)
	if err != nil {
		api.ServerError(500, errors.Wrap(err, "Unable to delete chart").Error(), w)
		return nil
	}

	err = api.triggerKiboshReload()
	if err != nil {
		//todo: retry? rollback? what's on disk now doesn't match Kibosh
		api.ServerError(500, errors.Wrap(err, "Chart deleted, but Kibosh reload failed").Error(), w)
		return nil
	}
	return api.WriteJSONResponse(w, DisplayResponse{
		Message: fmt.Sprintf("Chart [%v] deleted", chartName),
	})
}

func (api *api) WriteJSONResponse(w http.ResponseWriter, body interface{}) error {
	serialized, err := json.Marshal(body)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write([]byte(serialized))

	return err
}

func getUrlPart(position int, r *http.Request) (string, error) {
	parts := strings.Split(r.URL.Path, "/")
	print()
	if parts[0] == "" {
		parts = parts[1:]
	}
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts)-1 < position {
		return "", errors.New("url didn't have required param")
	}
	return parts[position], nil
}

func (api *api) saveChartToRepository(r *http.Request) error {
	err := r.ParseMultipartForm(1000000)
	if err != nil {
		api.logger.WithError(err).Error("SaveChart: Couldn't parse the multipart form request")
		return err
	}
	formdata := r.MultipartForm

	files := formdata.File["chart"]
	for i := range files {
		file, err := files[i].Open()
		if err != nil {
			api.logger.WithError(err).Error("SaveChart: Couldn't read request POST form data")
			return err
		}

		chartDir, err := ioutil.TempDir("", "chart-")
		chartPath := filepath.Join(chartDir, files[i].Filename)
		f, err := os.OpenFile(chartPath, os.O_WRONLY|os.O_CREATE, 0666)

		if err != nil {
			api.logger.WithError(err).Error("SaveChart: Couldn't write on disk ")
			return err
		}

		buffer := make([]byte, 1000000)
		_, err = io.CopyBuffer(f, file, buffer)
		if err != nil {
			api.logger.WithError(err).Error("SaveChart: Couldn't copy file to buffer")
			return err
		}

		if api.credStore != nil {
			chartPath, err = api.removeAndStorePlanDetails(chartPath)
			if err != nil {
				return errors.Wrap(err, "unable to store plans in CredHub")
			}
		}
		err = api.repo.SaveChart(chartPath)
		if err != nil {
			api.logger.WithError(err).Error("SaveChart: Couldn't save the chart")
			return err
		}

		err = file.Close()
		if err != nil {
			api.logger.WithError(err).Error("error closing source file")
			return err
		}
		err = f.Close()
		if err != nil {
			api.logger.WithError(err).Error("error closing target file")
			return err
		}
	}
	return nil
}

func (api *api) triggerKiboshReload() error {
	client := &http.Client{}
	kiboshURL := fmt.Sprintf("%v/reload_charts", api.kiboshConfig.Server)
	req, err := http.NewRequest("GET", kiboshURL, nil)
	if err != nil {
		api.logger.WithError(err).Error("reload_charts failed")
		return err
	}

	httphelpers.AddBasicAuthHeader(req, api.kiboshConfig.User, api.kiboshConfig.Pass)
	res, err := client.Do(req)
	if err != nil {
		api.logger.WithError(err).Error("Couldn't call kibosh to update")
		return err
	}
	if res.StatusCode != 200 {
		err = errors.Errorf("kibosh return non 200 status code [%v]", res.StatusCode)
		api.logger.WithError(err).Error("Error triggering Kibosh reload")
		return err
	}
	return nil
}

func (api *api) ServerError(code int, message string, w http.ResponseWriter) {
	w.WriteHeader(code)
	_, err := w.Write([]byte(message))
	if err != nil {
		api.logger.WithError(err).Error("error writing server error response")
	}
}

func (api *api) removeAndStorePlanDetails(chartPath string) (string, error) {
	chartFile, err := os.Open(chartPath)
	if err != nil {
		return "", err
	}
	defer chartFile.Close()

	gzipReader, err := gzip.NewReader(chartFile)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)

	tarFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Wrap(err, "cannot create temp file for chart")
	}
	defer tarFile.Close()
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	plans := []helm.Plan{}
	planFiles := map[string][]byte{}
	chartYaml := map[string]string{}
	var chartName string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		if strings.HasSuffix(header.Name, "plans.yaml") || strings.HasSuffix(header.Name, "plans.yml") {
			api.logger.Info(fmt.Sprintf("plans.yaml found, reading from plans.yaml"))
			plansBytes, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}

			err = yaml.Unmarshal(plansBytes, &plans)
			if err != nil {
				return "", errors.Wrap(err, "Error unmarshalling plan")
			}
			err = tarWriter.WriteHeader(header)
			if err != nil {
				return "", errors.Wrap(err, "error writing header to tar file")
			}
			_, err = tarWriter.Write(plansBytes)
			if err != nil {
				return "", errors.Wrap(err, "error writing body to tar file")
			}

		} else if strings.Contains(header.Name, "Chart.yaml") {
			chartYamlBytes, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}

			err = yaml.Unmarshal(chartYamlBytes, &chartYaml)
			if err != nil {
				return "", err
			}

			chartName = chartYaml["name"]
			err = tarWriter.WriteHeader(header)
			if err != nil {
				return "", errors.Wrap(err, "error writing header to tar file")
			}
			_, err = tarWriter.Write(chartYamlBytes)
			if err != nil {
				return "", errors.Wrap(err, "error writing body to tar file")
			}
		} else if strings.Contains(header.Name, "/plans") {
			planFileBytes, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}
			planFiles[header.Name] = planFileBytes
		} else {
			err = tarWriter.WriteHeader(header)
			if err != nil {
				return "", errors.Wrap(err, "error writing header to tar file")
			}
			_, err = io.Copy(tarWriter, tarReader)
			if err != nil {
				return "", errors.Wrap(err, "error writing body to tar file")
			}
		}
	}

	for _, plan := range plans {
		keyPath := fmt.Sprintf("/c/%s/%s/%s/values", broker.CredhubClientIdentifier, chartName, plan.Name)
		_, err = api.credStore.Put(keyPath, planFiles[chartName+"/plans/"+plan.ValuesFile])
		if err != nil {
			return "", errors.Wrap(err, "failed to save chart plan to CredHub")
		}
		if plan.CredentialsPath != "" {
			credPath := fmt.Sprintf("/c/%s/%s/%s/cluster-creds", broker.CredhubClientIdentifier, chartName, plan.Name)
			_, err = api.credStore.Put(credPath, planFiles[chartName+"/plans/"+plan.CredentialsPath])
			if err != nil {
				return "", errors.Wrap(err, "failed to save cluster credential to CredHub")
			}
		}

	}

	return tarFile.Name(), err
}
