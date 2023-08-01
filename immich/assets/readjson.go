package assets

import (
	"encoding/json"
	"io/fs"
)

// readJSON reads a JSON file from the provided file system (fs.FS)
// with the given name and unmarshals it into the provided type T.

func readJSON[T any](FSys fs.FS, name string) (*T, error) {
	var object T
	b, err := fs.ReadFile(FSys, name)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &object)
	if err != nil {
		return nil, err
	}

	return &object, nil
}
