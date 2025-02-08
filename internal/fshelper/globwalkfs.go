package fshelper

import (
	"fmt"
	"io/fs"
	"os"
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
	rootFS fs.FS
	dir    string
	parts  []string
}

func NewGlobWalkFS(pattern string) (fs.FS, error) {
	dir, magic := FixedPathAndMagic(pattern)
	if magic == "" {
		s, err := os.Stat(dir)
		if err != nil {
			return nil, err
		}

		if !s.IsDir() {
			magic = strings.ToLower(path.Base(dir))
			dir = path.Dir(dir)
			if dir == "" {
				dir, _ = os.Getwd()
			}
			return &GlobWalkFS{
				rootFS: NewFSWithName(os.DirFS(dir), filepath.Base(dir)),
				dir:    dir,
				parts:  []string{magic},
			}, nil
		} else {
			name := filepath.Base(dir)
			if name == "." {
				name, _ = os.Getwd()
				name = filepath.Base(name)
			}

			return &GlobWalkFS{
				rootFS: NewFSWithName(os.DirFS(dir), name),
				dir:    dir,
			}, nil
		}
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	parts := strings.Split(magic, string(os.PathSeparator))
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}

	return &GlobWalkFS{
		rootFS: NewFSWithName(os.DirFS(dir), filepath.Base(dir)),
		dir:    dir,
		parts:  parts,
	}, nil
}

// match the current file name with the pattern
// matches files having a path starting by the patten
//
//	ex:  file /path/to/file matches with the pattern /*/to
func (gw GlobWalkFS) match(name string) bool {
	if name == "." {
		return true
	}

	parts := strings.Split(name, string(os.PathSeparator))
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	for i := 0; i < min(len(gw.parts), len(parts)); i++ {
		if m, err := path.Match(gw.parts[i], parts[i]); err != nil || !m {
			return false
		}
	}
	parts = strings.Split(name, string(os.PathSeparator))
	if len(gw.parts) > len(parts) {
		s, err := fs.Stat(gw, path.Join(parts[:min(len(gw.parts), len(parts))]...))
		if err != nil || !s.IsDir() {
			return false
		}
	}
	return true
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
	match := gw.match(name)
	if !match {
		return nil, fs.ErrNotExist
	}
	entries, err := fs.ReadDir(gw.rootFS, name)
	if err != nil {
		return nil, fmt.Errorf("ReadDir %s: %w", name, err)
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
		match = gw.match(p)
		if match {
			returned = append(returned, e)
		}
	}
	return returned, nil
}

// FSName gives the folder name when argument was .
func (gw GlobWalkFS) Name() string {
	if fsys, ok := gw.rootFS.(NameFS); ok {
		return fsys.Name()
	}
	return filepath.Base(gw.dir)
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

type FSWithName struct {
	name string
	fsys fs.FS
}

func NewFSWithName(fsys fs.FS, name string) fs.FS {
	return &FSWithName{
		name: name,
		fsys: fsys,
	}
}

func (f FSWithName) Open(name string) (fs.File, error) {
	return f.fsys.Open(name)
}

func (f FSWithName) Name() string {
	return f.name
}

func (f FSWithName) ReadDir(name string) ([]fs.DirEntry, error) {
	if fsys, ok := f.fsys.(fs.ReadDirFS); ok {
		return fsys.ReadDir(name)
	}
	return fs.ReadDir(f.fsys, name)
}

func (f FSWithName) Stat(name string) (fs.FileInfo, error) {
	if fsys, ok := f.fsys.(fs.StatFS); ok {
		return fsys.Stat(name)
	}
	return fs.Stat(f.fsys, name)
}

func (f FSWithName) ReadFile(name string) ([]byte, error) {
	if fsys, ok := f.fsys.(fs.ReadFileFS); ok {
		return fsys.ReadFile(name)
	}
	return fs.ReadFile(f.fsys, name)
}

type NameFS interface {
	Name() string
}
