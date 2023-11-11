package fshelper

import (
	"archive/zip"
	"io/fs"

	"github.com/yalue/merged_fs"
)

func multiZip(names ...string) (fs.FS, error) {
	fss := []fs.FS{}

	for _, p := range names {
		fsys, err := zip.OpenReader(p)
		if err != nil {
			return nil, err
		}
		fss = append(fss, fsys)
	}
	return merged_fs.MergeMultiple(fss...), nil
}
