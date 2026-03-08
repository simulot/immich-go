package folder

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/gen"
)

// icloudCSVSuffixPattern matches trailing digits appended directly to a word
// character (letter/underscore) — this is how iCloud splits large CSVs into chunks
// (e.g., "Trip1", "Trip10"). Names where digits follow a space or punctuation
// (e.g., "Summer 2022", "Part 3") are NOT modified.
var icloudCSVSuffixPattern = regexp.MustCompile(`^(.*[a-zA-Z_])\d+$`)

type iCloudMeta struct {
	albums               []assets.Album
	originalCreationDate time.Time
}

// DateCollector is a callback invoked for each date parsed during iCloud CSV processing.
// Callers can use this to track min/max dates or collect active months.
type DateCollector func(t time.Time)

func UseICloudMemory(m *gen.SyncMap[string, iCloudMeta], fsys fs.FS, filename string) (string, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	albumName := "Memory " + stripICloudCSVSuffix(baseName)

	return albumName, useAlbum(m, file, albumName)
}

func UseICloudAlbum(m *gen.SyncMap[string, iCloudMeta], fsys fs.FS, filename string) (string, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	albumName := stripICloudCSVSuffix(baseName)

	return albumName, useAlbum(m, file, albumName)
}

// stripICloudCSVSuffix removes trailing digits that iCloud appends when splitting
// large Memory/Album CSVs into chunks (e.g., "Trip1.csv", "Trip10.csv" are chunks
// of "Trip.csv"). Uses greedy matching so "Summer 2022" stays as "Summer 2022"
// (the digit must follow a non-digit character to be considered a suffix).
func stripICloudCSVSuffix(name string) string {
	if m := icloudCSVSuffixPattern.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return name
}

func useAlbum(m *gen.SyncMap[string, iCloudMeta], file fs.File, albumName string) error {
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.Join(err, errors.New("failed to read all csv records"))
	}
	// icloud takeouts can have empty csv files
	// https://github.com/simulot/immich-go/issues/924
	if len(records) == 0 {
		return nil // nothing to do
	}
	for _, record := range records[1:] {
		if len(record) != 1 {
			return errors.Join(err, errors.New("invalid record"))
		}
		fileName := record[0]
		meta, _ := m.Load(fileName)
		// Deduplicate: only add the album if this file doesn't already have it
		hasAlbum := false
		for _, a := range meta.albums {
			if a.Title == albumName {
				hasAlbum = true
				break
			}
		}
		if !hasAlbum {
			meta.albums = append(meta.albums, assets.Album{Title: albumName})
			m.Store(fileName, meta)
		}
	}

	return nil
}

// Example:
// imgName,fileChecksum,favorite,hidden,deleted,originalCreationDate,viewCount,importDate
// IMG_7938.HEIC,AfQj57ORF2JIumUCjO+PawZ9nqPg,no,no,no,"Saturday June 4,2022 12:11 PM GMT",10,"Saturday June 4,2022 12:11 PM GMT"
func UseICloudPhotoDetails(m *gen.SyncMap[string, iCloudMeta], fsys fs.FS, filename string, collectors ...DateCollector) error {
	file, err := fsys.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.Join(err, errors.New("failed to read all csv records"))
	}
	// icloud takeouts can have empty csv files
	// https://github.com/simulot/immich-go/issues/924
	if len(records) == 0 {
		return nil // nothing to do
	}

	// skip header
	for _, record := range records[1:] {
		if len(record) != 8 {
			return errors.Join(err, errors.New("invalid record"))
		}
		fileName := record[0]
		originalCreationDate := record[5]
		if originalCreationDate == "" || originalCreationDate == "null" {
			// Skip records with missing dates (deleted photos, etc.)
			continue
		}
		t, err := time.Parse("Monday January 2,2006 15:04 PM GMT", originalCreationDate)
		if err != nil {
			return errors.Join(err, errors.New("invalid original creation date"))
		}
		meta, _ := m.Load(fileName)
		meta.originalCreationDate = t
		m.Store(fileName, meta)

		// Notify collectors of parsed date
		for _, collect := range collectors {
			collect(t)
		}
	}

	return nil
}
