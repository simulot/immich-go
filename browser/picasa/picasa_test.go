package picasa

import (
	"bufio"
	"fmt"
	"github.com/psanford/memfs"
	"github.com/spf13/afero"
	"reflect"
	"strings"
	"testing"
)

type inMemFS struct {
	*memfs.FS
	err error
}

func newInMemFS() *inMemFS {
	return &inMemFS{
		FS: memfs.New(),
	}
}

func Test2(t *testing.T) {
	expected := DirectoryData{
		Name:        "A Name",
		Description: "A Description",
		Location:    "A Location",
		Files: map[string]FileData{
			"file-name-1.jpg": {},
			"file-name-2.jpg": {
				IsStar:  true,
				Caption: "A Caption",
			},
		},
		Albums: map[string]AlbumData{},
	}

	sample := `
[Picasa]
name=A Name
description=A Description
location=A Location

[file-name-1.jpg]
some_other_key=some value

[file-name-2.jpg]
star=yes
caption=A Caption
`

	appFS = afero.NewMemMapFs()
	// create test files and directories
	appFS.MkdirAll("sample", 0o755)
	afero.WriteFile(appFS, "sample/.picasa.ini", []byte(sample), 0o644)

	actual := ParseDirectory("sample")

	if !reflect.DeepEqual(expected, actual) {
		fmt.Printf("%+v\n", expected)
		fmt.Printf("%+v\n", actual)
		t.Error("ParseDirectory did not yield expected results")
	}
}

func TestReadLines(t *testing.T) {
	sample := `
[Picasa]
key1=value1
key2=value2

[file-name.jpg]
key3=value3
key4=value4
`
	buf := strings.NewReader(sample)
	s := bufio.NewScanner(buf)

	actual := parseScanner(s)
	expected := map[string]map[string]string{
		"Picasa": {
			"key1": "value1",
			"key2": "value2",
		},
		"file-name.jpg": {
			"key3": "value3",
			"key4": "value4",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Error("parsed ini did not match expected")
	}
}
