package fshelper

import (
	"io/fs"
	"os"
	"reflect"
	"testing"
)

func Test_GlobWalkFS(t *testing.T) {
	tc := []struct {
		pattern  string
		expected []string
	}{
		{
			pattern: "A/T/10.jpg",
			expected: []string{
				"A/T/10.jpg",
			},
		},
		{
			pattern: "A/T/*.*",
			expected: []string{
				"A/T/10.jpg",
				"A/T/10.json",
			},
		},
		{
			pattern: "A/T/*.jpg",
			expected: []string{
				"A/T/10.jpg",
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
				"A/1.jpg",
				"A/1.json",
				"A/2.jpg",
				"A/2.json",
				"A/T/10.jpg",
				"A/T/10.json",
			},
		},
	}

	for _, c := range tc {
		t.Run(c.pattern, func(t *testing.T) {
			fsys, err := NewGlobWalkFS(os.DirFS("TESTDATA"), c.pattern)
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
				t.Error("Result differs")
			}
		})
	}
}

func TestFixedPathAndMagic(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		want1 string
	}{
		{
			name:  "A/B/C/file",
			want:  "A/B/C/file",
			want1: "",
		},
		{
			name:  "A/B/C/*.*",
			want:  "A/B/C",
			want1: "*.*",
		},
		{
			name:  "A/*/C/file",
			want:  "A",
			want1: "*/C/file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FixedPathAndMagic(tt.name)
			if got != tt.want {
				t.Errorf("FixedPathAndMagic() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("FixedPathAndMagic() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
