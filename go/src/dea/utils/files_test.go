package utils_test

import (
	"crypto/sha1"
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
)

var _ = Describe("Files", func() {
	Describe("SHA1 Digest", func() {
		It("Computes Sha1 of a file", func() {
			bytes := []byte("some bytes")
			digest := sha1.New()
			digest.Write(bytes)
			expected := digest.Sum(nil)

			file, _ := ioutil.TempFile("", "sha1")
			defer os.RemoveAll(file.Name())
			file.Write(bytes)
			file.Close()

			actual, err := SHA1Digest(file.Name())
			Expect(err).To(BeNil())
			Expect(expected).To(Equal(actual))
		})

		It("Error when file is missing", func() {
			_, err := SHA1Digest("bogus")
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("File_Exists", func() {
		It("true when file exists", func() {
			file, _ := ioutil.TempFile("", "sha1")
			defer os.RemoveAll(file.Name())
			file.Close()

			b := File_Exists(file.Name())
			Expect(b).To(BeTrue())
		})

		It("false when file exists", func() {
			b := File_Exists("bogus")
			Expect(b).To(BeFalse())
		})

	})

	Describe("CopyFile", func() {
		It("copies a file", func() {
			srcBytes := []byte("copy a file")
			src, _ := ioutil.TempFile("", "copysrc")
			defer os.RemoveAll(src.Name())
			src.Write(srcBytes)
			src.Close()

			dst, _ := ioutil.TempFile("", "copydst")
			defer os.RemoveAll(dst.Name())
			dst.Close()

			err := CopyFile(src.Name(), dst.Name())
			Expect(err).To(BeNil())

			dstBytes, _ := ioutil.ReadFile(dst.Name())
			Expect(dstBytes).To(Equal(srcBytes))
		})

		It("fails when the source is invalid", func() {
			dst, _ := ioutil.TempFile("", "copydst")
			defer os.RemoveAll(dst.Name())
			dst.Close()

			err := CopyFile("/a/b/c/d", dst.Name())
			Expect(err).NotTo(BeNil())
		})

		It("fails when the destination is invalid", func() {
			srcBytes := []byte("copy a file")
			src, _ := ioutil.TempFile("", "copysrc")
			defer os.RemoveAll(src.Name())
			src.Write(srcBytes)
			src.Close()

			err := CopyFile(src.Name(), "/a/b/c/d")
			Expect(err).NotTo(BeNil())
		})
	})
})
