package test

import (
	"io/ioutil"
	"path/filepath"
	"os"
)

type TestChart struct {
	ChartYaml    []byte
	ValuesYaml   []byte
	PlansYaml    []byte
	PlanContents map[string][]byte
}

func DefaultChart() *TestChart {
	chartYaml := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	valuesYaml := []byte(`
name: value
`)
	plansYaml := []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yaml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yaml"
`)

	smallYaml := []byte(``)
	mediumYaml := []byte(`
persistence:
  size: 16Gi
`)
	planContents := map[string][]byte{
		"small":  smallYaml,
		"medium": mediumYaml,
	}

	return &TestChart{
		ChartYaml:    chartYaml,
		ValuesYaml:   valuesYaml,
		PlansYaml:    plansYaml,
		PlanContents: planContents,
	}
}

func (t *TestChart) WriteChart(chartPath string) error {
	plansPath := filepath.Join(chartPath, "plans")
	_, plansPathExists := os.Stat(plansPath)
	if os.IsNotExist(plansPathExists) {
		err := os.Mkdir(plansPath, 0700)
		if err != nil {
			return err
		}
	}

	err := ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), t.ChartYaml, 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), t.ValuesYaml, 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), t.PlansYaml, 0666)
	if err != nil {
		return err
	}
	for key, value := range t.PlanContents {
		path := filepath.Join(chartPath, "plans", key + ".yaml")
		err = ioutil.WriteFile(path, value, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}
