package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AddonMetaData struct {
	Id    int            `json:"id"`
	Title string         `json:"title"`
	Game  string         `json:"game"`
	Type  string         `json:"type"`
	Files []FileMetadata `json:"files"`
}

type FileMetadata struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Url         string `json:"url"`
	Type        string `json:"type"`
	GameVersion string `json:"version"`
	UploadedAt  string `json:"uploaded_at"`
}

func (metadata *AddonMetaData) GetLatestFile(majorGameVersion string) (*FileMetadata, error) {
	var latest *FileMetadata
	for i, f := range metadata.Files {
		if strings.HasPrefix(f.GameVersion, majorGameVersion) && f.Type == "release" {
			if latest == nil {
				latest = &metadata.Files[i]
			} else {
				latestUploadedAt, err := time.Parse(time.RFC3339, latest.UploadedAt)
				if err != nil {
					return nil, fmt.Errorf("failed to parse UploadedAt=%v for latest file with Id=%v and Name=%v: %w", latest.UploadedAt, latest.Id, latest.Name, err)
				}
				currentUploadedAt, err := time.Parse(time.RFC3339, f.UploadedAt)
				if err != nil {
					return nil, fmt.Errorf("failed to parse UploadedAt=%v for current file with Id=%v and Name=%v: %w", f.UploadedAt, f.Id, f.Name, err)
				}
				if currentUploadedAt.After(latestUploadedAt) {
					latest = &metadata.Files[i]
				}
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("did not find a released version of addon=%v for majorGameVersion=%v", metadata.Title, majorGameVersion)
	}
	return latest, nil
}

func (metadata *AddonMetaData) Validate() error {
	if metadata.Game != "wow" {
		return fmt.Errorf("the addon is not associated with WoW. metadata=%v", metadata)
	}
	return nil
}

func Fetch(addonName string) (*http.Response, error) {
	resp, err := http.Get("https://api.cfwidget.com/wow/addons/" + addonName)

	if err != nil {
		return nil, fmt.Errorf("failed to download project metadata for addon=%v: %w", addonName, err)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 202 {
			return nil, fmt.Errorf("failed to download project metadata for addon=%v because it was not cached, but a fetch has been queued. retry later", addonName)
		}
		return nil, fmt.Errorf("failed to download project metadata for addon=%v, status=%v", addonName, resp.StatusCode)
	}

	return resp, nil
}

func Decode(r io.Reader) AddonMetaData {
	decoder := json.NewDecoder(r)

	var metadata AddonMetaData
	decoder.Decode(&metadata)

	for i := range metadata.Files {
		idString := strconv.Itoa(metadata.Files[i].Id)
		id1 := idString[:4]
		id2 := strings.TrimLeft(idString[4:], "0")
		if id2 == "" {
			id2 = "0"
		}
		metadata.Files[i].Url = "https://media.forgecdn.net/files/" + id1 + "/" + id2 + "/" + metadata.Files[i].Name
	}

	return metadata
}
