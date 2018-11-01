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
	"github.com/cf-platform-eng/kibosh/pkg/moreio"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/cf-platform-eng/kibosh/pkg/docker"
)

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
	if !moreio.DirExistsAndIsReadable(chartPath) {
		return errors.New(fmt.Sprintf("chart path [%s] is not a directory on disk", chartPath))
	}
	registry := argsWithoutProgramName[1]

	imagesPath := path.Join(chartPath, "images")
	if !moreio.DirExistsAndIsReadable(imagesPath) {
		return errors.New(fmt.Sprintf("image chart subpath [%s] is not a directory on disk", imagesPath))
	}

	_, err := docker.ParseValues(chartPath)
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
		err := docker.LoadImage(path.Join(imagesPath, file.Name()))
		if err != nil {
			return err
		}
	}

	err = docker.TagAndPush(registry)
	if err != nil {
		return err
	}

	return nil
}
