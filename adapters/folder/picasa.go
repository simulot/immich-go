package folder

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

type PicasaAlbum struct {
	Name        string
	Description string
}

func ReadPicasaIni(fsys fs.FS, filename string) (PicasaAlbum, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return PicasaAlbum{}, err
	}
	defer file.Close()
	a, err := parsePicasaIni(file)
	if err != nil {
		return PicasaAlbum{}, fmt.Errorf("error parsing picasa ini file: %w", err)
	}
	return a, nil
}

// parsePicasaIni parses the content of an INI file.
func parsePicasaIni(r io.Reader) (PicasaAlbum, error) {
	scanner := bufio.NewScanner(r)
	var currentSection string
	var Album PicasaAlbum

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			// Skip empty lines and comments
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// New section
			currentSection = line[1 : len(line)-1]
		} else if currentSection == "Picasa" {
			// Key-value pair
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				return PicasaAlbum{}, errors.New("invalid line: " + line)
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			switch key {
			case "name":
				Album.Name = value
			case "description":
				Album.Description = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return PicasaAlbum{}, err
	}

	return Album, nil
}
