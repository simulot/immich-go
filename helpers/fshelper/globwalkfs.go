package fshelper

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

//  GlobWalkFS create a FS that limits the WalkDir function to the
//  list of files that match the glob expression, and cheats to
//  matches *.XMP files in all circumstances
//
//  It implements ReadDir and Stat to filter the file list
//

type GlobWalkFS struct {
	rootFS  fs.FS
	pattern string
	parts   []string
}

func NewGlobWalkFS(fsys fs.FS, pattern string) (fs.FS, error) {
	pattern = filepath.ToSlash(pattern)

	return &GlobWalkFS{
		rootFS:  fsys,
		pattern: pattern,
		parts:   strings.Split(pattern, "/"),
	}, nil
}

// match the current file name with the pattern
// matches files having a path starting by the patten
//
//	ex:  file /path/to/file matches with the pattern /*/to
func (gw GlobWalkFS) match(name string) (bool, error) {
	if name == "." {
		return true, nil
	}
	nParts := strings.Split(name, "/")
	for i := 0; i < min(len(gw.parts), len(nParts)); i++ {
		match, err := path.Match(gw.parts[i], nParts[i])
		if !match || err != nil {
			return match, err
		}
	}
	return true, nil
}

// Open the name only if the name matches with the pattern
func (gw GlobWalkFS) Open(name string) (fs.File, error) {
	return gw.rootFS.Open(name)
}

// Stat the name only if the name matches with the pattern
func (gw GlobWalkFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(gw.rootFS, name)
}

// ReadDir return all DirEntries that match with the pattern or .XMP files
func (gw GlobWalkFS) ReadDir(name string) ([]fs.DirEntry, error) {
	match, err := gw.match(name)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, fs.ErrNotExist
	}
	entries, err := fs.ReadDir(gw.rootFS, name)
	if err != nil {
		return nil, err
	}

	returned := []fs.DirEntry{}
	for _, e := range entries {
		p := path.Join(name, e.Name())

		// Always matches .XMP files...
		if !e.IsDir() {
			ext := strings.ToUpper(path.Ext(e.Name()))
			if ext == ".XMP" {
				returned = append(returned, e)
				continue
			}
		}
		match, _ = gw.match(p)
		if match {
			returned = append(returned, e)
		}
	}
	return returned, nil
}

// FixedPathAndMagic split the path with the fixed part and the variable part
func FixedPathAndMagic(name string) (string, string) {
	if !HasMagic(name) {
		return name, ""
	}
	name = filepath.ToSlash(name)
	parts := strings.Split(name, "/")
	p := 0
	for p = range parts {
		if HasMagic(parts[p]) {
			break
		}
	}
	fixed := ""
	if name[0] == '/' {
		fixed = "/"
	}
	return fixed + path.Join(parts[:p]...), path.Join(parts[p:]...)
}
