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
	rootFS NameFS
	files  map[string]fs.DirEntry
	dirs   map[string][]string
	parts  []string
}

var _ fs.FS = GlobWalkFS{}
var _ fs.ReadDirFS = GlobWalkFS{}
var _ fs.StatFS = GlobWalkFS{}
var _ NameFS = GlobWalkFS{}

func NewGlobWalkFS(pattern string) (fs.FS, error) {
	rootFs, parts, err := getRootFs(pattern)
	if err != nil {
		return nil, err
	}

	gwfs := &GlobWalkFS{
		rootFS: rootFs,
		files:  make(map[string]fs.DirEntry),
		dirs:   make(map[string][]string),
		parts:  parts,
	}

	err = fs.WalkDir(gwfs.rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// directories should always be added
		parentDir := filepath.Dir(path)
		if parentDir != path {
			gwfs.dirs[parentDir] = append(gwfs.dirs[parentDir], path)
			gwfs.files[path] = d
			return nil
		}

		if !gwfs.match(path) {
			return nil
		}

		gwfs.files[path] = d
		return nil
	})

	return gwfs, err
}

func getRootFs(pattern string) (NameFS, []string, error) {
	dir, magic := FixedPathAndMagic(pattern)
	if magic == "" {
		return simpleRootFS(dir)
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}

	return NewFSWithName(dir), strings.Split(strings.ToLower(magic), string(os.PathSeparator)), nil
}

func simpleRootFS(root string) (NameFS, []string, error) {
	s, err := os.Stat(root)
	if err != nil {
		return nil, nil, err
	}

	if !s.IsDir() {
		// In the case of a file, we use the filename as the matching "magic" component.
		file := strings.ToLower(path.Base(root))
		rootDir := path.Dir(root)
		if rootDir == "" {
			rootDir, _ = os.Getwd()
		}
		return NewFSWithName(rootDir), []string{file}, nil
	}

	// Otherwise, we just use the directory
	return NewFSWithName(root), nil, nil
}

// match the current file name with the pattern
// matches files having a path starting by the patten
//
//	ex:  file /path/to/file matches with the pattern /*/to
func (gw GlobWalkFS) match(filename string) bool {
	if filename == "." {
		return true
	}

	lowerFName := strings.ToLower(filename)
	if strings.HasSuffix(lowerFName, ".xmp") {
		return true
	}

	parts := strings.Split(lowerFName, string(os.PathSeparator))

	for i := 0; i < min(len(gw.parts), len(parts)); i++ {
		if m, err := path.Match(gw.parts[i], parts[i]); err != nil || !m {
			return false
		}
	}

	parts = strings.Split(filename, string(os.PathSeparator))
	if len(gw.parts) > len(parts) {
		join := path.Join(parts[:min(len(gw.parts), len(parts))]...)
		if f, ok := gw.files[join]; !ok || !f.IsDir() {
			return false
		}
	}
	return true
}

// Open the name only if the name matches with the pattern
func (gw GlobWalkFS) Open(name string) (fs.File, error) {
	if _, ok := gw.files[name]; !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return gw.rootFS.Open(name)
}

// Stat the name only if the name matches with the pattern
func (gw GlobWalkFS) Stat(name string) (fs.FileInfo, error) {
	fi, ok := gw.files[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return fi.Info()
}

// ReadDir return all DirEntries that match with the pattern or .XMP files
func (gw GlobWalkFS) ReadDir(name string) ([]fs.DirEntry, error) {
	files, ok := gw.dirs[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}

	entries := make([]fs.DirEntry, 0, len(files))
	for _, f := range files {
		if !gw.match(f) {
			continue
		}
		entries = append(entries, gw.files[f])
	}
	return entries, nil
}

// Name gives the folder name when the argument was '.'
func (gw GlobWalkFS) Name() string {
	return gw.rootFS.Name()
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

// NameFS creates an fs.FS which can be named to pass around details.
type NameFS interface {
	fs.FS
	Name() string
}
type FSWithName struct {
	fs.FS
	name string
}

var _ NameFS = FSWithName{}

// NewFSWithName creates a new rooted filesystem at the provided root. See os.DirFS for details.
// It then extends that by naming the FS by the base directory that's being used.
func NewFSWithName(root string) NameFS {
	return &FSWithName{
		name: filepath.Base(root),
		FS:   os.DirFS(root),
	}
}

func (f FSWithName) Name() string {
	return f.name
}
