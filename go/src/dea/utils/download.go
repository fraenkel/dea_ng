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

/*
class Download
  attr_reader :source_uri, :destination_file, :sha1_expected
  attr_reader :logger

  class DownloadError < StandardError
    attr_reader :data

    def initialize(msg, data = {})
      @data = data

      super("Error downloading: %s (%s)" % [uri, msg])
    end

    def uri
      data[:droplet_uri] || "(unknown)"
    end
  end

  def initialize(source_uri, destination_file, sha1_expected=nil, custom_logger=nil)
    @source_uri = source_uri
    @destination_file = destination_file
    @sha1_expected = sha1_expected
    @logger = custom_logger || self.class.logger
  end

  def download!(&blk)
    destination_file.binmode
    sha1 = Digest::SHA1.new

    http = EM::HttpRequest.new(source_uri).get

    http.stream do |chunk|
      destination_file << chunk
      sha1 << chunk
    end

    context = { :droplet_uri => source_uri }

    http.errback do
      error = DownloadError.new("Response status: unknown", context)
      logger.warn(error.message, error.data)
      blk.call(error)
    end

    http.callback do
      destination_file.close
      http_status = http.response_header.status

      context[:droplet_http_status] = http_status

      if http_status == 200
        sha1_actual = sha1.hexdigest
        if !sha1_expected || sha1_expected == sha1_actual
          logger.info("Download succeeded")
          blk.call(nil)
        else
          context[:droplet_sha1_expected] = sha1_expected
          context[:droplet_sha1_actual] = sha1_actual

          error = DownloadError.new("SHA1 mismatch", context)
          logger.warn(error.message, error.data)
          blk.call(error)
        end
      else
        error = DownloadError.new("HTTP status: #{http_status}", context)
        logger.warn(error.message, error.data)
        blk.call(error)
      end
    end
  end
end
*/
