package utils

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func HttpDownload(uri string, destination *os.File, sha1_expected string) error {
	defer destination.Close()

	resp, err := http.Get(uri)
	defer resp.Body.Close()

	shaDigest := sha1.New()
	_, err = io.Copy(io.MultiWriter(shaDigest, destination), resp.Body)

	logger := Logger("Download")
	if err != nil {
		logger.Warnf("Error downloading: %s (Response status: unknown)", uri)
		return err
	}

	context := make(map[string]interface{})
	context["droplet_uri"] = uri

	destination.Close()
	http_status := resp.StatusCode
	context["droplet_http_status"] = http_status

	if http_status == http.StatusOK {
		sha1_actual := fmt.Sprintf("%x", shaDigest.Sum(nil))
		if sha1_expected == "" || sha1_expected == sha1_actual {
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
