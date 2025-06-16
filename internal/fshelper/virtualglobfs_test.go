package fshelper

import (
	"io/fs"
	"path"
	"reflect"
	"testing"
)

func Test_VirtualGlobFS(t *testing.T) {
	tc := []struct {
		pattern        string
		expected       []string
		expectedFSName string
	}{
		{
			// Exact match
			pattern:        "A/T/10.jpg",
			expected:       []string{"10.jpg"},
			expectedFSName: "T",
		},
		{
			// Also return XMP files, even on exact matches
			pattern:        "C.JPG",
			expected:       []string{"C.JPG", "C.XMP"},
			expectedFSName: "TESTDATA",
		},
		{
			// All files, of all types, in a directory
			pattern: "A/T/*.*",
			expected: []string{
				"10.jpg",
				"10.json",
			},
			expectedFSName: "T",
		},
		{
			// All files of one type in a directory (and always XMP)
			pattern: "B/T/*.jpg",
			expected: []string{
				"20.jpg",
				"20.xmp",
			},
			expectedFSName: "T",
		},
		{
			// All files in directories called "T"
			pattern: "*/T/*.*",
			expected: []string{
				"A/T/10.jpg",
				"A/T/10.json",
				"B/4.xmp",
				"B/T/20.jpg",
				"B/T/20.json",
				"B/T/20.xmp",
				"C.XMP",
			},
			expectedFSName: "TESTDATA",
		},
		{
			// All JPG (and XMP) files in directories called "T"
			pattern: "*/T/*.jpg",
			expected: []string{
				"A/T/10.jpg",
				"B/4.xmp",
				"B/T/20.jpg",
				"B/T/20.xmp",
				"C.XMP",
			},
			expectedFSName: "TESTDATA",
		},
		{
			// All JPGs (and XMP) of top-level directories
			pattern: "*/*.jpg",
			expected: []string{
				"A/1.jpg",
				"A/2.jpg",
				"B/4.jpg",
				"B/4.xmp",
				"C.XMP",
			},
			expectedFSName: "TESTDATA",
		},
		{
			// All contents of directory 'A'
			pattern: "A",
			expected: []string{
				"1.jpg",
				"1.json",
				"2.jpg",
				"2.json",
				"T/10.jpg",
				"T/10.json",
			},
			expectedFSName: "A",
		},
		{
			// Top level patterns
			pattern: "*.jpg",
			expected: []string{
				"C.JPG",
				"C.XMP",
			},
			expectedFSName: "TESTDATA",
		},
	}

	for _, c := range tc {
		t.Run(c.pattern, func(t *testing.T) {
			fsys, err := NewVirtualGlobFS(path.Join("TESTDATA", c.pattern))
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
				t.Errorf("unexpected filelist; expected %v, got %v", c.expected, files)
			}
			if c.expectedFSName != fsys.(NameFS).Name() {
				t.Errorf("unexpected FSName; expected %v, got %v", c.expectedFSName, fsys.(NameFS).Name())
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
		{
			name:  "*.JPG",
			want:  "",
			want1: "*.JPG",
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
