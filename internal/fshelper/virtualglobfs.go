package fshelper

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//  VirtualGlobFS create a FS that limits the WalkDir function to the
//  list of files that match the glob expression, and cheats to
//  matches *.XMP files in all circumstances
//
//  It implements ReadDir and Stat to filter the file list
//

type VirtualGlobFS struct {
	rootFS NameFS
	files  map[string]fs.DirEntry
	dirs   map[string][]string
	parts  []string
}

var (
	_ fs.FS        = VirtualGlobFS{}
	_ fs.ReadDirFS = VirtualGlobFS{}
	_ fs.StatFS    = VirtualGlobFS{}
	_ NameFS       = VirtualGlobFS{}
)

func NewVirtualGlobFS(pattern string) (fs.FS, error) {
	rootFs, parts, err := getRootFs(pattern)
	if err != nil {
		return nil, err
	}

	vfs := &VirtualGlobFS{
		rootFS: rootFs,
		files:  make(map[string]fs.DirEntry),
		dirs:   make(map[string][]string),
		parts:  parts,
	}

	err = fs.WalkDir(vfs.rootFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// normalize to unix style paths
		p = filepath.ToSlash(p)

		// directories should always be added
		parentDir := path.Dir(p)
		if parentDir != p {
			vfs.dirs[parentDir] = append(vfs.dirs[parentDir], p)
			vfs.files[p] = d
			return nil
		}

		if !vfs.match(p) {
			return nil
		}

		vfs.files[p] = d
		return nil
	})

	return vfs, err
}

func getRootFs(pattern string) (NameFS, []string, error) {
	dir, magic := FixedPathAndMagic(pattern)
	if magic == "" {
		return simpleRootFS(dir)
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}

	return NewFSWithName(dir), strings.Split(strings.ToLower(magic), "/"), nil
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
func (gw VirtualGlobFS) match(filename string) bool {
	if filename == "." {
		return true
	}

	lowerFName := strings.ToLower(filename)
	if strings.HasSuffix(lowerFName, ".xmp") {
		return true
	}

	parts := strings.Split(lowerFName, "/")

	for i := 0; i < min(len(gw.parts), len(parts)); i++ {
		if m, err := path.Match(gw.parts[i], parts[i]); err != nil || !m {
			return false
		}
	}

	parts = strings.Split(filename, "/")
	if len(gw.parts) > len(parts) {
		join := path.Join(parts[:min(len(gw.parts), len(parts))]...)
		if f, ok := gw.files[join]; !ok || !f.IsDir() {
			return false
		}
	}
	return true
}

// Open the name only if the name matches with the pattern
func (gw VirtualGlobFS) Open(name string) (fs.File, error) {
	if _, ok := gw.files[name]; !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return gw.rootFS.Open(name)
}

// Stat the name only if the name matches with the pattern
func (gw VirtualGlobFS) Stat(name string) (fs.FileInfo, error) {
	fi, ok := gw.files[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return fi.Info()
}

// ReadDir return all DirEntries that match with the pattern or .XMP files
func (gw VirtualGlobFS) ReadDir(name string) ([]fs.DirEntry, error) {
	// canonicalize to unix style paths
	name = filepath.ToSlash(name)

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
func (gw VirtualGlobFS) Name() string {
	return gw.rootFS.Name()
}

// FixedPathAndMagic split the path with the fixed part and the variable part
func FixedPathAndMagic(name string) (string, string) {
	// canonicalize to unix style paths
	name = filepath.ToSlash(name)

	if !HasMagic(name) {
		return name, ""
	}
	parts := strings.Split(name, "/")
	p := 0
	for p = range parts {
		if HasMagic(parts[p]) {
			break
		}
	}
	fixed := ""

	// this root name only works on Unix-like systems
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
