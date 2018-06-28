package httphelpers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

func CreateFormRequest(url string, fieldname string, filepath string) (*http.Request, error) {
	body, boundary, err := CreateFormFile(fieldname, filepath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "multipart/form-data; boundary="+boundary)

	return req, nil

}

func CreateFormFile(fieldname string, path string) (io.Reader, string, error) {
	chartFileInfo, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
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
	boundary := writer.Boundary()
	_, err = io.Copy(part, strings.NewReader(fmt.Sprintf("\r\n--%s--\r\n", boundary)))
	if err != nil {
		return nil, "", err
	}

	return body, boundary, nil
}
