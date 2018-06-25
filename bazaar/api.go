package bazaar

import (
	"code.cloudfoundry.org/lager"
	"encoding/json"
	"github.com/cf-platform-eng/kibosh/repository"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type API interface {
	ListCharts() http.Handler
	SaveChart() http.Handler
}

type api struct {
	repo   repository.Repository
	logger lager.Logger
}

func NewAPI(repo repository.Repository, l lager.Logger) API {
	return &api{
		repo:   repo,
		logger: l,
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

			displayCharts := []displayChart{}
			for _, chart := range charts {
				plans := []string{}
				for _, plan := range chart.Plans {
					plans = append(plans, plan.Name)
				}
				displayCharts = append(displayCharts, displayChart{
					Name:      chart.Metadata.Name,
					Plans:     plans,
					Chartpath: chart.Chartpath,
				})
			}
			serialized, _ := json.Marshal(displayCharts)

			header := w.Header()
			header.Set("Content-Type", "application/json")
			w.Write([]byte(serialized))
		}
	})

}

func (api *api) SaveChart() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseMultipartForm(1000000)
			file, handler, err := r.FormFile("chart")
			if err != nil {
				api.logger.Error("SaveChart: Couldn't read request POST form data", err)
				api.ServerError(500, "Unable to save charts", w)
				return
			}
			defer file.Close()
			chartPath, err := ioutil.TempDir("", "chart-")
			f, err := os.OpenFile(filepath.Join(chartPath, handler.Filename), os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				api.logger.Error("SaveChart: Couldn't write on disk ", err)
				api.ServerError(500, "Unable to save charts", w)
				return
			}
			defer f.Close()
			buffer := make([]byte, 1000000)
			io.CopyBuffer(f, file, buffer)

			err = api.repo.SaveChart(filepath.Join(chartPath, handler.Filename))
			if err != nil {
				api.logger.Error("SaveChart: Couldn't save the chart", err)
				api.ServerError(500, "Unable to save charts", w)
				return
			}
			//todo: call kibosh update charts
		} else {
			w.WriteHeader(405)
			w.Header().Set("Allow", "POST")
		}
	})
}

func (api *api) ServerError(code int, message string, w http.ResponseWriter) {
	w.WriteHeader(code)
	w.Write([]byte(message))
}
