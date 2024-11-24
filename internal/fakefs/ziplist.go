package fakefs

/*
	for f in *.zip; do echo "$f: "; unzip -l $f; done >list.lst
*/
import (
	"archive/zip"
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/gen"
)

// `  2104348  07-20-2023 00:00   Takeout/Google Photos/2020 - Costa Rica/IMG_3235.MP4`

var (
	reZipList  = regexp.MustCompile(`(-rw-r--r-- 0/0\s+)?(\d+)\s+(.{16})\s+(.*)$`)
	reFileLine = regexp.MustCompile(`^(\d+)\s+(\d+)\s+files$`) // 2144740441                     10826 files
)

func readFileLine(l string, dateFormat string) (string, int64, time.Time) {
	if len(l) < 30 {
		return "", 0, time.Time{}
	}
	m := reZipList.FindStringSubmatch(l)
	if len(m) < 5 {
		return "", 0, time.Time{}
	}
	size, _ := strconv.ParseInt(m[2], 10, 64)
	modTime, _ := time.ParseInLocation(dateFormat, m[3], time.Local)
	return m[4], size, modTime
}

func ScanStringList(dateFormat string, s string) ([]fs.FS, error) {
	r := strings.NewReader(s)

	return ScanFileListReader(r, dateFormat)
}

func ScanFileList(name string, dateFormat string) ([]fs.FS, error) {
	var r io.ReadCloser
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	if strings.ToLower(filepath.Ext(name)) == ".zip" {
		i, err := f.Stat()
		if err != nil {
			return nil, err
		}
		z, err := zip.NewReader(f, i.Size())
		if err != nil {
			return nil, err
		}
		if len(z.File) == 0 {
			return nil, errors.New("zip file is empty")
		}
		r, err = z.File[0].Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
	} else {
		r = f
	}

	defer f.Close()
	return ScanFileListReader(r, dateFormat)
}

func ScanFileListReader(f io.Reader, dateFormat string) ([]fs.FS, error) {
	fsyss := map[string]*FakeFS{}
	var fsys *FakeFS
	currentZip := ""
	ok := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "Part:") {
			currentZip = strings.TrimSpace(strings.TrimPrefix(l, "Part:"))
			fsys, ok = fsyss[currentZip]
			if !ok {
				fsys = &FakeFS{
					name:  currentZip,
					files: map[string]map[string]FakeDirEntry{},
				}

				fsyss[currentZip] = fsys
			}
			continue
		}
		if strings.HasPrefix(l, "Archive:") {
			currentZip = strings.TrimSpace(strings.TrimPrefix(l, "Archive:"))
			fsys, ok = fsyss[currentZip]
			if !ok {
				fsys = &FakeFS{
					name:  currentZip,
					files: map[string]map[string]FakeDirEntry{},
				}

				fsyss[currentZip] = fsys
			}
			continue
		}
		if reFileLine.MatchString(l) {
			continue
		}
		if name, size, modTime := readFileLine(l, dateFormat); name != "" {
			fsys.addFile(name, size, modTime)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	names := gen.MapKeys(fsyss)
	sort.Strings(names)
	output := make([]fs.FS, len(fsyss))
	i := 0
	for _, name := range names {
		output[i] = fsyss[name]
		i++
	}
	return output, nil
}
