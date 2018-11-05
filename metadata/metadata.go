package metadata

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
)

type AddonMetaData struct {
	Id       int               `json:"id"`
	Game     string            `json:"game"`
	Type     string            `json:"type"`
	Download *DownloadMetadata `json:"download"`
}

type DownloadMetadata struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
}

func (metadata *AddonMetaData) Validate() error {
	if metadata.Download == nil {
		return errors.New("The download URL is missing from the returned metadata")
	}
	if metadata.Game != "wow" {
		return errors.New("The addon is not associated with WoW")
	}
	return nil
}

func Fetch(addonName string) (*http.Response, error) {
	resp, err := http.Get("https://api.cfwidget.com/wow/addons/" + addonName)

	if err != nil {
		return nil, errors.Wrap(err, "Failed to download project metadata for "+addonName)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 202 {
			return nil, errors.New("Failed to download project metadata for " + addonName + " because it was not cached, but a fetch has been queued. Retry later.")
		}
		return nil, errors.New("Failed to download project metadata for " + addonName + ", the status code was " + strconv.Itoa(resp.StatusCode))
	}

	return resp, nil
}

func Decode(r io.Reader) AddonMetaData {
	decoder := json.NewDecoder(r)

	var metadata AddonMetaData
	decoder.Decode(&metadata)

	return metadata
}
