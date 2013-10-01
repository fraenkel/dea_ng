package dea

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
	"testing"
)

func TestCreatePidFile(t *testing.T) {
	pidFilename := createTempFile(t)
	pidFile, err := NewPidFile(pidFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer pidFile.Release()

	assert.True(t, exists(t, pidFilename))
	f, err := os.Open(pidFilename)
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(bytes))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, pid, os.Getpid())
}

func TestReleasePidFile(t *testing.T) {
	pidFilename := createTempFile(t)
	pidFile, err := NewPidFile(pidFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer pidFile.Release()
	pidFile.Release()

	assert.False(t, exists(t, pidFilename))
}

func TestPidFileAlreadyLocked(t *testing.T) {
	pidFilename := createTempFile(t)
	pidFile, err := NewPidFile(pidFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer pidFile.Release()

	pidFile2, err2 := NewPidFile(pidFilename)
	defer func() {
		if pidFile2 != nil {
			pidFile2.Release()
		}
	}()

	assert.Equal(t, err2, syscall.EWOULDBLOCK)
}

func createTempFile(t *testing.T) string {
	file, err := ioutil.TempFile("", "pidfile")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(file.Name())
	return file.Name()
}

func exists(t *testing.T, filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
