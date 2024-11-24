package fakefs

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/gen"
)

/*
	simulate a file system based on the list of files contained into a set of archive.

*/

type FakeDirEntry struct {
	name    string      //  name of the file
	size    int64       // length in bytes for regular files; system-dependent for others
	mode    fs.FileMode // file mode bits
	modTime time.Time   // modification time
}

func (fi FakeDirEntry) Name() string               { return path.Base(fi.name) }
func (fi FakeDirEntry) Size() int64                { return fi.size }
func (fi FakeDirEntry) Mode() fs.FileMode          { return fi.mode }
func (fi FakeDirEntry) ModTime() time.Time         { return fi.modTime }
func (fi FakeDirEntry) IsDir() bool                { return fi.mode.IsDir() }
func (fi FakeDirEntry) Sys() any                   { return nil }
func (fi FakeDirEntry) Type() fs.FileMode          { return fi.mode }
func (fi FakeDirEntry) Info() (fs.FileInfo, error) { return fi, nil }

type FakeFile struct {
	fi  FakeDirEntry
	r   io.Reader
	pos int64
}

func (f FakeFile) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

func (f *FakeFile) Read(b []byte) (int, error) {
	if f.pos < f.fi.size {
		n, err := f.r.Read(b)
		f.pos += int64(n)
		return n, err
	}
	return 0, io.EOF
}

func (f *FakeFile) Close() error {
	f.pos = 0
	return nil
}

type FakeFS struct {
	name  string
	files map[string]map[string]FakeDirEntry
}

func (fsys FakeFS) Name() string {
	return fsys.name
}

func normalizeName(name string) string {
	if name != "." && !strings.HasPrefix(name, "./") {
		return "./" + name
	}
	return name
}

func (fsys FakeFS) Stat(name string) (fs.FileInfo, error) {
	name = normalizeName(name)
	name = filepath.ToSlash(name)
	dir, base := path.Split(name)
	dir = strings.TrimSuffix(dir, "/")
	var l map[string]FakeDirEntry
	if dir == "" {
		dir = "."
	}
	l = fsys.files[dir]
	if len(l) == 0 {
		return nil, fmt.Errorf("%s:%s: %w", fsys.name, name, fs.ErrNotExist)
	}
	if e, ok := l[base]; ok {
		return e, nil
	}
	return nil, fs.ErrNotExist
}

func (fsys FakeFS) Open(name string) (fs.File, error) {
	name = normalizeName(name)
	info, err := fsys.Stat(name)
	if err != nil {
		return nil, err
	}

	fakeInfo := info.(FakeDirEntry)
	var r io.Reader

	ext := path.Ext(name)
	if strings.ToLower(ext) == ".json" {
		base := path.Base(name)
		switch base {
		case "métadonnées.json", "metadata.json", "metadati.json", "metadáta.json", "Metadaten.json":
			album := path.Base(path.Dir(name))
			r, fakeInfo.size = fakeAlbumData(album)
		case "print-subscriptions.json", "shared_album_comments.json", "user-generated-memory-titles.json":
			r, fakeInfo.size = fakeJSON()
		default:
			d := info.ModTime()
			if d2 := filenames.TakeTimeFromName(name, time.Local); !d2.IsZero() {
				d = d2
			}
			title := strings.TrimSuffix(path.Base(name), path.Ext(base))
			r, fakeInfo.size = fakePhotoData(title, d)
		}
	} else {
		r = rand.Reader
	}
	return &FakeFile{fi: fakeInfo, r: r}, nil
}

func (fsys FakeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = normalizeName(name)
	info, err := fsys.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fs.ErrNotExist
	}

	entries := fsys.files[name]
	if len(entries) == 0 {
		return nil, fs.ErrNotExist
	}

	keys := gen.MapKeys(entries)
	sort.Strings(keys)
	out := []fs.DirEntry{}
	for _, k := range keys {
		if k != "." {
			out = append(out, entries[k])
		}
	}
	return out, nil
}

func (fsys FakeFS) addFile(name string, size int64, modDate time.Time) {
	name = normalizeName(name)
	dir, base := path.Split(name)
	dir = strings.TrimSuffix(dir, "/")
	parts := strings.Split(dir, "/")

	for i, p := range parts {
		// create the entry in the parent
		if i == 0 {
			sub := "."
			if _, ok := fsys.files[sub]; !ok {
				//
				e := FakeDirEntry{
					name:    ".",
					modTime: time.Now(),
					size:    0,
					mode:    0o777 | fs.ModeDir,
				}

				fsys.files[sub] = map[string]FakeDirEntry{
					".": e,
				}
			}
		} else {
			// add entry in the parent
			parent := strings.Join(parts[:i], "/")
			dir := parent + "/" + p
			if _, ok := fsys.files[parent][p]; !ok {
				fsys.files[parent][p] = FakeDirEntry{
					name:    dir,
					modTime: time.Now(),
					size:    0,
					mode:    0o777 | fs.ModeDir,
				}
			}
			// create the dir entry
			if _, ok := fsys.files[dir]; !ok {
				fsys.files[dir] = map[string]FakeDirEntry{
					".": {
						name:    dir + "/.",
						modTime: time.Now(),
						size:    0,
						mode:    0o777 | fs.ModeDir,
					},
				}
			}
		}
	}
	l := fsys.files[dir]
	l[base] = FakeDirEntry{
		name:    name,
		modTime: modDate,
		size:    size,
		mode:    0o777,
	}
}
