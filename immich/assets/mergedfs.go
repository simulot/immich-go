package assets

import "io/fs"

type MergedFS struct {
	fss []fs.FS
}

func NewMergedFS(fss []fs.FS) *MergedFS {
	return &MergedFS{fss: fss}
}
func (mfs *MergedFS) Open(name string) (file fs.File, err error) {
	for i := range mfs.fss {
		file, err = mfs.fss[i].Open(name)
		if err == nil {
			return
		}
	}
	return
}

func (mfs *MergedFS) Stat(name string) (info fs.FileInfo, err error) {
	for i := range mfs.fss {
		info, err = fs.Stat(mfs.fss[i], name)
		if err == nil {
			return
		}
	}
	return
}

func (mfs *MergedFS) Remove(name string) (err error) {
	for i := range mfs.fss {
		_, err = fs.Stat(mfs.fss[i], name)
		if err == nil {
			return Remove(mfs.fss[i], name)
		}
	}
	return

}

type Remover interface {
	Remove(name string) error
}

func Remove(fsys fs.FS, name string) error {
	if fsys, ok := fsys.(Remover); ok {
		return fsys.Remove(name)
	}
	return nil
}
