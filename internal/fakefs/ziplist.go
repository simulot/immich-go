package fakefs

/*
	for f in *.zip; do echo "$f: "; unzip -l $f; done >list.lst
*/
import (
	"bufio"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"
)

func readFileLine(l string, dateFormat string) (string, int64, time.Time) {
	if len(l) < 30 {
		return "", 0, time.Time{}
	}
	// `  2104348  07-20-2023 00:00   Takeout/Google Photos/2020 - Costa Rica/IMG_3235.MP4`
	s := strings.TrimSpace(l[:9])
	d := l[11:27]
	name := l[30:]
	size, _ := strconv.ParseInt(s, 10, 64)
	modTime, _ := time.ParseInLocation(dateFormat, d, time.Local)
	return name, size, modTime
}

func ScanStringList(dateFormat string, s string) ([]fs.FS, error) {
	r := strings.NewReader(s)

	return ScanFileListReader(r, dateFormat)
}

func ScanFileList(name string, dateFormat string) ([]fs.FS, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ScanFileListReader(f, dateFormat)
}

func ScanFileListReader(f io.Reader, dateFormat string) ([]fs.FS, error) {
	fsyss := map[string]*FakeFS{}
	var fsys *FakeFS
	currentZip := ""
	inList := false
	ok := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "Archive:  ") {
			currentZip = strings.TrimPrefix(l, "Archive:  ")
			fsys, ok = fsyss[currentZip]
			if !ok {
				fsys = &FakeFS{
					name:  currentZip,
					files: map[string]map[string]FakeDirEntry{},
				}

				fsyss[currentZip] = fsys
			}
			scanner.Scan()
			scanner.Scan()
			inList = true
			continue
		}
		if strings.HasPrefix(l, "--------- ") {
			scanner.Scan()
			inList = false
			continue
		}
		if inList {
			if name, size, modTime := readFileLine(l, dateFormat); name != "" {
				fsys.addFile(name, size, modTime)
			} else {
				inList = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	output := make([]fs.FS, len(fsyss))
	i := 0
	for _, fs := range fsyss {
		output[i] = fs
		i++
	}
	return output, nil
}
