package dea

import (
	"fmt"
	"os"
	"syscall"
)

type PidFile struct {
	file *os.File
}

func NewPidFile(pidFilename string) (*PidFile, error) {
	file, err := os.OpenFile(pidFilename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(file, "%d", os.Getpid())
	file.Sync()

	return &PidFile{file: file}, nil
}

func (pidFile *PidFile) Release() {
	syscall.Flock(int(pidFile.file.Fd()), syscall.LOCK_UN)
	os.Remove(pidFile.file.Name())
}
