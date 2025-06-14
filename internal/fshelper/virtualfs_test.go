package fshelper

import (
	"io/fs"
	"path"
	"reflect"
	"testing"
)

func Test_VirtualGlobFS(t *testing.T) {
	tc := []struct {
		pattern  string
		expected []string
	}{
		{
			pattern:  "A/T/10.jpg",
			expected: []string{"10.jpg"},
		},
		{
			pattern: "A/T/*.*",
			expected: []string{
				"10.jpg",
				"10.json",
			},
		},
		{
			pattern: "A/T/*.jpg",
			expected: []string{
				"10.jpg",
			},
		},
		{
			pattern: "*/T/*.*",
			expected: []string{
				"A/T/10.jpg",
				"A/T/10.json",
				"B/T/20.jpg",
				"B/T/20.json",
			},
		},
		{
			pattern: "*/T/*.jpg",
			expected: []string{
				"A/T/10.jpg",
				"B/T/20.jpg",
			},
		},
		{
			pattern: "*/*.jpg",
			expected: []string{
				"A/1.jpg",
				"A/2.jpg",
				"B/4.jpg",
			},
		},
		{
			pattern: "A",
			expected: []string{
				"1.jpg",
				"1.json",
				"2.jpg",
				"2.json",
				"T/10.jpg",
				"T/10.json",
			},
		},
		{
			pattern: "*.jpg",
			expected: []string{
				"C.JPG",
			},
		},
	}

	for _, c := range tc {
		t.Run(c.pattern, func(t *testing.T) {
			fsys, err := NewVFS(path.Join("TESTDATA", c.pattern))
			if err != nil {
				t.Error(err)
				return
			}

			files := []string{}

			err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
				if p == "." || d.IsDir() {
					return nil
				}
				files = append(files, p)
				return nil
			})
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(c.expected, files) {
				t.Error("Result differs", c.expected, files)
			}
		})
	}
}
