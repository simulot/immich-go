package xmp

import (
	"io"

	"github.com/simulot/immich-go/internal/assets"
)

/*
This package encapsulate the ugly details of writing and reading XMP files
*/

func WriteXMP(a *assets.Asset, w io.Writer) error {
	// Write the XMP data to the writer
	return nil
}

func ReadXMP(a *assets.Asset, r io.Reader) error {
	// Read the XMP data from the reader and return an Asset
	return nil
}
