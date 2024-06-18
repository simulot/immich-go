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
			pattern:  "A/T/10.jpg",
			expected: []string{"./A/T/10.jpg"},
		},
		// {
		// 	pattern: "A/T/*.*",
		// 	expected: []string{
		// 		"A/T/10.jpg",
		// 		"A/T/10.json",
		// 	},
		// },
		// {
		// 	pattern: "A/T/*.jpg",
		// 	expected: []string{
		// 		"A/T/10.jpg",
		// 	},
		// },
		// {
		// 	pattern: "*/T/*.*",
		// 	expected: []string{
		// 		"A/T/10.jpg",
		// 		"A/T/10.json",
		// 		"B/T/20.jpg",
		// 		"B/T/20.json",
		// 	},
		// },
		// {
		// 	pattern: "*/T/*.jpg",
		// 	expected: []string{
		// 		"A/T/10.jpg",
		// 		"B/T/20.jpg",
		// 	},
		// },
		// {
		// 	pattern: "*/*.jpg",
		// 	expected: []string{
		// 		"A/1.jpg",
		// 		"A/2.jpg",
		// 		"B/4.jpg",
		// 	},
		// },
		// {
		// pattern: "A",
		// expected: []string{
		// 	"A/1.jpg",
		// 	"A/1.json",
		// 	"A/2.jpg",
		// 	"A/2.json",
		// 	"A/T/10.jpg",
		// 	"A/T/10.json",
		// },
		// },
		// {
		// 	pattern: "*.jpg",
		// 	expected: []string{
		// 		"C.jpg",
		// 	},
		// },
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
			want:  "./A/B/C/file",
			want1: "",
		},
		{
			name:  "A/B/C/*.*",
			want:  "./A/B/C",
			want1: "*.*",
		},
		{
			name:  "A/*/C/file",
			want:  "./A",
			want1: "*/C/file",
		},
		{
			name:  "*.JPG",
			want:  "./",
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

func Test_patternRegexp(t *testing.T) {
	type testSet struct {
		file  string
		match bool
	}

	tests := []struct {
		name  string
		want  string
		tests []testSet
	}{
		{
			name: "IMAGE.JPG",
			want: `(?mi)^IMAGE\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: true},
				{file: "IMAGE.jpg", match: true},
				{file: ".JPG", match: false},
				{file: "A/ABC.JPG", match: false},
			},
		},
		{
			name: "*.JPG",
			want: `(?mi)^[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: true},
				{file: ".JPG", match: true},
				{file: "A/ABC.JPG", match: false},
			},
		},
		{
			name: "A/*.JPG",
			want: `(?mi)^A/[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: true},
				{file: "A/ABC.JPG", match: true},
			},
		},
		{
			name: "A/IMAGE.JPG",
			want: `(?mi)^A/IMAGE\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/ABC.JPG", match: false},
			},
		},
		{
			name: "A/AA/*.JPG",
			want: `(?mi)^A/AA/[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: false},
				{file: "A/AA/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/AA/.JPG", match: true},
				{file: "A/ABC.JPG", match: false},
				{file: "A/AA/ABC.JPG", match: true},
			},
		},
		{
			name: "A/AA/IMAGE.JPG",
			want: `(?mi)^A/AA/IMAGE\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: false},
				{file: "A/AA/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/AA/.JPG", match: false},
				{file: "A/ABC.JPG", match: false},
				{file: "A/AA/ABC.JPG", match: false},
			},
		},
		{
			name: "A/*/*.JPG",
			want: `(?mi)^A/[^/]*/[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: false},
				{file: "A/AA/IMAGE.JPG", match: true},
				{file: "B/BB/IMAGE.JPG", match: false},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/AA/.JPG", match: true},
				{file: "A/BB/.JPG", match: true},
				{file: "B/BB/.JPG", match: false},
				{file: "A/ABC.JPG", match: false},
				{file: "A/AA/ABC.JPG", match: true},
				{file: "A/BB/ABC.JPG", match: true},
				{file: "B/BB/ABC.JPG", match: false},
			},
		},
		{
			name: "*/B*/*.JPG",
			want: `(?mi)^[^/]*/B[^/]*/[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: false},
				{file: "A/AA/IMAGE.JPG", match: false},
				{file: "B/BB/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/AA/.JPG", match: false},
				{file: "A/BB/.JPG", match: true},
				{file: "B/BB/.JPG", match: true},
				{file: "A/ABC.JPG", match: false},
				{file: "A/AA/ABC.JPG", match: false},
				{file: "A/BB/ABC.JPG", match: true},
				{file: "B/BB/ABC.JPG", match: true},
			},
		},
		{
			name: "*/?B/*.JPG",
			want: `(?mi)^[^/]*/[^/]B/[^/]*\.JPG`,
			tests: []testSet{
				{file: "IMAGE.JPG", match: false},
				{file: "A/IMAGE.JPG", match: false},
				{file: "A/AA/IMAGE.JPG", match: false},
				{file: "B/BB/IMAGE.JPG", match: true},
				{file: ".JPG", match: false},
				{file: "A/.JPG", match: false},
				{file: "A/AA/.JPG", match: false},
				{file: "A/BB/.JPG", match: true},
				{file: "A/CB/.JPG", match: true},
				{file: "B/BB/.JPG", match: true},
				{file: "A/ABC.JPG", match: false},
				{file: "A/AA/ABC.JPG", match: false},
				{file: "A/BB/ABC.JPG", match: true},
				{file: "A/CB/ABC.JPG", match: true},
				{file: "B/BB/ABC.JPG", match: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patternRegexp(tt.name)
			if err != nil {
				t.Errorf("patternRegexp() error = %v", err)
				return
			}
			if tt.want != got.String() {
				t.Errorf("patternRegexp(%s)=%s want = %s", tt.name, got.String(), tt.want)
				return
			}
			for _, c := range tt.tests {
				if match := got.MatchString(c.file); match != c.match {
					t.Errorf("regexp:%s match(%s)=%v, got:%v ", got.String(), c.file, match, c.match)
					return
				}
			}
		})
	}
}
