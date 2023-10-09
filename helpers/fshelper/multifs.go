package fshelper

import (
	"archive/zip"
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

		switch {
		case !s.IsDir() && strings.ToLower(filepath.Ext(s.Name())) == ".zip":
			fsys, err := zip.OpenReader(p)
			if err != nil {
				return nil, err
			}
			fss = append(fss, fsys)
		default:
			fsys := DirRemoveFS(p)
			fss = append(fss, fsys)
		}
	}
	return merged_fs.MergeMultiple(fss...), nil
}
