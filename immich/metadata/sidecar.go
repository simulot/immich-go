package metadata

import (
	"io"
	"io/fs"
)

type SideCarFile struct {
	FSys     fs.FS
	FileName string
}

func (m SideCarFile) Write(w io.Writer) error {
	f, err := m.FSys.Open(m.FileName)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func (m *SideCarFile) IsSet() bool {
	if m == nil {
		return false
	}
	return m.FSys != nil && m.FileName != ""
}
