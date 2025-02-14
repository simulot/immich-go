package fshelper

import (
	"encoding/json"
	"io/fs"
)

// readJSON reads a JSON file from the provided file system (fs.FS)
// with the given name and unmarshals it into the provided type T.

func ReadJSON[T any](fsys fs.FS, name string) (*T, error) {
	var object T
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &object)
	if err != nil {
		return nil, err
	}

	return &object, nil
}

func UnmarshalJSON[T any](b []byte) (*T, error) {
	var object T
	err := json.Unmarshal(b, &object)
	if err != nil {
		return nil, err
	}

	return &object, nil
}
