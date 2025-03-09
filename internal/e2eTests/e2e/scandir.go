package e2e

import (
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/fshelper/hash"
)

type FileInfo struct {
	Size int
	SHA1 string
}

func ExtensionFilter(seq map[string]FileInfo, ext []string) map[string]FileInfo {
	for i := range ext {
		ext[i] = strings.ToLower(ext[i])
	}
	r := map[string]FileInfo{}
	for k, v := range seq {
		for _, e := range ext {
			if strings.HasSuffix(strings.ToLower(k), e) {
				r[k] = v
				break
			}
		}
	}
	return r
}

func BaseNameFilter(seq map[string]FileInfo) map[string]FileInfo {
	r := map[string]FileInfo{}
	for k, v := range seq {
		k = path.Base(k)
		r[k] = v
	}
	return r
}

func ScanDirectory(t *testing.T, dir string) map[string]FileInfo {
	entries := make(map[string]FileInfo)
	dirfs := os.DirFS(dir)
	err := fs.WalkDir(dirfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		SHA1, err := hash.Base64Encode(hash.FileSHA1Hash(dirfs, path))
		if err != nil {
			return err
		}
		entries[path] = FileInfo{
			Size: int(info.Size()),
			SHA1: SHA1,
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
