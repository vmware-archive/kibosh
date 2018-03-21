package helm

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type MyChart struct {
	*chart.Chart

	chartPath             string
	privateRegistryServer string
	Values                []byte
}

func NewChart(chartPath string, privateRegistryServer string) (*MyChart, error) {
	loadedChart, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	myChart := &MyChart{
		Chart:                 loadedChart,
		chartPath:             chartPath,
		privateRegistryServer: privateRegistryServer,
	}
	err = myChart.LoadChartValues()
	if err != nil {
		return nil, err
	}

	return myChart, nil
}

func (c *MyChart) LoadChartValues() error {
	raw, err := c.ReadDefaultVals(c.chartPath)
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
