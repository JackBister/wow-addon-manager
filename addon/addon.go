package addon

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

type Addon struct {
	resp *http.Response
}

func (a *Addon) Close() error {
	return a.resp.Body.Close()
}

func (a *Addon) Read(p []byte) (int, error) {
	return a.resp.Body.Read(p)
}

func (a *Addon) ToFile(fileName string) error {
	file, err := os.Create(fileName)

	if err != nil {
		return fmt.Errorf("couldn't create addon zip file: %w", err)
	}

	defer file.Close()

	_, err = io.Copy(file, a)

	if err != nil {
		return fmt.Errorf("couldn't write to addon zip file: %w", err)
	}

	return nil
}

func Download(url string) (*Addon, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		returnedError := "Status code was not 200 at download URL " + url + ", status code was " + strconv.Itoa(resp.StatusCode)
		bb, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			returnedError += ", body was: " + string(bb)
		}
		return nil, errors.New(returnedError)
	}

	return &Addon{resp}, nil
}
