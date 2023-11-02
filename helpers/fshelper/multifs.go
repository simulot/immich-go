package fshelper

import (
	"archive/zip"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yalue/merged_fs"
)

func OpenMultiFile(names ...string) (fs.FS, error) {
	fss := []fs.FS{}

	for _, p := range names {
		s, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if s.IsDir() {
			fsys := DirRemoveFS(p)
			fss = append(fss, fsys)
		} else {
			switch strings.ToLower(filepath.Ext(s.Name())) {
			case ".zip":
				fsys, err := zip.OpenReader(p)
				if err != nil {
					return nil, err
				}
				fss = append(fss, fsys)
			case ".tgz":
				return nil, errors.New("can't read directly .tgz files. Decompress it in a folder, an process the folder")
			default:
				fsys, err := newSingleFileFS(p)
				if err != nil {
					return nil, err
				}
				fss = append(fss, fsys)
			}
		}
	}
	return merged_fs.MergeMultiple(fss...), nil
}
