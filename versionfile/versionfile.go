package versionfile

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

type VersionFile struct {
	versions map[string]int
}

func New() *VersionFile {
	return &VersionFile{versions: make(map[string]int)}
}

func (v *VersionFile) GetVersion(name string) int {
	return v.versions[name]
}

func (v *VersionFile) PutVersion(name string, version int) {
	v.versions[name] = version
}

func (v *VersionFile) ToFile(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Wrap(err, "Couldn't create lockfile")
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(v)
	if err != nil {
		return errors.Wrap(err, "Couldn't encode lockfile")
	}

	return nil
}

// JSON marshalling functions to circumvent Go not being able to encode/decode unexported fields

func (v *VersionFile) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.versions)
}

func (v *VersionFile) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &v.versions)
}

func FromFile(fileName string) (*VersionFile, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't open lockfile")
	}
	defer file.Close()

	ret := New()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ret)
	if err != nil {
		return nil, errors.Wrap(err, "Couldn't decode lockfile")
	}

	return ret, nil
}
