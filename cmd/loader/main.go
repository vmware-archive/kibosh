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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

type ImageValues struct {
	Image    string                 `yaml:"image"`
	ImageTag string                 `yaml:"imageTag"`
	Images   map[string]ImageValues `yaml:"images"`
}

type DockerImageOutput struct {
	Repo string `yaml:"repo"`
	Tag  string `yaml:"tag"`
}

func main() {
	err := run()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	argsWithoutProgramName := os.Args[1:]
	if len(argsWithoutProgramName) != 2 {
		return errors.New("single arg expected the path to parse")
	}
	chartPath := argsWithoutProgramName[0]
	if !DirExistsAndIsReadable(chartPath) {
		return errors.New(fmt.Sprintf("chart path [%s] is not a directory on disk", chartPath))
	}
	registry := argsWithoutProgramName[1]

	imagesPath := path.Join(chartPath, "images")
	if !DirExistsAndIsReadable(imagesPath) {
		return errors.New(fmt.Sprintf("image chart subpath [%s] is not a directory on disk", imagesPath))
	}

	_, err := ParseValues(chartPath)
	if err != nil {
		return errors.New(fmt.Sprintf("Error parsing values file %s", err.Error()))
	}

	files, err := ioutil.ReadDir(imagesPath)
	if err != nil {
		return errors.New(fmt.Sprintf("error reading files in images subpath: %s", err.Error()))
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}
		err := LoadImage(path.Join(imagesPath, file.Name()))
		if err != nil {
			return err
		}
	}

	err = TagAndPush(registry)
	if err != nil {
		return err
	}

	return nil
}

func TagAndPush(privateRegistryServer string) error {
	cmd := exec.Command("docker", "images", "--format", "- repo: {{ json .Repository }}\n  tag: {{ json .Tag }}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	images := []DockerImageOutput{}
	err = yaml.Unmarshal([]byte(output), &images)
	if err != nil {
		return err
	}

	for _, image := range images {
		split := strings.Split(image.Repo, "/")
		imageName := fmt.Sprintf("%s/%s", privateRegistryServer, split[len(split)-1])

		origTag := fmt.Sprintf("%s:%s", image.Repo, image.Tag)
		newTag := fmt.Sprintf("%s:%s", imageName, image.Tag)

		cmd := exec.Command("docker", "tag", origTag, newTag)
		output, err := cmd.CombinedOutput()
		println(string(output))
		if err != nil {
			return err
		}

		cmd = exec.Command("docker", "push", newTag)
		output, err = cmd.CombinedOutput()
		println(string(output))
		if err != nil {
			return err
		}
	}

	return nil
}

func LoadImage(imagePath string) error {
	cmd := exec.Command("docker", "load", "-i", imagePath)
	output, err := cmd.CombinedOutput()
	println(string(output))
	return err
}

func ParseValues(chartPath string) (*ImageValues, error) {
	valuesPath := path.Join(chartPath, "values.yaml")
	bytes, err := ioutil.ReadFile(valuesPath)
	if err != nil {
		return nil, err
	}

	parsedImages := &ImageValues{}
	err = yaml.Unmarshal(bytes, parsedImages)
	if err != nil {
		return nil, err
	}
	if !parsedImages.ValidateImages() {
		return nil, err
	}

	return parsedImages, nil
}

func (i *ImageValues) ValidateImages() bool {
	if !i.validateImage() && len(i.Images) == 0 {
		return false

	}
	for _, image := range i.Images {
		if !image.validateImage() {
			return false
		}
	}
	return true
}

func (i *ImageValues) validateImage() bool {
	if i.Image == "" || i.ImageTag == "" {
		return false
	}
	return true
}

func DirExistsAndIsReadable(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	if !stat.IsDir() {
		return false
	}
	return true
}
