package asciimage

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	_ "golang.org/x/image/webp"
)

// var UnsupportedImageFormat = errors.New("unsupported image format")

func LoadFile(name string) (image.Image, error) {
	r, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return LoadReader(r)
}

func LoadReader(r io.Reader) (image.Image, error) {
	i, _, err := image.Decode(r)
	return i, err
}
