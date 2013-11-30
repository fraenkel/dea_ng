package droplet

import (
	"crypto/sha1"
	"dea/utils"
	"encoding/hex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
)

var _ = Describe("Droplet", func() {
	var tmpdir string
	var droplet Droplet
	var sha1Hash string
	payload := []byte("droplet")

	BeforeEach(func() {
		shaDigest := sha1.New()
		shaDigest.Write(payload)
		sha1Hash = hex.EncodeToString(shaDigest.Sum(nil))
		tmpdir, _ = ioutil.TempDir("", "droplet")
		droplet, _ = NewDroplet(tmpdir, sha1Hash)
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	It("should export its sha1", func() {
		Expect(hex.EncodeToString(droplet.SHA1())).To(Equal(sha1Hash))
	})

	It("should make sure its directory exists", func() {
		Expect(utils.File_Exists(droplet.Dir())).To(BeTrue())
	})

	Describe("destroy", func() {
		It("should remove the associated directory", func() {
			Expect(utils.File_Exists(droplet.Dir())).To(BeTrue())
			droplet.Destroy()
			Expect(utils.File_Exists(droplet.Dir())).To(BeFalse())
		})
	})

	Describe("download", func() {
		Context("when unsucessful", func() {
			It("should fail when server is unreachable", func() {
				err := droplet.Download("http://127.0.0.1:12346/droplet")
				Expect(err).NotTo(BeNil())
			})

			Context("when response has status other than 200", func() {
				It("should fail", func() {
					server := httptest.NewServer(http.NotFoundHandler())
					defer server.Close()
					err := droplet.Download(server.URL + "/droplet")
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("404 Not Found"))
				})

				It("should not create droplet file", func() {
					server := httptest.NewServer(http.NotFoundHandler())
					defer server.Close()
					droplet.Download(server.URL + "/droplet")
					Expect(utils.File_Exists(droplet.Path())).To(BeFalse())
				})

				It("should fail when response payload has invalid SHA1", func() {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Write([]byte("fooz"))
					}))
					defer server.Close()
					err := droplet.Download(server.URL + "/droplet")
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("SHA1 mismatch"))
				})

			})

		})

		Context("when sucessful", func() {
			var server *httptest.Server
			var download_count int

			BeforeEach(func() {
				download_count = 0
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					download_count++
					w.Write(payload)
				}))
			})

			AfterEach(func() {
				server.Close()
				os.Remove(droplet.Path())
			})

			It("should download without error", func() {
				err := droplet.Download(server.URL + "/droplet")
				Expect(err).To(BeNil())
				Expect(utils.File_Exists(droplet.Path())).To(BeTrue())
			})

			Context("when the same dea is running multiple instances of the app", func() {
				It("only downloads the droplet once", func() {
					for i := 0; i < 3; i++ {
						err := droplet.Download(server.URL + "/droplet")
						Expect(err).To(BeNil())
					}
					Expect(download_count).To(Equal(1))
				})
			})

			Context("when the droplet is already downloaded", func() {
				Context("and the sha matches", func() {
					BeforeEach(func() {
						ioutil.WriteFile(droplet.Path(), payload, 0755)
					})

					It("does not download the file", func() {
						err := droplet.Download(server.URL + "/droplet")
						Expect(err).To(BeNil())
						Expect(download_count).To(Equal(0))
					})
				})

				Context("and the sha does not matches", func() {
					BeforeEach(func() {
						ioutil.WriteFile(droplet.Path(), []byte("bogus"), 0755)
					})

					It("does not download the file", func() {
						err := droplet.Download(server.URL + "/droplet")
						Expect(err).To(BeNil())
						Expect(download_count).To(Equal(1))
					})
				})

			})
		})
	})

	Describe("local_copy", func() {
		var source_file string
		BeforeEach(func() {
			source_file = path.Join(tmpdir, "source_file")
		})

		Context("when copy was successful", func() {
			some_data := []byte("some_data")
			BeforeEach(func() {
				ioutil.WriteFile(source_file, some_data, 0755)
			})

			It("saves file in droplet path", func() {
				err := droplet.Local_copy(source_file)
				Expect(err).To(BeNil())
				Expect(utils.File_Exists(droplet.Path())).To(BeTrue())
				bytes, err := ioutil.ReadFile(source_file)
				Expect(err).To(BeNil())
				Expect(bytes).To(Equal(some_data))
			})
		})

		Context("when copy fails", func() {
			BeforeEach(func() {
				os.Remove(source_file)
			})

			It("saves file in droplet path", func() {
				err := droplet.Local_copy(source_file)
				Expect(err).NotTo(BeNil())
				Expect(utils.File_Exists(droplet.Path())).To(BeFalse())
			})
		})

	})

})
