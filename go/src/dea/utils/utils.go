package utils

import (
	"errors"
	"fmt"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"launchpad.net/goyaml"
	"time"
)

func Repeat(what func(), delay time.Duration) *time.Ticker {
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

func Timeout(what func() error, delay time.Duration) error {
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

func Intersection(a []string, b []string) []string {
	max := len(a)
	big, small := a, b
	if len(b) > len(a) {
		max = len(b)
		big, small = b, a
	}

	intersection := make([]string, 0, max)
	// loop over smaller set
	for _, elem := range big {
		if Contains(small, elem) {
			intersection = append(intersection, elem)
		}
	}
	return intersection
}

func Contains(a []string, e string) bool {
	for _, elem := range a {
		if e == elem {
			return true
		}
	}
	return false
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
			Logger("Utils").Warnf("Failed to read %s: %s", path, err.Error())
			return err
		}

		err = goyaml.Unmarshal(bytes, result)
		if err != nil {
			Logger("Utils").Warnf("Failed to unmarshal %s: %s", path, err.Error())
			return err
		}
	}

	return nil
}
