package hash

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

// sha1BufPool pools 4MB read buffers to reduce allocations and GC pressure
// during concurrent SHA1 hashing. 4MB matches typical SD card page sizes
// and reduces syscall count from ~640 to ~5 per 20MB RAW file on ARM devices.
var sha1BufPool = sync.Pool{
	New: func() any {
		return bufio.NewReaderSize(nil, 4*1024*1024)
	},
}

func GetSHA1Hash(r io.Reader) ([]byte, error) {
	h := sha1.New()
	// ARM/SD-card fix: use a pooled 4MB buffered reader to batch small reads
	// into large sequential I/O operations. Go's default io.Copy buffer is
	// 32KB, causing ~640 read syscalls per 20MB file. With 4MB buffers this
	// drops to ~5 syscalls, cutting hashing time significantly on SD card storage.
	br := sha1BufPool.Get().(*bufio.Reader)
	br.Reset(r)
	defer sha1BufPool.Put(br)
	if _, err := io.Copy(h, br); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func FileSHA1Hash(fsys fs.FS, filePath string) ([]byte, error) {
	f, err := fsys.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("can't get SHA1: %w", err)
	}
	defer f.Close()
	return GetSHA1Hash(f)
}
