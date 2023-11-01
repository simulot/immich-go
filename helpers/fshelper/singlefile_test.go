package fshelper

import (
	"io/fs"
	"reflect"
	"testing"
)

func TestSingleFile(t *testing.T) {
	fsys, err := newSingleFileFS("singlefile_test.go")
	if err != nil {
		t.Error(err)
		return
	}

	l, err := fsys.ReadDir(".")
	if err != nil {
		t.Error(err)
		return
	}
	if len(l) != 1 {
		t.Errorf("unexpected number of items: %d", len(l))
		return
	}
	if l[0].Name() != "singlefile_test.go" {
		t.Errorf("unexpected file number: %s", l[0].Name())
		return
	}

	found := []string{}
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		found = append(found, path)
		return err
	})
	if !reflect.DeepEqual(found, []string{".", "singlefile_test.go"}) {
		t.Errorf("unexpected walkdir result")
		return
	}

	f, err := fsys.Open("singlefile_test.go")
	if err != nil {
		t.Errorf("unexpected error while opening an existing file: %s", err)
		return
	}
	f.Close()

	f, err = fsys.Open("doesnotexist.md")
	if err == nil {
		t.Error("must have an error when opening an non existing file")
		return
	}

}
