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
	"path"
	"strings"

	"os"
	"path/filepath"
	"regexp"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
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
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Bullets     []string `yaml:"bullets"`
	File        string `yaml:"file"`
	Values      []byte
}

func NewChart(chartPath string, privateRegistryServer string, requirePlans bool) (*MyChart, error) {
	myChart := &MyChart{
		Chartpath:             chartPath,
		privateRegistryServer: privateRegistryServer,
	}
	err := myChart.EnsureIgnore()
	if err != nil {
		return nil, errors.Wrap(err, "Error fixing .helmignore")
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
	raw, err := c.ReadDefaultVals(c.Chartpath)
	if err != nil {
		return err
	}

	baseVals := map[string]interface{}{}
	err = yaml.Unmarshal(raw, &baseVals)
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
			transformedImagesVals := map[string]interface{}{}
			imageMap, ok := val.(map[string]interface{})
			if !ok {
				return nil, errors.New("'images' key value isn't correctly structured, see dos")
			}
			for imageName, imageDef := range imageMap {
				imageDefMap, ok := imageDef.(map[string]interface{})
				if !ok {
					return nil, errors.New("'images' key value isn't correctly structured, see dos")
				}
				transformedImage, err := c.OverrideImageSources(imageDefMap)
				if err != nil {
					return nil, err
				}
				transformedImagesVals[imageName] = transformedImage
			}
			transformedVals["images"] = transformedImagesVals
		} else {
			transformedVals[key] = val
		}
	}
	return transformedVals, nil
}

func (c *MyChart) loadPlans() error {
	plansPath := path.Join(c.Chartpath, "plans.yaml")
	bytes, err := ioutil.ReadFile(plansPath)
	if err != nil {
		return err
	}

	plans := []Plan{}
	err = yaml.Unmarshal(bytes, &plans)
	if err != nil {
		return err
	}

	c.Plans = map[string]Plan{}
	for _, p := range plans {

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
