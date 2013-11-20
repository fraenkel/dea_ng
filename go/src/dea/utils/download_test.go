package utils_test

import (
	"crypto/sha1"
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
)

type DownloadResponder struct {
	response []byte
}

func (br DownloadResponder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write(br.response)
}

var _ = Describe("Download", func() {
	var (
		sha     = []byte("DEADBEEF")
		to_file *os.File
		logger  = Logger("download_test", nil)
	)

	BeforeEach(func() {
		var err error
		to_file, err = ioutil.TempFile("", "some_dest")
		Expect(err).To(BeNil())

	})

	AfterEach(func() {
		os.RemoveAll(to_file.Name())
	})

	It("should download the file if the sha1 matches", func() {
		body := []byte("The Body")
		server := httptest.NewServer(DownloadResponder{body})
		defer server.Close()

		shaDigest := sha1.New()
		shaDigest.Write(body)

		err := HttpDownload(server.URL+"/droplet", to_file, shaDigest.Sum(nil), logger)
		Expect(err).To(BeNil())

		actualBody, err := ioutil.ReadFile(to_file.Name())
		Expect(err).To(BeNil())
		Expect(body).To(Equal(actualBody))
	})

	Context("when the sha is not given", func() {
		It("does not verify the sha1", func() {
			body := []byte("The Body")
			server := httptest.NewServer(DownloadResponder{body})
			defer server.Close()

			err := HttpDownload(server.URL+"/droplet", to_file, nil, logger)
			Expect(err).To(BeNil())

			actualBody, err := ioutil.ReadFile(to_file.Name())
			Expect(err).To(BeNil())
			Expect(body).To(Equal(actualBody))
		})
	})

	It("fails when the file isn't found", func() {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		err := HttpDownload(server.URL+"/droplet", to_file, sha, logger)
		Expect(err.Error()).To(ContainSubstring("status: 404"))
	})

	It("should fail when response payload has invalid SHA1", func() {
		server := httptest.NewServer(DownloadResponder{[]byte("fooz")})
		defer server.Close()

		err := HttpDownload(server.URL+"/droplet", to_file, sha, logger)
		Expect(err.Error()).To(ContainSubstring("SHA1 mismatch"))
	})

	It("should fail when the destination is invalid", func() {
		server := httptest.NewServer(DownloadResponder{[]byte("fooz")})
		defer server.Close()

		to_file.Close()
		err := HttpDownload(server.URL+"/droplet", to_file, sha, logger)
		Expect(err.Error()).To(ContainSubstring("bad file descriptor"))
	})

	It("should time out if the server is down", func() {
		oldTimeouts := GetHttpTimeouts()
		SetHttpTimeouts(HttpTimeouts{1 * time.Second, 1 * time.Second})
		defer SetHttpTimeouts(oldTimeouts)

		err := HttpDownload("http://1.2.3.4", to_file, sha, logger)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("timeout"))
	})
})
