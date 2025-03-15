package folder

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/gen"
)

type iCloudMeta struct {
	albums               []assets.Album
	originalCreationDate time.Time
}

func UseICloudAlbum(m *gen.SyncMap[string, iCloudMeta], fsys fs.FS, filename string) (string, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	albumName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", errors.Join(err, errors.New("failed to read all csv records"))
	}
	for _, record := range records[1:] {
		if len(record) != 1 {
			return "", errors.Join(err, errors.New("invalid record"))
		}
		fileName := record[0]
		meta, _ := m.Load(fileName)
		meta.albums = append(meta.albums, assets.Album{Title: albumName})
		m.Store(fileName, meta)
	}

	return albumName, nil
}

// Example:
// imgName,fileChecksum,favorite,hidden,deleted,originalCreationDate,viewCount,importDate
// IMG_7938.HEIC,AfQj57ORF2JIumUCjO+PawZ9nqPg,no,no,no,"Saturday June 4,2022 12:11 PM GMT",10,"Saturday June 4,2022 12:11 PM GMT"
func UseICloudPhotoDetails(m *gen.SyncMap[string, iCloudMeta], fsys fs.FS, filename string) error {
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
	// skip header
	for _, record := range records[1:] {
		if len(record) != 8 {
			return errors.Join(err, errors.New("invalid record"))
		}
		fileName := record[0]
		originalCreationDate := record[5]
		t, err := time.Parse("Monday January 2,2006 15:04 PM GMT", originalCreationDate)
		if err != nil {
			return errors.Join(err, errors.New("invalid original creation date"))
		}
		meta, _ := m.Load(fileName)
		meta.originalCreationDate = t
		m.Store(fileName, meta)
	}

	return nil
}
