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

package helm

import (
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"path"
	"strings"

	"os"
	"path/filepath"
	"regexp"

	"github.com/Sirupsen/logrus"
	"github.com/go-yaml/yaml"
	"github.com/pkg/errors"
	k8sAPI "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type MyChart struct {
	*chart.Chart

	Chartpath             string
	privateRegistryServer string
	Values                []byte
	Plans                 map[string]Plan
}

type Plan struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Bullets         []string `yaml:"bullets"`
	File            string   `yaml:"file"`
	Free            *bool    `yaml:"free,omitempty"`
	Bindable        *bool    `yaml:"bindable,omitempty"`
	CredentialsPath string   `yaml:"credentials"`

	ClusterConfig *k8sAPI.Config
	Values        []byte
}

func LoadFromDir(dir string, log *logrus.Logger) ([]*MyChart, error) {
	sourceDirStat, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !sourceDirStat.IsDir() {
		return nil, errors.New(fmt.Sprintf("The provided path [%s] is not a directory", dir))
	}
	sources, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	charts := []*MyChart{}
	for _, source := range sources {
		chartPath := path.Join(dir, source.Name())
		c, err := NewChart(chartPath, "", false)
		if err != nil {
			log.Debug(fmt.Sprintf("The file [%s] not failed to load as a chart", chartPath), err)
		} else {
			charts = append(charts, c)
		}
	}

	return charts, nil
}

func NewChart(chartPath string, privateRegistryServer string, requirePlans bool) (*MyChart, error) {
	myChart := &MyChart{
		Chartpath:             chartPath,
		privateRegistryServer: privateRegistryServer,
	}

	chartPathStat, err := os.Stat(chartPath)
	if err != nil {
		return nil, err
	}

	if chartPathStat.IsDir() {
		err = myChart.EnsureIgnore()
		if err != nil {
			return nil, errors.Wrap(err, "Error fixing .helmignore")
		}
	}

	loadedChart, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}
	myChart.Chart = loadedChart

	err = myChart.LoadChartValues()
	if err != nil {
		return nil, err
	}

	if requirePlans {
		err = myChart.loadPlans()
		if err != nil {
			return nil, err
		}
	}

	return myChart, nil
}

func (c *MyChart) LoadChartValues() error {
	baseVals := map[string]interface{}{}
	if c.Chart.Values == nil {
		return errors.New("values.yaml is requires")
	}
	err := yaml.Unmarshal([]byte(c.Chart.Values.Raw), &baseVals)
	if err != nil {
		return err
	}

	transformed, err := c.OverrideImageSources(baseVals)
	if err != nil {
		return err
	}

	finalVals, err := yaml.Marshal(transformed)
	if err != nil {
		return err
	}

	c.Values = finalVals

	return nil
}

func (c *MyChart) ReadDefaultVals(chartPath string) ([]byte, error) {
	valuesPath := path.Join(chartPath, "values.yaml")
	bytes, err := ioutil.ReadFile(valuesPath)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (c *MyChart) OverrideImageSources(rawVals map[string]interface{}) (map[string]interface{}, error) {
	if c.privateRegistryServer == "" {
		return rawVals, nil
	}

	transformedVals := map[string]interface{}{}
	for key, val := range rawVals {
		if key == "image" {
			stringVal, ok := val.(string)
			if !ok {
				return nil, errors.New("'image' key value is not a string, vals structure is incorrect")
			}
			split := strings.Split(stringVal, "/")
			transformedVals[key] = fmt.Sprintf("%s/%s", c.privateRegistryServer, split[len(split)-1])
		} else if key == "images" {
			remarshalled, err := yaml.Marshal(val)
			if err != nil {
				return nil, err
			}

			imageMap := map[string]map[string]interface{}{}
			err = yaml.Unmarshal(remarshalled, imageMap)
			if err != nil {
				return nil, err
			}

			for imageName, imageDefMap := range imageMap {
				transformedImage, err := c.OverrideImageSources(imageDefMap)
				if err != nil {
					return nil, err
				}
				imageMap[imageName] = transformedImage
			}
			transformedVals["images"] = imageMap
		} else {
			transformedVals[key] = val
		}
	}
	return transformedVals, nil
}

func (c *MyChart) loadPlans() error {
	plansPath := path.Join(c.Chartpath, "plans.yaml")
	plansBytes, err := ioutil.ReadFile(plansPath)
	if err != nil {
		return err
	}

	plans := []Plan{}
	err = yaml.Unmarshal(plansBytes, &plans)
	if err != nil {
		return err
	}

	c.Plans = map[string]Plan{}
	for _, p := range plans {
		if p.Free == nil {
			t := true
			p.Free = &t
		}
		if p.Bindable == nil {
			t := true
			p.Bindable = &t
		}
		match, err := regexp.MatchString(`^[0-9a-z.\-]+$`, p.Name)
		if err != nil {
			return err
		}
		if !match {
			return errors.New(fmt.Sprintf("Name [%s] contains invalid characters", p.Name))
		}

		planValues, err := ioutil.ReadFile(filepath.Join(c.Chartpath, "plans", p.File))
		if err != nil {
			return err
		}
		p.Values = planValues

		if p.CredentialsPath != "" {
			loader := &clientcmd.ClientConfigLoadingRules{
				ExplicitPath: filepath.Join(c.Chartpath, "plans", p.CredentialsPath),
			}
			loadedConfig, err := loader.Load()
			if err != nil {
				return err
			}

			p.ClusterConfig = loadedConfig
		}

		c.Plans[p.Name] = p
	}

	return nil
}

func (c *MyChart) EnsureIgnore() error {
	_, err := os.Stat(c.Chartpath)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Error reading chart dir [%s]", c.Chartpath))
	}

	ignoreFilePath := filepath.Join(c.Chartpath, ".helmignore")
	_, err = os.Stat(ignoreFilePath)
	if err != nil {
		file, err := os.Create(ignoreFilePath)
		defer file.Close()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error creating .helmignore [%s]", c.Chartpath))
		} else {
			file.Write([]byte("images"))
		}
	} else {
		contents, err := ioutil.ReadFile(ignoreFilePath)
		lines := strings.Split(string(contents), "\n")
		for _, line := range lines {
			if line == "images" {
				return nil
			}
		}

		file, err := os.OpenFile(ignoreFilePath, os.O_APPEND|os.O_WRONLY, 0666)
		defer file.Close()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error opening .helmignore [%s]", c.Chartpath))
		} else {
			_, err = file.Write([]byte("\nimages\n"))
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("Error appending to .helmignore [%s]", c.Chartpath))
			}
		}
	}

	return nil
}

func (c *MyChart) String() string {
	return c.Metadata.Name
}
