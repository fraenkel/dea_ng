package utils

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Download struct {
	uri         string
	destination *os.File
	sha1        string
}

func NewDownload(uri string, file *os.File, sha1 string) Download {
	return Download{uri, file, sha1}
}

func (d Download) Download() error {
	defer d.destination.Close()

	resp, err := http.Get(d.uri)
	defer resp.Body.Close()

	shaDigest := sha1.New()
	_, err = io.Copy(io.MultiWriter(shaDigest, d.destination), resp.Body)

	logger := Logger("Download")
	if err != nil {
		logger.Warnf("Error downloading: %s (Response status: unknown)", d.uri)
		return err
	}

	context := make(map[string]interface{})
	context["droplet_uri"] = d.uri

	d.destination.Close()
	http_status := resp.StatusCode
	context["droplet_http_status"] = http_status

	if http_status == http.StatusOK {
		sha1_actual := fmt.Sprintf("%x", shaDigest.Sum(nil))
		if d.sha1 == "" || d.sha1 == sha1_actual {
			logger.Info("Download succeeded")
			return nil
		}

		context["droplet_sha1_expected"] = d.sha1
		context["droplet_sha1_actual"] = sha1_actual

		errMsg := fmt.Sprintf("Error downloading: %s (SHA1 mismatch)", d.uri)
		logger.Warnf("%s %v", errMsg, context)
		return errors.New(errMsg)
	} else {
		errMsg := fmt.Sprintf("Error downloading: %s (HTTP status: %d)", d.uri, http_status)
		logger.Warnf("%s %v", errMsg, context)
		return errors.New(errMsg)
	}
}