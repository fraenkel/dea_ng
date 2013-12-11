package utils

import (
	"errors"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"launchpad.net/goyaml"
	"time"
)

var utilsLogger = Logger("Utils", nil)

func Repeat(delay time.Duration, what func()) *time.Timer {
	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		what()
		timer.Reset(delay)
	})

	return timer
}

func RepeatFixed(delay time.Duration, what func()) *time.Ticker {
	ticker := time.NewTicker(delay)

	go func() {
		for {
			select {
			case <-ticker.C:
				what()
			}
		}
	}()

	return ticker
}

func Timeout(delay time.Duration, what func() error) error {
	ch := make(chan error, 1)
	go func() {
		ch <- what()
	}()

	var err error
	timeout := time.After(delay)
	select {
	case <-timeout:
		err = errors.New("Timed out")
	case err = <-ch:
	}

	return err
}

func UUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return ""
	}
	return UUIDString(u)
}

func UUIDString(u *uuid.UUID) string {
	return fmt.Sprintf("%x%x%x%x%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func Yaml_Load(path string, result interface{}) error {
	if File_Exists(path) {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			utilsLogger.Warnf("Failed to read %s: %s", path, err.Error())
			return err
		}

		err = goyaml.Unmarshal(bytes, result)
		if err != nil {
			utilsLogger.Warnf("Failed to unmarshal %s: %s", path, err.Error())
			return err
		}
	}

	return nil
}
