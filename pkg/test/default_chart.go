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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"

	"github.com/cf-platform-eng/kibosh/pkg/helm"
)

type TestChart struct {
	ChartYaml    []byte
	ValuesYaml   []byte
	PlansYaml    []byte
	PlanContents map[string][]byte
	Templates    map[string][]byte
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

	templates := map[string][]byte{
		"loadbalancer.yaml": []byte(`
apiVersion: v1
kind: Service
metadata:
  name: my-service-lb
spec:
  selector:
    role: foo
  type: LoadBalancer
  ports:
    - port: 1234
      targetPort: 1234
      protocol: TCP
`),
	}

	return &TestChart{
		ChartYaml:    chartYaml,
		ValuesYaml:   valuesYaml,
		PlansYaml:    plansYaml,
		PlanContents: planContents,
		Templates:    templates,
		HasPlans:     true,
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

func (t *TestChart) WriteChart(chartPath string) error {
	subdirs := []string{"plans", "templates"}
	for _, subdir := range subdirs {
		subdirPath := filepath.Join(chartPath, subdir)
		_, subdirPathExists := os.Stat(subdirPath)
		if os.IsNotExist(subdirPathExists) {
			err := os.Mkdir(subdirPath, 0700)
			if err != nil {
				return err
			}
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

	if t.HasPlans {
		err = ioutil.WriteFile(filepath.Join(chartPath, "plans.yaml"), t.PlansYaml, 0666)
		if err != nil {
			return err
		}
		for key, value := range t.PlanContents {
			path := filepath.Join(chartPath, "plans", key+".yaml")
			err = ioutil.WriteFile(path, value, 0666)
			if err != nil {
				return err
			}
		}
	}

	for key, value := range t.Templates {
		path := filepath.Join(chartPath, "templates", key)
		err = ioutil.WriteFile(path, value, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TestChart) WriteChartPackage(log *logrus.Logger) (string, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	err = t.WriteChart(tmpDir)
	if err != nil {
		return "", err
	}

	myChart, err := helm.NewChart(tmpDir, "", log)
	if err != nil {
		return "", err
	}

	outDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	chartTarball, err := chartutil.Save(&myChart.Chart, outDir)
	if err != nil {
		return "", err
	}

	return chartTarball, err
}

func (t *TestChart) WriteChartYML(chartPath string) error {
	plansPath := filepath.Join(chartPath, "plans")
	_, plansPathExists := os.Stat(plansPath)
	if os.IsNotExist(plansPathExists) {
		err := os.Mkdir(plansPath, 0700)
		if err != nil {
			return err
		}
	}

	t.PlansYaml = []byte(`
- name: "small"
  description: "default (small) plan for mysql"
  file: "small.yml"
- name: "medium"
  description: "medium sized plan for mysql"
  file: "medium.yml"
  free: false
  credentials: "medium-creds.yml"
`)

	err := ioutil.WriteFile(filepath.Join(chartPath, "Chart.yaml"), t.ChartYaml, 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(chartPath, "values.yaml"), t.ValuesYaml, 0666)
	if err != nil {
		return err
	}

	if t.HasPlans {
		err = ioutil.WriteFile(filepath.Join(chartPath, "plans.yml"), t.PlansYaml, 0666)
		if err != nil {
			return err
		}
		for key, value := range t.PlanContents {
			path := filepath.Join(chartPath, "plans", key+".yml")
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

	return helm.NewChart(chartPath, "docker.example.com", nil)
}
