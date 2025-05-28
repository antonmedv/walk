//go:build plan9
// +build plan9

package clipboard

import (
	"io/ioutil"
	"os"
)

func readAll() (string, error) {
	f, err := os.Open("/dev/snarf")
	if err != nil {
		return "", err
	}
	defer f.Close()

	str, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(str), nil
}

func writeAll(text string) error {
	f, err := os.OpenFile("/dev/snarf", os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(text))
	if err != nil {
		return err
	}

	return nil
}
