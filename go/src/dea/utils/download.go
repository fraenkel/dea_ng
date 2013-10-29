package utils

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	steno "github.com/cloudfoundry/gosteno"
	"io"
	"net/http"
	"net/url"
	"os"
)

func HttpDownload(uri string, destination *os.File, sha1_expected []byte, logger *steno.Logger) error {
	defer destination.Close()

	client := newHttpClient()
	rsp, err := client.Get(uri)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			return urlErr.Err
		}

		logger.Warnf("Get error: %s %s", uri, err.Error())
		return err
	}
	defer rsp.Body.Close()

	shaDigest := sha1.New()
	_, err = io.Copy(io.MultiWriter(shaDigest, destination), rsp.Body)

	if err != nil {
		logger.Warnf("Error downloading: %s (Response status: unknown)", uri)
		return err
	}

	context := make(map[string]interface{})
	context["droplet_uri"] = uri

	http_status := rsp.StatusCode
	context["droplet_http_status"] = http_status

	if http_status == http.StatusOK {
		sha1_actual := shaDigest.Sum(nil)
		if sha1_expected == nil || bytes.Equal(sha1_expected, sha1_actual) {
			logger.Info("Download succeeded")
			return nil
		}

		context["droplet_sha1_expected"] = sha1_expected
		context["droplet_sha1_actual"] = sha1_actual

		errMsg := fmt.Sprintf("Error downloading: %s (SHA1 mismatch)", uri)
		logger.Warnf("%s %v", errMsg, context)
		return errors.New(errMsg)
	} else {
		errMsg := fmt.Sprintf("Error downloading: %s (HTTP status: %d)", uri, http_status)
		logger.Warnf("%s %v", errMsg, context)
		return errors.New(errMsg)
	}
}
