package fshelper

import (
	"github.com/gocarina/gocsv"
	"io/fs"
)

// ReadCSV reads a CSV file from the provided file system (fs.FS)
// with the given name and unmarshals it into the provided type T.

func ReadCSV[T any](FSys fs.FS, name string) ([]T, error) {
	var objects []T
	b, err := fs.ReadFile(FSys, name)
	if err != nil {
		return nil, err
	}

	err = gocsv.UnmarshalBytes(b, &objects)
	if err != nil {
		return nil, err
	}

	return objects, nil
}
