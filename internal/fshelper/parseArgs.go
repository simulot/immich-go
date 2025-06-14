package fshelper

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	zipname "github.com/simulot/immich-go/internal/fshelper/zipName"
)

// ParsePath return a list of FS bases on args
//
// Zip files are opened and returned as FS
// Manage wildcards in path
//
// TODO: Implement a tgz reader for non google-photos archives

func ParsePath(args []string) ([]fs.FS, error) {
	var errs error
	fsyss := []fs.FS{}

	for _, a := range args {
		a = filepath.ToSlash(a)
		files, err := expandNames(a)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			lowF := strings.ToLower(f)
			switch {
			case strings.HasSuffix(lowF, ".tgz") || strings.HasSuffix(lowF, ".tar.gz"):
				errs = errors.Join(fmt.Errorf("immich-go can't use tgz archives: %s", filepath.Base(a)))
			case strings.HasSuffix(lowF, ".zip"):
				fsys, err := zipname.OpenReader(f) //   zip.OpenReader(f)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("%s: %w", a, err))
					continue
				}
				fsyss = append(fsyss, fsys)
			default:
				fsys, err := NewVirtualGlobFS(f)
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
