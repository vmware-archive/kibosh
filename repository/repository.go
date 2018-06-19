package repository

import (
	"code.cloudfoundry.org/lager"
	"github.com/cf-platform-eng/kibosh/helm"
)

type Repository interface {
	LoadCharts() ([]*helm.MyChart, error)
}

type repository struct {
	helmChartDir          string
	privateRegistryServer string
	logger                lager.Logger
}

func NewRepository(chartPath string, privateRegistryServer string, logger lager.Logger) Repository {
	return &repository{
		helmChartDir:          chartPath,
		privateRegistryServer: privateRegistryServer,
		logger:                logger,
	}
}

func (r *repository) LoadCharts() ([]*helm.MyChart, error) {
	//todo: check chart dir to see if 1 or N
	//todo: if 1, load dir. If many, loop
	//todo: hard-coding n=1 for first commit
	charts := []*helm.MyChart{}
	myChart, err := helm.NewChart(r.helmChartDir, r.privateRegistryServer)
	if err != nil {
		return charts, err
	}
	charts = append(charts, myChart)

	return charts, nil
}
