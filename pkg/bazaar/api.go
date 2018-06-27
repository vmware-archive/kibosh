package bazaar

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/pkg/auth"
	"github.com/cf-platform-eng/kibosh/pkg/repository"
	"github.com/pkg/errors"
	"strings"
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
	logger       lager.Logger
}

func NewAPI(repo repository.Repository, kiboshConfig *KiboshConfig, l lager.Logger) API {
	return &api{
		repo:         repo,
		kiboshConfig: kiboshConfig,
		logger:       l,
	}
}

type DisplayChart struct {
	Name      string   `json:"name"`
	Plans     []string `json:"plans"`
	Chartpath string   `json:"chartpath"`
	Version   string   `json:"version"`
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
			api.logger.Error("Error writing response", err)
		}
	})
}

func (api *api) ListCharts(w http.ResponseWriter, r *http.Request) error {
	charts, err := api.repo.LoadCharts()
	if err != nil {
		api.logger.Error("Unable to load charts", err)
		api.ServerError(500, "Unable to load charts", w)
	} else {
		var displayCharts []DisplayChart
		for _, chart := range charts {
			var plans []string
			for _, plan := range chart.Plans {
				plans = append(plans, plan.Name)
			}
			displayCharts = append(displayCharts, DisplayChart{
				Name:      chart.Metadata.Name,
				Version:   chart.Metadata.Version,
				Plans:     plans,
				Chartpath: chart.Chartpath,
			})
		}
		return api.WriteJSONResponse(w, displayCharts)
	}
	return nil
}

func (api *api) SaveChart(w http.ResponseWriter, r *http.Request) error {
	err := api.saveChartToRepository(r)
	if err != nil {
		api.ServerError(500, "Unable to save charts", w)
		return nil
	}

	err = api.triggerKiboshReload()
	if err != nil {
		//todo: retry? rollback? what's on disk now doesn't match Kibosh
		api.ServerError(500, "Chart persisted, but Kibosh reload failed", w)
		return nil
	}
	return api.WriteJSONResponse(w, map[string]interface{}{"message": "Chart saved"})
}

func (api *api) DeleteChart(w http.ResponseWriter, r *http.Request) error {
	chartName, err := getUrlPart(1, r)
	if err != nil {
		api.ServerError(500, "Unable to parse url path parameters", w)
		return nil
	}

	err = api.repo.DeleteChart(chartName)
	if err != nil {
		api.ServerError(500, "Unable to delete chart", w)
		return nil
	}

	err = api.triggerKiboshReload()
	if err != nil {
		//todo: retry? rollback? what's on disk now doesn't match Kibosh
		api.ServerError(500, "Chart deleted, but Kibosh reload failed", w)
		return nil
	}
	return api.WriteJSONResponse(w, map[string]interface{}{
		"message": fmt.Sprintf("Chart [%v] deleted", chartName),
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
	r.ParseMultipartForm(1000000)
	file, handler, err := r.FormFile("chart")
	if err != nil {
		api.logger.Error("SaveChart: Couldn't read request POST form data", err)
		return err
	}
	defer file.Close()
	chartPath, err := ioutil.TempDir("", "chart-")
	f, err := os.OpenFile(filepath.Join(chartPath, handler.Filename), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		api.logger.Error("SaveChart: Couldn't write on disk ", err)
		return err
	}
	defer f.Close()
	buffer := make([]byte, 1000000)
	io.CopyBuffer(f, file, buffer)

	err = api.repo.SaveChart(filepath.Join(chartPath, handler.Filename))
	if err != nil {
		api.logger.Error("SaveChart: Couldn't save the chart", err)
		return err
	}
	return nil
}

func (api *api) triggerKiboshReload() error {
	client := &http.Client{}
	kiboshURL := fmt.Sprintf("%v/reload_charts", api.kiboshConfig.Server)
	req, err := http.NewRequest("GET", kiboshURL, nil)
	if err != nil {
		api.logger.Error("reload_charts failed", err)
		return err
	}

	req.Header.Set(
		"Authorization",
		auth.BasicAuthorizationHeaderVal(api.kiboshConfig.User, api.kiboshConfig.Pass),
	)
	res, err := client.Do(req)
	if err != nil {
		api.logger.Error("Couldn't call kibosh to update", err)
		return err
	}
	if res.StatusCode != 200 {
		err = errors.Errorf("kibosh return non 200 status code [%v]", res.StatusCode)
		api.logger.Error("Error triggering Kibosh reload", err)
		return err
	}
	return nil
}

func (api *api) ServerError(code int, message string, w http.ResponseWriter) {
	w.WriteHeader(code)
	w.Write([]byte(message))
}
