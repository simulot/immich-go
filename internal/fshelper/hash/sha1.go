package hash

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/fs"
)

func GetSHA1Hash(r io.Reader) ([]byte, error) {
	h := sha1.New()
	if _, err := io.Copy(h, r); err != nil {
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
