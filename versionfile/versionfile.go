package versionfile

import (
	"encoding/json"
	"fmt"
	"os"
)

type VersionFile struct {
	versions map[string]map[string]int
}

func New() *VersionFile {
	return &VersionFile{versions: make(map[string]map[string]int)}
}

func (v *VersionFile) GetVersion(prefix string, name string) int {
	if m, ok := v.versions[prefix]; ok {
		return m[name]
	}
	return 0
}

func (v *VersionFile) PutVersion(prefix string, name string, version int) {
	if _, ok := v.versions[prefix]; !ok {
		v.versions[prefix] = map[string]int{name: version}
		return
	}
	v.versions[prefix][name] = version
}

func (v *VersionFile) ToFile(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("couldn't create lockfile: %w", err)
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(v)
	if err != nil {
		return fmt.Errorf("couldn't encode lockfile: %w", err)
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
		return nil, fmt.Errorf("couldn't open lockfile: %w", err)
	}
	defer file.Close()

	ret := New()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ret)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode lockfile: %w", err)
	}

	return ret, nil
}
