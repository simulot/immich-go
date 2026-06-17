package assets

import (
	"bytes"
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
)

type countingFS struct {
	data      []byte
	openCount int
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.openCount++
	return &countingFile{
		Reader: bytes.NewReader(c.data),
		info: countingFileInfo{
			name:    name,
			size:    int64(len(c.data)),
			modTime: time.Unix(0, 0),
		},
	}, nil
}

type countingFile struct {
	*bytes.Reader
	info countingFileInfo
}

func (c *countingFile) Stat() (fs.FileInfo, error) {
	return c.info, nil
}

func (c *countingFile) Close() error {
	return nil
}

type countingFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (c countingFileInfo) Name() string       { return c.name }
func (c countingFileInfo) Size() int64        { return c.size }
func (c countingFileInfo) Mode() fs.FileMode  { return 0 }
func (c countingFileInfo) ModTime() time.Time { return c.modTime }
func (c countingFileInfo) IsDir() bool        { return false }
func (c countingFileInfo) Sys() any           { return nil }

func TestGetChecksumThenOpenFileUsesSingleSourceRead(t *testing.T) {
	t.Parallel()

	source := &countingFS{data: []byte("nextcloud source bytes")}
	a := &Asset{
		File:             fshelper.FSName(source, "photo.jpg"),
		OriginalFileName: "photo.jpg",
		FileSize:         len(source.data),
	}
	t.Cleanup(func() {
		_ = a.Close()
	})

	checksum, err := a.GetChecksum()
	if err != nil {
		t.Fatalf("GetChecksum() error = %v", err)
	}
	if checksum == "" {
		t.Fatal("GetChecksum() returned an empty checksum")
	}

	f, err := a.OpenFile()
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != string(source.data) {
		t.Fatalf("OpenFile() content = %q, want %q", string(content), string(source.data))
	}

	if source.openCount != 1 {
		t.Fatalf("source open count = %d, want 1", source.openCount)
	}

	checksum2, err := a.GetChecksum()
	if err != nil {
		t.Fatalf("second GetChecksum() error = %v", err)
	}
	if checksum2 != checksum {
		t.Fatalf("second GetChecksum() = %q, want %q", checksum2, checksum)
	}
	if source.openCount != 1 {
		t.Fatalf("source open count after second checksum = %d, want 1", source.openCount)
	}
}

func TestOpenFileThenGetChecksumUsesCachedCopy(t *testing.T) {
	t.Parallel()

	source := &countingFS{data: []byte("cached source bytes")}
	a := &Asset{
		File:             fshelper.FSName(source, "video.mp4"),
		OriginalFileName: "video.mp4",
		FileSize:         len(source.data),
	}
	t.Cleanup(func() {
		_ = a.Close()
	})

	f, err := a.OpenFile()
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != string(source.data) {
		t.Fatalf("OpenFile() content = %q, want %q", string(content), string(source.data))
	}

	checksum, err := a.GetChecksum()
	if err != nil {
		t.Fatalf("GetChecksum() error = %v", err)
	}
	if checksum == "" {
		t.Fatal("GetChecksum() returned an empty checksum")
	}

	if source.openCount != 1 {
		t.Fatalf("source open count = %d, want 1", source.openCount)
	}
}
