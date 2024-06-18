package fshelper

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ParsePath return a list of FS bases on args
//
// Zip files are opened and returned as FS
// Manage wildcards in path
//
// TODO: Implement a tgz reader for non google-photos archives

func ParsePath(args []string, googlePhoto bool) ([]fs.FS, error) {
	var errs error
	fsyss := []fs.FS{}

	for _, a := range args {
		a = filepath.ToSlash(a)
		lowA := strings.ToLower(a)
		switch {
		case strings.HasSuffix(lowA, ".tgz") || strings.HasSuffix(lowA, ".tar.gz"):
			errs = errors.Join(fmt.Errorf("immich-go cant use tgz archives: %s", filepath.Base(a)))
		case strings.HasSuffix(lowA, ".zip"):
			files, err := expandNames(a)
			if err != nil {
				errs = errors.Join(errs, err)
				break
			}
			for _, f := range files {
				fsys, err := zip.OpenReader(f)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				fsyss = append(fsyss, fsys)
			}
		default:
			fixed, magic := FixedPathAndMagic(a)
			if magic == "" {
				stat, err := os.Stat(a)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				if stat.IsDir() {
					fsyss = append(fsyss, os.DirFS(fixed))
				} else {
					d, f := path.Split(a)
					fsys, err := NewGlobWalkFS(os.DirFS(strings.TrimSuffix(d, "/")), f)
					if err != nil {
						errs = errors.Join(errs, err)
						continue
					}
					fsyss = append(fsyss, fsys)
				}
			} else {
				if fixed == "" {
					fixed = "."
				}
				fsys, err := NewGlobWalkFS(os.DirFS(fixed), magic)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				fsyss = append(fsyss, fsys)
			}
		}
	}
	if errs != nil {
		return nil, errs
	}
	return fsyss, nil
}

func expandNames(name string) ([]string, error) {
	if HasMagic(name) {
		return filepath.Glob(name)
	}
	return []string{name}, nil
}

// CloseFSs closes each FS that provides a Close() error  interface
func CloseFSs(fsyss []fs.FS) error {
	var errs error
	for _, fsys := range fsyss {
		if closer, ok := fsys.(interface{ Close() error }); ok {
			errs = errors.Join(errs, closer.Close())
		}
	}
	return errs
}
