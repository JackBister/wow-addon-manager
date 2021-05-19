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
	"strings"

	"github.com/jackbister/wow-addon-manager/addon"
	"github.com/jackbister/wow-addon-manager/metadata"
	"github.com/jackbister/wow-addon-manager/versionfile"
)

type GameVersionJSON struct {
	Prefix           string   `json:"prefix"`
	MajorGameVersion string   `json:"majorGameVersion"`
	Addons           []string `json:"addons"`
}

type AddonsJSON struct {
	WowFolder string            `json:"wowFolder"`
	Versions  []GameVersionJSON `json:"versions"`
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

	for _, v := range gAddonsJSON.Versions {
		fmt.Printf("Downloading addons for majorGameVersion=%v, prefix=%v\n", v.MajorGameVersion, v.Prefix)
		for _, a := range v.Addons {
			err = downloadAddon(
				v.Prefix,
				v.MajorGameVersion,
				a,
				func(name string) int {
					return lockFile.GetVersion(v.Prefix, name)
				},
				func(name string, version int) {
					lockFile.PutVersion(v.Prefix, name, version)
				})
			if err != nil {
				fmt.Println(err.Error())
			}
		}

	}

	err = lockFile.ToFile("addons.lock.json")
	if err != nil {
		fmt.Println(err.Error())
	}
}

func downloadAddon(prefix string, majorGameVersion string, addonName string, getVersion func(name string) int, putVersion func(name string, version int)) error {
	resp, err := metadata.Fetch(addonName)
	if err != nil {
		return err
	}

	addonMetadata := metadata.Decode(resp.Body)
	err = addonMetadata.Validate()
	if err != nil {
		return fmt.Errorf("%v has invalid metadata: %w", addonName, err)
	}

	latest, err := addonMetadata.GetLatestFile(majorGameVersion)
	if err != nil {
		return fmt.Errorf("failed to get latest version of addon=%v: %w", addonName, err)
	}
	fmt.Printf("Latest Id=%v for addon=%v\n", latest.Id, addonName)

	if getVersion(addonName) == latest.Id {
		return fmt.Errorf("%v with version %v is already present in lockfile", addonName, latest.Id)
	}

	addonDownload, err := addon.Download(latest.Url)
	if err != nil {
		return err
	}

	addonsPath := getAddonsPath(prefix)

	zipPath := path.Join(addonsPath, latest.Name)

	err = addonDownload.ToFile(zipPath)
	if err != nil {
		return err
	}

	fileNames, err := unzip(zipPath, addonsPath)
	if err != nil {
		return fmt.Errorf("failed to unzip addon %v: %w", addonName, err)
	}

	err = os.Remove(path.Join(addonsPath, latest.Name))
	if err != nil {
		return fmt.Errorf("failed to delete zip file for addon %v: %w", addonName, err)
	}

	fmt.Println("Unzipped", len(fileNames), "files for addon", addonName)

	putVersion(addonName, latest.Id)

	return nil
}

func getAddonsPath(prefix string) string {
	return path.Join(gAddonsJSON.WowFolder, prefix, "Interface", "AddOns")
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
