package utils_test

import (
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"
)

var _ = Describe("Upload", func() {
	var logger = Logger("upload_test", nil)

	//func HttpUpload(fieldName, srcFile, uri string, logger *steno.Logger) error {
	It("uploads  a file", func() {
		srcBytes := []byte("copy a file")
		src, _ := ioutil.TempFile("", "upload")
		defer os.RemoveAll(src.Name())
		src.Write(srcBytes)
		src.Close()

		uploadResponse := func(w http.ResponseWriter, r *http.Request) {
			file, _, err := r.FormFile("droplet")
			Expect(err).To(BeNil())
			upBytes := make([]byte, 256)
			n, _ := file.Read(upBytes)
			Expect(upBytes[:n]).To(Equal(srcBytes))
			w.WriteHeader(http.StatusOK)
		}

		server := httptest.NewServer(http.HandlerFunc(uploadResponse))
		defer server.Close()

		err := HttpUpload("droplet", src.Name(), server.URL, logger)
		Expect(err).To(BeNil())
	})

	It("fails when the source file isn't found", func() {
		err := HttpUpload("droplet", "bogus", "http://1.2.3.4", logger)
		Expect(err).NotTo(BeNil())
		_, ok := err.(*os.PathError)
		Expect(ok).To(BeTrue())
	})

	It("connect timeout with an invalid ip address", func() {
		srcBytes := []byte("copy a file")
		src, _ := ioutil.TempFile("", "upload")
		defer os.RemoveAll(src.Name())
		src.Write(srcBytes)
		src.Close()

		oldTimeouts := GetHttpTimeouts()
		SetHttpTimeouts(HttpTimeouts{1 * time.Second, 1 * time.Second})
		defer SetHttpTimeouts(oldTimeouts)

		err := HttpUpload("droplet", src.Name(), "http://1.2.3.4", logger)
		Expect(err).NotTo(BeNil())
		netErr := err.(net.Error)
		Expect(netErr.Timeout()).To(BeTrue())
	})

	It("read timeout when the server is unresponsive", func() {
		srcBytes := []byte("copy a file")
		src, _ := ioutil.TempFile("", "upload")
		defer os.RemoveAll(src.Name())
		src.Write(srcBytes)
		src.Close()

		blocker := sync.NewCond(&sync.Mutex{})

		longResponse := func(w http.ResponseWriter, r *http.Request) {
			blocker.L.Lock()
			blocker.Wait()
			blocker.L.Unlock()
		}

		server := httptest.NewServer(http.HandlerFunc(longResponse))
		defer server.Close()

		oldTimeouts := GetHttpTimeouts()
		SetHttpTimeouts(HttpTimeouts{1 * time.Second, 1 * time.Second})
		defer SetHttpTimeouts(oldTimeouts)

		err := HttpUpload("droplet", src.Name(), server.URL, logger)
		blocker.Signal()
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("timeout awaiting response headers"))
	})

	It("fails when the server rejects the upload", func() {
		srcBytes := []byte("copy a file")
		src, _ := ioutil.TempFile("", "upload")
		defer os.RemoveAll(src.Name())
		src.Write(srcBytes)
		src.Close()

		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		err := HttpUpload("droplet", src.Name(), server.URL, logger)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("404 Not Found"))
	})

})
