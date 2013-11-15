package dea_test

import (
	. "dea"
	"dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

var _ = Describe("PidFile", func() {
	var pidFilename string
	var pidFile *PidFile

	BeforeEach(func() {
		pidFilename = createTempFile()
		pidFile, _ = NewPidFile(pidFilename)
	})

	AfterEach(func() {
		pidFile.Release()
	})

	Context("NewPidFile", func() {

		It("creates a new pid file", func() {
			Expect(utils.File_Exists(pidFilename)).To(BeTrue())
		})

		It("contains a pid", func() {
			f, _ := os.Open(pidFilename)
			bytes, _ := ioutil.ReadAll(f)
			pid, _ := strconv.Atoi(string(bytes))
			Expect(os.Getpid()).To(Equal(pid))
		})

		It("fails when already created", func() {
			pidFile2, err := NewPidFile(pidFilename)
			defer func() {
				if pidFile2 != nil {
					pidFile2.Release()
				}
			}()

			Expect(err).To(Equal(syscall.EWOULDBLOCK))
		})
	})

	Context("Release", func() {
		It("removes the pid file", func() {
			pidFile.Release()
			Expect(utils.File_Exists(pidFilename)).To(BeFalse())
		})
		It("doesn't fail when called twice", func() {
			pidFile.Release()
			pidFile.Release()
		})
	})

})

func createTempFile() string {
	file, err := ioutil.TempFile("", "pidfile")
	if err != nil {
		panic(err)
	}
	os.Remove(file.Name())
	return file.Name()
}
