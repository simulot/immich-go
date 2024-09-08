package metadata

import (
	"slices"
	"sort"
	"strings"
	"sync"
)

type SupportedMedia map[string]string

const (
	TypeVideo   = "video"
	TypeImage   = "image"
	TypeSidecar = "sidecar"
	TypeUnknown = ""
)

var DefaultSupportedMedia = SupportedMedia{
	".3gp": TypeVideo, ".avi": TypeVideo, ".flv": TypeVideo, ".insv": TypeVideo, ".m2ts": TypeVideo, ".m4v": TypeVideo, ".mkv": TypeVideo, ".mov": TypeVideo, ".mp4": TypeVideo, ".mpg": TypeVideo, ".mts": TypeVideo, ".webm": TypeVideo, ".wmv": TypeVideo,
	".3fr": TypeImage, ".ari": TypeImage, ".arw": TypeImage, ".avif": TypeImage, ".bmp": TypeImage, ".cap": TypeImage, ".cin": TypeImage, ".cr2": TypeImage, ".cr3": TypeImage, ".crw": TypeImage, ".dcr": TypeImage, ".dng": TypeImage, ".erf": TypeImage,
	".fff": TypeImage, ".gif": TypeImage, ".heic": TypeImage, ".heif": TypeImage, ".hif": TypeImage, ".iiq": TypeImage, ".insp": TypeImage, ".jpe": TypeImage, ".jpeg": TypeImage, ".jpg": TypeImage,
	".jxl": TypeImage, ".k25": TypeImage, ".kdc": TypeImage, ".mrw": TypeImage, ".nef": TypeImage, ".orf": TypeImage, ".ori": TypeImage, ".pef": TypeImage, ".png": TypeImage, ".psd": TypeImage, ".raf": TypeImage, ".raw": TypeImage, ".rw2": TypeImage,
	".rwl": TypeImage, ".sr2": TypeImage, ".srf": TypeImage, ".srw": TypeImage, ".tif": TypeImage, ".tiff": TypeImage, ".webp": TypeImage, ".x3f": TypeImage,
	".xmp": TypeSidecar,
	".mp":  TypeVideo,
}

func (sm SupportedMedia) TypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	if strings.HasPrefix(ext, ".mp~") {
		// #405
		ext = ".mp4"
	}
	return sm[ext]
}

func (sm SupportedMedia) IsMedia(ext string) bool {
	t := sm.TypeFromExt(ext)
	return t == TypeVideo || t == TypeImage
}

var (
	_supportedExtension    []string
	initSupportedExtension sync.Once
)

func (sm SupportedMedia) IsExtensionPrefix(ext string) bool {
	initSupportedExtension.Do(func() {
		_supportedExtension = make([]string, len(sm))
		i := 0
		for k := range sm {
			_supportedExtension[i] = k[:len(k)-2]
			i++
		}
		sort.Strings(_supportedExtension)
	})
	ext = strings.ToLower(ext)
	_, b := slices.BinarySearch(_supportedExtension, ext)
	return b
}

func (sm SupportedMedia) IsIgnoredExt(ext string) bool {
	t := sm.TypeFromExt(ext)
	return t == ""
}
