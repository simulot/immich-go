package filetypes

import (
	"path"
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
	TypeUseless = "useless"
	TypeUnknown = ""
)

var DefaultSupportedMedia = SupportedMedia{
	".3gp": TypeVideo, ".avi": TypeVideo, ".flv": TypeVideo, ".insv": TypeVideo, ".m2ts": TypeVideo, ".m4v": TypeVideo, ".mkv": TypeVideo, ".mov": TypeVideo, ".mp4": TypeVideo, ".mpg": TypeVideo, ".mts": TypeVideo, ".webm": TypeVideo, ".wmv": TypeVideo,
	".3fr": TypeImage, ".ari": TypeImage, ".arw": TypeImage, ".avif": TypeImage, ".bmp": TypeImage, ".cap": TypeImage, ".cin": TypeImage, ".cr2": TypeImage, ".cr3": TypeImage, ".crw": TypeImage, ".dcr": TypeImage, ".dng": TypeImage, ".erf": TypeImage,
	".fff": TypeImage, ".gif": TypeImage, ".heic": TypeImage, ".heif": TypeImage, ".hif": TypeImage, ".iiq": TypeImage, ".insp": TypeImage, ".jpe": TypeImage, ".jpeg": TypeImage, ".jpg": TypeImage,
	".jxl": TypeImage, ".k25": TypeImage, ".kdc": TypeImage, ".mrw": TypeImage, ".nef": TypeImage, ".orf": TypeImage, ".ori": TypeImage, ".pef": TypeImage, ".png": TypeImage, ".psd": TypeImage, ".raf": TypeImage, ".raw": TypeImage, ".rw2": TypeImage,
	".rwl": TypeImage, ".sr2": TypeImage, ".srf": TypeImage, ".srw": TypeImage, ".tif": TypeImage, ".tiff": TypeImage, ".webp": TypeImage, ".x3f": TypeImage,
	".xmp":  TypeSidecar,
	".json": TypeSidecar,
	".mp":   TypeUseless,
}

func (sm SupportedMedia) TypeFromName(name string) string {
	ext := name[strings.LastIndex(name, "."):]
	return sm.TypeFromExt(ext)
}

func (sm SupportedMedia) TypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	if strings.HasPrefix(ext, ".mp~") {
		// #405
		ext = ".mp"
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

// MediaToExtensions defines a map from mediaType to mediaExtensions
// returns the map with the format map[mediatype] = extensions
func MediaToExtensions() map[string][]string {
	reversedMap := make(map[string][]string)

	for ext, mediaType := range DefaultSupportedMedia {
		reversedMap[mediaType] = append(reversedMap[mediaType], ext)
	}

	return reversedMap
}

// rawExtensions defines the supported RAW file extensions
// https://github.com/immich-app/immich/blob/39b571a95c99cbc4183e5d389e6d682cd8e903d9/server/src/utils/mime-types.ts#L1-L55
// source: https://en.wikipedia.org/wiki/Raw_image_format
var rawExtensions = map[string]bool{
	".3fr": true, ".ari": true, ".arw": true, ".cap": true,
	".cin": true, ".cr2": true, ".cr3": true, ".crw": true,
	".dcr": true, ".dng": true, ".erf": true, ".fff": true,
	".iiq": true, ".k25": true, ".kdc": true, ".mrw": true,
	".nef": true, ".nrw": true, ".orf": true, ".ori": true,
	".pef": true, ".psd": true, ".raf": true, ".raw": true,
	".rw2": true, ".rwl": true, ".sr2": true, ".srf": true,
	".srw": true, ".x3f": true,
}

// IsRawFile checks if the given filename has a RAW file extension
func IsRawFile(ext string) bool {
	ext = strings.ToLower(ext)
	return rawExtensions[ext]
}

func (sm SupportedMedia) IsUseLess(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	if sm.IsIgnoredExt(ext) {
		return true
	}

	// MVIMG* is a Google Motion Photo movie part, not useful
	if (ext == "" || sm.TypeFromExt(ext) == TypeVideo) && strings.HasPrefix(strings.ToUpper(name), "MVIMG") {
		return true
	}
	return false
}
