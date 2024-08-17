package picasa

import (
	"bufio"
	"errors"
	"github.com/spf13/afero"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type DirectoryData struct {
	Name        string
	Description string
	Location    string
	Files       map[string]FileData
	Albums      map[string]AlbumData
}
type AlbumData struct {
	Name        string
	Description string
	Location    string
}
type FileData struct {
	IsStar  bool
	Caption string
	Albums  []string
}

var appFS = afero.NewOsFs()

var DirectoryCache = map[string]DirectoryData{}

func CacheDirectory(dir string) {
	DirectoryCache[dir] = ParseDirectory(dir)
}

func HasPicasa(dir string) bool {
	fileName := path.Join(dir, ".picasa.ini")
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

func ParseDirectory(dir string) DirectoryData {
	directoryData := DirectoryData{
		Files:  map[string]FileData{},
		Albums: map[string]AlbumData{},
	}

	iniMap := parseFile(filepath.Join(dir, ".picasa.ini"))
	for sectionName, pairs := range iniMap {
		if sectionName == "Picasa" {
			if value, ok := pairs["name"]; ok {
				directoryData.Name = value
			}
			if value, ok := pairs["description"]; ok {
				directoryData.Description = value
			}
			if value, ok := pairs["location"]; ok {
				directoryData.Location = value
			}
		} else if strings.HasPrefix(sectionName, ".album:") {
			albumData := AlbumData{}
			token := sectionName[7:]
			if value, ok := pairs["name"]; ok {
				albumData.Name = value
			}
			if value, ok := pairs["description"]; ok {
				albumData.Description = value
			}
			if value, ok := pairs["location"]; ok {
				albumData.Location = value
			}
			directoryData.Albums[token] = albumData
		} else {
			fileData := FileData{}
			if value, ok := pairs["star"]; ok {
				fileData.IsStar = value == "yes"
			}
			if value, ok := pairs["caption"]; ok {
				fileData.Caption = value
			}
			if value, ok := pairs["albums"]; ok {
				fileData.Albums = strings.Split(value, ",")
			}
			directoryData.Files[sectionName] = fileData
		}
	}

	return directoryData
}

func parseFile(path string) (result map[string]map[string]string) {
	readFile, err := appFS.Open(path)
	if err != nil {
		slog.Error("could not open file", err)
		return
	}
	defer func() {
		err = readFile.Close()
		if err != nil {
			slog.Error("could not close file", err)
		}
	}()

	fileScanner := bufio.NewScanner(readFile)
	return parseScanner(fileScanner)
}

func parseScanner(fileScanner *bufio.Scanner) map[string]map[string]string {
	result := map[string]map[string]string{}
	fileScanner.Split(bufio.ScanLines)

	section := ""
	for fileScanner.Scan() {
		line := fileScanner.Text()

		if len(line) == 0 {
			continue
		}

		if line[0:1] == "[" {
			end := strings.Index(line, "]")
			section = line[1:end]
			if _, ok := result[section]; !ok {
				result[section] = map[string]string{}
			} else {
				panic("unexpected duplicate picasa.ini section. malformed .picasa.ini file?")
			}

		}
		if section != "" {
			if eqPos := strings.Index(line, "="); eqPos > 0 {
				key := line[0:eqPos]
				value := line[eqPos+1:]
				result[section][key] = value
			}
		}
	}

	return result
}

func (p DirectoryData) DescriptionAndLocation() string {
	result := p.Description
	if strings.TrimSpace(p.Location) != "" {
		result += "  Location: " + strings.TrimSpace(p.Location)
	}
	return result
}
