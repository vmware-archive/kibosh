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

package test

import (
	"github.com/cf-platform-eng/kibosh/pkg/helm"
	"io/ioutil"
	"os"
	"path/filepath"
)

type TestChart struct {
	ChartYaml    []byte
	ValuesYaml   []byte
	PlansYaml    []byte
	PlanContents map[string][]byte
	HasPlans     bool
}

func DefaultChart() *TestChart {
	chartYaml := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	valuesYaml := []byte(`
name: value
count: 1
nested:
  inner1: inner 1 value
  inner2: inner2val
`)
	plansYaml := []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yaml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yaml"
  free: false
  credentials: "medium-creds.yaml"
`)

	smallYaml := []byte(``)
	mediumYaml := []byte(`
persistence:
  size: 16Gi
`)
	mediumCredsYaml := []byte(`
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: c29tZS1jZXI=
    server: https://pks-cluster.example.com
  name: my-cluster
contexts:
- context:
    cluster: my-cluster
  name: context
current-context: my-context
`)

	planContents := map[string][]byte{
		"small":        smallYaml,
		"medium":       mediumYaml,
		"medium-creds": mediumCredsYaml,
	}

	return &TestChart{
		ChartYaml:    chartYaml,
		ValuesYaml:   valuesYaml,
		HasPlans:     true,
		PlansYaml:    plansYaml,
		PlanContents: planContents,
	}
}

func PlainChart() *TestChart {
	chartYaml := []byte(`
name: spacebears
description: spacebears service and spacebears broker helm chart
version: 0.0.1
`)

	valuesYaml := []byte(`
name: value
`)

	return &TestChart{
		ChartYaml:  chartYaml,
		ValuesYaml: valuesYaml,
		HasPlans:   false,
	}
}

func (t *TestChart) WriteChart(chartPath string, useYml ...bool) error {
	plansPath := filepath.Join(chartPath, "plans")
	_, plansPathExists := os.Stat(plansPath)
	if os.IsNotExist(plansPathExists) {
		err := os.Mkdir(plansPath, 0700)
		if err != nil {
			return err
		}
	}

	var suffix string
	plansYml := []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yml"
  free: false
  credentials: "medium-creds.yml"
`)

	if len(useYml) > 0 && useYml[0] {
		suffix = ".yml"
		t.PlansYaml = plansYml
	} else {
		suffix = ".yaml"
	}

	err := ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), t.ChartYaml, 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), t.ValuesYaml, 0666)
	if err != nil {
		return err
	}

	if t.HasPlans {
		err = ioutil.WriteFile(filepath.Join(chartPath, "plans"+suffix), t.PlansYaml, 0666)
		if err != nil {
			return err
		}
		for key, value := range t.PlanContents {
			path := filepath.Join(chartPath, "plans", key+suffix)
			err = ioutil.WriteFile(path, value, 0666)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DefaultMyChart() (*helm.MyChart, error) {
	chartPath, err := ioutil.TempDir("", "chart-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(chartPath)

	testChart := DefaultChart()
	err = testChart.WriteChart(chartPath)
	if err != nil {
		return nil, err
	}

	return helm.NewChart(chartPath, "docker.example.com")
}
