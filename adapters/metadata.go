package adapters

import (
	"io"
	"path/filepath"
	"strings"
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/metadata"
)

type ReadMetadataOptions struct {
	ExifTool         *metadata.ExifTool
	ExiftoolTimezone *time.Location
	FilenameTimeZone *time.Location
}

func (la *LocalAssetFile) ReadMetadata(method cliflags.DateMethod, options ReadMetadataOptions) error {
	ms := strings.Split(string(method), "-")
	for _, m := range ms {
		switch cliflags.DateMethod(m) {
		case cliflags.DateMethodNone:
			return nil
		case cliflags.DateMethodEXIF:
			if options.ExifTool != nil {
				err := la.metadataFromExiftool(options)
				if err != nil {
					continue
				}
				if !la.Metadata.DateTaken.IsZero() {
					return nil
				}
			} else {
				err := la.metadataFromDirectRead(options.ExiftoolTimezone)
				if err != nil {
					continue
				}
				if !la.Metadata.DateTaken.IsZero() {
					return nil
				}
			}
		case cliflags.DateMethodName:
			t := metadata.TakeTimeFromPath(la.FileName, options.FilenameTimeZone)
			if !t.IsZero() {
				la.Metadata.DateTaken = t
				return nil
			}
		}
	}
	return nil
}

// metadataFromExiftool call exiftool
func (la *LocalAssetFile) metadataFromExiftool(options ReadMetadataOptions) error {
	// Get a handler on a temporary file
	r, err := la.PartialSourceReader()
	if err != nil {
		return err
	}

	// be sure the file is completely extracted in the temp file
	_, err = io.Copy(io.Discard, r)
	if err != nil {
		return err
	}

	md, err := options.ExifTool.ReadMetaData(la.tempFile.Name())
	if err != nil {
		return err
	}
	la.Metadata = *md
	return nil
}

func (la *LocalAssetFile) metadataFromDirectRead(localTZ *time.Location) error {
	// Get a handler on a temporary file
	r, err := la.PartialSourceReader()
	if err != nil {
		return err
	}

	ext := filepath.Ext(la.FileName)
	md, err := metadata.GetFromReader(r, ext, localTZ)
	if err != nil {
		return err
	}
	la.Metadata = *md
	return nil
}
