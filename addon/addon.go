package addon

import (
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
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
		return errors.Wrap(err, "Couldn't create addon zip file")
	}

	defer file.Close()

	_, err = io.Copy(file, a)

	if err != nil {
		return errors.Wrap(err, "Couldn't write to addon zip file")
	}

	return nil
}

func Download(url string) (*Addon, error) {
	resp, err := http.Get(url + "/file")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("Status code was not 200 at download URL")
	}

	return &Addon{resp}, nil
}
