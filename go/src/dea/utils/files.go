package utils

import (
	"crypto/sha1"
	"io"
	"os"
)

func SHA1Digest(src string) ([]byte, error) {
	shaDigest := sha1.New()
	file, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = io.Copy(shaDigest, file)
	return shaDigest.Sum(nil), nil
}

func File_Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CopyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
