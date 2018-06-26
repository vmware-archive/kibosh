package bazaar

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/repository"
	"github.com/pkg/errors"
)

type API interface {
	ListCharts() http.Handler
	SaveChart() http.Handler
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

type displayChart struct {
	Name      string   `json:"name"`
	Plans     []string `json:"plans"`
	Chartpath string   `json:"chartpath"`
}

func (api *api) ListCharts() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		charts, err := api.repo.LoadCharts()
		if err != nil {
			api.logger.Error("Unable to load charts", err)
			api.ServerError(500, "Unable to load charts", w)
		} else {
			var displayCharts []displayChart
			for _, chart := range charts {
				var plans []string
				for _, plan := range chart.Plans {
					plans = append(plans, plan.Name)
				}
				displayCharts = append(displayCharts, displayChart{
					Name:      chart.Metadata.Name,
					Plans:     plans,
					Chartpath: chart.Chartpath,
				})
			}
			err = api.WriteJSONResponse(w, displayCharts)
			if err != nil {
				api.logger.Error("Error writing list response", err)
			}
		}
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

func (api *api) SaveChart() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			w.Header().Set("Allow", "POST")
			return
		}

		err := api.saveChartToRepository(r)
		if err != nil {
			api.ServerError(500, "Unable to save charts", w)
			return
		}

		err = api.triggerKiboshReload()
		if err != nil {
			//todo: retry? rollback? what's on disk now doesn't match Kibosh
			api.ServerError(500, "Chart persisted, but Kibosh reload failed", w)
			return
		}
		api.WriteJSONResponse(w, map[string]interface{}{"message": "Chart saved"})
	})
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
		api.logger.Error("SaveChart: reload_charts failed", err)
		return err
	}

	auth := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", api.kiboshConfig.User, api.kiboshConfig.Pass)),
	)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	res, err := client.Do(req)
	if err != nil {
		api.logger.Error("SaveChart: Couldn't call kibosh to update", err)
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
