package nextcloud

import "io/fs"

var (
	_ fs.FS        = (*preferredLocalFS)(nil)
	_ fs.ReadDirFS = (*preferredLocalFS)(nil)
	_ fs.StatFS    = (*preferredLocalFS)(nil)
)

type preferredLocalFS struct {
	local  fs.FS
	remote fs.FS
}

// NewPreferredLocalFS keeps Memories discovery and enumeration on the remote
// filesystem while preferring a local synced copy when opening file contents.
func NewPreferredLocalFS(local fs.FS, remote fs.FS) fs.FS {
	return &preferredLocalFS{local: local, remote: remote}
}

func (p *preferredLocalFS) Open(name string) (fs.File, error) {
	if p.local != nil {
		if f, err := p.local.Open(name); err == nil {
			return f, nil
		}
	}
	return p.remote.Open(name)
}

func (p *preferredLocalFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(p.remote, name)
}

func (p *preferredLocalFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(p.remote, name)
}
