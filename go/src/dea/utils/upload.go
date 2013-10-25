package utils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func newHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}
	return &http.Client{
		Transport: tr,
	}
}

func HttpUpload(fieldName, srcFile, uri string) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	bodyReader, bodyWriter := io.Pipe()
	multiWriter := multipart.NewWriter(bodyWriter)
	errChan := make(chan error, 1)
	go func() {
		defer bodyWriter.Close()
		defer file.Close()
		part, err := multiWriter.CreateFormFile(fieldName, filepath.Base(srcFile))
		if err != nil {
			errChan <- err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			errChan <- err
			return
		}
		errChan <- multiWriter.Close()
	}()

	req, err := http.NewRequest("POST", uri, bodyReader)
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	client := newHttpClient()
	rsp, err := client.Do(req)
	if err != nil {
		bodyErr := <-errChan
		if bodyErr != nil {
			return bodyErr
		}
		return err
	}

	if rsp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("HTTP status (%s) : %d - %s", uri, rsp.StatusCode, rsp.Status))
		Logger("Upload").Warnd(map[string]interface{}{"destination": uri, "error": err},
			"upload.failed")

		return err
	}

	Logger("Upload").Warnd(map[string]interface{}{"destination": uri},
		"upload.completion")

	return nil
}
