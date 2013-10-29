package utils

import (
	"errors"
	"fmt"
	steno "github.com/cloudfoundry/gosteno"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

func HttpUpload(fieldName, srcFile, uri string, logger *steno.Logger) error {
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
		if urlErr, ok := err.(*url.Error); ok {
			return urlErr.Err
		}

		bodyErr := <-errChan
		if bodyErr != nil {
			return bodyErr
		}
		
		return err
	}

	if rsp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("HTTP status (%s) : %d - %s", uri, rsp.StatusCode, rsp.Status))
		logger.Warnd(map[string]interface{}{"destination": uri, "error": err},
			"upload.failed")

		return err
	}

	logger.Warnd(map[string]interface{}{"destination": uri},
		"upload.completion")

	return nil
}
