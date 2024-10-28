package adapters

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/metadata"
)

type ReadMetadataOptions struct {
	ExifTool         *metadata.ExifTool
	ExiftoolTimezone *time.Location
	FilenameTimeZone *time.Location
}

func (la *Asset) ReadMetadata(method cliflags.DateMethod, options ReadMetadataOptions) (*metadata.Metadata, error) {
	ms := strings.Split(string(method), "-")
	for _, m := range ms {
		switch cliflags.DateMethod(m) {
		case cliflags.DateMethodNone:
			return nil, nil
		case cliflags.DateMethodEXIF:
			if options.ExifTool != nil {
				md, err := la.metadataFromExiftool(options)
				if err != nil {
					continue
				}
				if !md.DateTaken.IsZero() {
					return md, nil
				}
			} else {
				md, err := la.metadataFromDirectRead(options.ExiftoolTimezone)
				if err != nil {
					continue
				}
				if !md.DateTaken.IsZero() {
					return md, nil
				}
			}
		case cliflags.DateMethodName:
			t := filenames.TakeTimeFromPath(la.FileName, options.FilenameTimeZone)
			if !t.IsZero() {
				return &metadata.Metadata{
					DateTaken: t,
				}, nil
			}
		}
	}
	return nil, nil
}

// metadataFromExiftool call exiftool
func (la *Asset) metadataFromExiftool(options ReadMetadataOptions) (*metadata.Metadata, error) {
	// Get a handler on a temporary file
	r, err := la.PartialSourceReader()
	if err != nil {
		return nil, err
	}

	// be sure the file is completely extracted in the temp file
	_, err = io.Copy(io.Discard, r)
	if err != nil {
		return nil, err
	}

	md, err := options.ExifTool.ReadMetaData(la.tempFile.Name())
	return md, err
}

func (la *Asset) metadataFromDirectRead(localTZ *time.Location) (*metadata.Metadata, error) {
	// Get a handler on a temporary file
	r, err := la.PartialSourceReader()
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(la.FileName)
	md, err := metadata.GetFromReader(r, ext, localTZ)
	return md, err
}
