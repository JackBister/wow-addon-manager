package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackbister/addon-manager/metadata"
	"github.com/pkg/errors"
)

type AddonsJSON struct {
	Addons    []string `json:"addons"`
	WowFolder string   `json:"wowFolder"`
}

type LockFile struct {
	Versions map[string]int `json:"versions"`
}

var gAddonsJSON AddonsJSON
var gLockFile = LockFile{Versions: make(map[string]int)}

func main() {
	readLockFile()

	file, err := os.Open("addons.json")
	if err != nil {
		panic(err)
	}

	jsonDecoder := json.NewDecoder(file)

	err = jsonDecoder.Decode(&gAddonsJSON)
	if err != nil {
		panic(err)
	}

	for _, v := range gAddonsJSON.Addons {
		err = downloadAddon(v)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	err = writeLockFile()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func downloadAddon(addonName string) error {
	resp, err := metadata.Fetch(addonName)
	if err != nil {
		return err
	}

	addonMetadata := metadata.Decode(resp.Body)
	err = metadata.Validate(addonMetadata)
	if err != nil {
		return errors.Wrap(err, addonName+" has invalid metadata.")
	}

	if gLockFile.Versions[addonName] == addonMetadata.Download.Id {
		return errors.New(addonName + " with version " + strconv.Itoa(addonMetadata.Download.Id) + " is already present in lockfile")
	}

	resp, err = downloadAddonFromURL(addonMetadata.Download.Url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	err = writeAddonToFile(resp.Body, addonMetadata.Download.Name)
	if err != nil {
		return err
	}

	fileNames, err := unzip(path.Join(getAddonsPath(), addonMetadata.Download.Name), getAddonsPath())
	if err != nil {
		return errors.Wrap(err, "Failed to unzip addon "+addonName)
	}

	err = os.Remove(path.Join(getAddonsPath(), addonMetadata.Download.Name))
	if err != nil {
		return errors.Wrap(err, "Failed to delete zip file for addon "+addonName)
	}

	fmt.Println("Unzipped", len(fileNames), "files for addon", addonName)

	gLockFile.Versions[addonName] = addonMetadata.Download.Id

	return nil
}

func downloadAddonFromURL(url string) (*http.Response, error) {
	resp, err := http.Get(url + "/file")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("Status code was not 200 at download URL")
	}

	return resp, nil
}

func writeAddonToFile(addon io.Reader, fileName string) error {
	file, err := os.Create(path.Join(getAddonsPath(), fileName))

	if err != nil {
		return errors.Wrap(err, "Couldn't create addon zip file")
	}

	defer file.Close()

	_, err = io.Copy(file, addon)

	if err != nil {
		return errors.Wrap(err, "Couldn't write to addon zip file")
	}

	return nil
}

func getAddonsPath() string {
	return path.Join(gAddonsJSON.WowFolder, "Interface", "AddOns")
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	zipDir, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer zipDir.Close()

	for _, file := range zipDir.File {
		rc, err := file.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		fpath := filepath.Join(dest, file.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if file.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return filenames, err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return filenames, err
			}

			_, err = io.Copy(outFile, rc)

			outFile.Close()

			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}

func readLockFile() error {
	file, err := os.Open("addons.lock.json")
	if err != nil {
		return errors.Wrap(err, "Couldn't open lockfile")
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&gLockFile)
	if err != nil {
		return errors.Wrap(err, "Couldn't decode lockfile")
	}

	return nil
}

func writeLockFile() error {
	file, err := os.Create("addons.lock.json")
	if err != nil {
		return errors.Wrap(err, "Couldn't create lockfile")
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(&gLockFile)
	if err != nil {
		return errors.Wrap(err, "Couldn't encode lockfile")
	}

	return nil
}
