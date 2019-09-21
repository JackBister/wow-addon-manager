package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jackbister/wow-addon-manager/addon"
	"github.com/jackbister/wow-addon-manager/metadata"
	"github.com/jackbister/wow-addon-manager/versionfile"
	"github.com/pkg/errors"
)

type AddonsJSON struct {
	Addons    []string `json:"addons"`
	WowFolder string   `json:"wowFolder"`
}

var gAddonsJSON AddonsJSON

func main() {
	fileName := "addons.json"
	flag.Parse()
	if len(flag.Args()) > 0 {
		fileName = flag.Args()[0]
	}

	fileNameWithoutExtension := strings.Split(fileName, ".")[0]

	lockFile, err := versionfile.FromFile(fileNameWithoutExtension + ".lock.json")
	if err != nil {
		fmt.Println(err.Error())
		lockFile = versionfile.New()
	}

	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}

	jsonDecoder := json.NewDecoder(file)

	err = jsonDecoder.Decode(&gAddonsJSON)
	if err != nil {
		panic(err)
	}

	for _, v := range gAddonsJSON.Addons {
		err = downloadAddon(v, lockFile.GetVersion, lockFile.PutVersion)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	err = lockFile.ToFile("addons.lock.json")
	if err != nil {
		fmt.Println(err.Error())
	}
}

func downloadAddon(addonName string, getVersion func(name string) int, putVersion func(name string, version int)) error {
	resp, err := metadata.Fetch(addonName)
	if err != nil {
		return err
	}

	addonMetadata := metadata.Decode(resp.Body)
	err = addonMetadata.Validate()
	if err != nil {
		return errors.Wrap(err, addonName+" has invalid metadata.")
	}

	if getVersion(addonName) == addonMetadata.Download.Id {
		return errors.New(addonName + " with version " + strconv.Itoa(addonMetadata.Download.Id) + " is already present in lockfile")
	}

	addonDownload, err := addon.Download(addonMetadata.Download.Url)
	if err != nil {
		return err
	}

	zipPath := path.Join(getAddonsPath(), addonMetadata.Download.Name)

	err = addonDownload.ToFile(zipPath)
	if err != nil {
		return err
	}

	fileNames, err := unzip(zipPath, getAddonsPath())
	if err != nil {
		return errors.Wrap(err, "Failed to unzip addon "+addonName)
	}

	err = os.Remove(path.Join(getAddonsPath(), addonMetadata.Download.Name))
	if err != nil {
		return errors.Wrap(err, "Failed to delete zip file for addon "+addonName)
	}

	fmt.Println("Unzipped", len(fileNames), "files for addon", addonName)

	putVersion(addonName, addonMetadata.Download.Id)

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
