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

package httphelpers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

func CreateFormRequest(url string, fieldname string, filepaths []string) (*http.Request, error) {

	body, boundary, err := CreateFormFile(fieldname, filepaths)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", boundary)

	return req, nil

}

func CreateFormFile(fieldname string, paths []string) (io.Reader, string, error) {

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, path := range paths {

		chartFileInfo, err := os.Stat(path)
		if err != nil {
			return nil, "", err
		}

		part, err := writer.CreateFormFile(fieldname, chartFileInfo.Name())
		if err != nil {
			return nil, "", err
		}

		file, err := os.Open(path)
		if err != nil {
			return nil, "", err
		}

		_, err = io.CopyBuffer(part, file, make([]byte, 4096))
		if err != nil {
			return nil, "", err
		}

	}

	//boundary := writer.Boundary()
	//_, err := io.Copy(part, strings.NewReader(fmt.Sprintf("\r\n--%s--\r\n", boundary)))
	writer.Close()

	return body, writer.FormDataContentType(), nil
}
