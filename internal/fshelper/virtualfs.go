package fshelper

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type VFS struct {
	rootFS fs.FS
	files  map[string]fs.DirEntry
	dirs   map[string][]string
	parts  []string
}

var _ fs.FS = VFS{}
var _ fs.ReadDirFS = VFS{}
var _ fs.StatFS = VFS{}

func getRootFs(root string) (fs.FS, []string, error) {
	dir, magic := FixedPathAndMagic(root)
	if magic == "" {
		return simpleRootFS(dir)
	}

	if dir == "" {
		dir, _ = os.Getwd()
	}

	return os.DirFS(dir), strings.Split(strings.ToLower(magic), string(os.PathSeparator)), nil
}

func simpleRootFS(root string) (fs.FS, []string, error) {
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
		return os.DirFS(rootDir), []string{file}, nil
	}

	// Otherwise, we just use the directory
	return os.DirFS(root), nil, nil
}

func NewVFS(root string) (fs.FS, error) {
	rootFs, parts, err := getRootFs(root)
	if err != nil {
		return nil, err
	}

	vfs := &VFS{
		rootFS: rootFs,
		files:  make(map[string]fs.DirEntry),
		dirs:   make(map[string][]string),
		parts:  parts,
	}

	err = fs.WalkDir(vfs.rootFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// directories should always be added
		parentDir := filepath.Dir(path)
		if parentDir != path {
			vfs.dirs[parentDir] = append(vfs.dirs[parentDir], path)
			vfs.files[path] = d
			return nil
		}

		if !vfs.match(path) {
			return nil
		}

		vfs.files[path] = d
		return nil
	})

	return vfs, err
}

func (v VFS) ReadDir(name string) ([]fs.DirEntry, error) {
	files, ok := v.dirs[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	entries := make([]fs.DirEntry, 0, len(files))
	for _, f := range files {
		if !v.match(f) {
			continue
		}
		entries = append(entries, v.files[f])
	}
	return entries, nil
}

func (v VFS) Stat(name string) (fs.FileInfo, error) {
	fi, ok := v.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return fi.Info()
}

func (v VFS) Open(name string) (fs.File, error) {
	if _, ok := v.files[name]; !ok {
		return nil, os.ErrNotExist
	}
	return v.rootFS.Open(name)
}

// match the current file name with the pattern
// matches files having a path starting by the patten
//
//	ex:  file /path/to/file matches with the pattern /*/to
func (v VFS) match(filename string) bool {
	if filename == "." {
		return true
	}

	parts := strings.Split(strings.ToLower(filename), string(os.PathSeparator))

	for i := 0; i < min(len(v.parts), len(parts)); i++ {
		if m, err := path.Match(v.parts[i], parts[i]); err != nil || !m {
			return false
		}
	}

	parts = strings.Split(filename, string(os.PathSeparator))
	if len(v.parts) > len(parts) {
		join := path.Join(parts[:min(len(v.parts), len(parts))]...)
		if f, ok := v.files[join]; !ok || !f.IsDir() {
			return false
		}
	}
	return true
}
