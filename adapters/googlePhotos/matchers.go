package gp

import (
	"path"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/simulot/immich-go/internal/filetypes"
)

func matchFastTrack(jsonName string, fileName string, _ filetypes.SupportedMedia) bool {
	//  fast track: if the file name is the same as the JSON name
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	return jsonName == fileName
}

func matchNormal(jsonName string, fileName string, _ filetypes.SupportedMedia) bool {
	// Extract the index from the file name
	fileName, fileIndex := getFileIndex(fileName)
	// Extract the index from the JSON name
	jsonName, jsonIndex := getFileIndex(jsonName)

	// Check if the indexes are the same
	if fileIndex != jsonIndex {
		return false
	}

	// supplemental-metadata  check
	p2 := strings.LastIndex(jsonName, ".")
	if p2 > 1 {
		p1 := strings.LastIndex(jsonName[:p2], ".")
		if p1 > 1 {
			if strings.HasPrefix("supplemental-metadata", jsonName[p1+1:p2]) { //nolint:all
				jsonName = jsonName[:p1] + jsonName[p2:]
			}
		}
	}

	// Check if the file name is the same as the JSON name
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	if jsonName == fileName {
		return true
	}

	if len(fileName) > 46 {
		if utf8.RuneCountInString(fileName) > 46 {
			fileName = string([]rune(fileName)[:46])
			if fileName == jsonName {
				return true
			}
		} else {
			fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
			_, size := utf8.DecodeLastRuneInString(fileName)
			fileName = fileName[:len(fileName)-size]
			if fileName == jsonName {
				return true
			}
		}
	}
	return false
}

// matchEditedName
//   PXL_20220405_090123740.PORTRAIT.jpg.json
//   PXL_20220405_090123740.PORTRAIT.jpg
//   PXL_20220405_090123740.PORTRAIT-modifié.jpg
// but not DSC_0104.JPG.json with DSC_0104(1).JPG

func matchEditedName(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	if _, index := getFileIndex(fileName); index != "" {
		return false
	}
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	p1 := strings.LastIndex(base, ".")
	if p1 > 1 {
		if strings.HasPrefix("supplemental-metadata", base[p1+1:]) { //nolint:all
			base = jsonName[:p1]
		}
	}

	ext := path.Ext(base)
	if ext != "" && sm.IsMedia(ext) {
		base = strings.TrimSuffix(base, ext)
		fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	}
	return strings.HasPrefix(fileName, base)
}

// matchForgottenDuplicates
// "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json"
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg

func matchForgottenDuplicates(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	if strings.HasPrefix(fileName, jsonName) {
		a, b := utf8.RuneCountInString(jsonName), utf8.RuneCountInString(fileName)
		if b-a < 10 {
			return true
		}
	}
	return false
}

// matchLivePhotoVideo
// Google Takeout writes a single supplemental-metadata JSON for a Live Photo
// pair, named after the still image. The video half never gets a sidecar of
// its own, which is most visible in duplicate-indexed sets:
//
//	IMG_4488.HEIC.supplemental-metadata(1).json   (JSON exists only for the still)
//	IMG_4488(1).HEIC                              (still image, matched by matchNormal)
//	IMG_4488(1).MP4                                (video half, no JSON of its own)
//
// It runs right after matchFastTrack/matchNormal, so a video that does have
// its own sidecar is matched there first and never reaches this function.
// It must stay ahead of matchForgottenDuplicates/matchEditedName: those use
// loose prefix matching that can misfire on a plain video/image pair with no
// index (see matchEditedName's own doc comment).
func matchLivePhotoVideo(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	if sm.TypeFromExt(path.Ext(fileName)) != filetypes.TypeVideo {
		return false
	}

	fileName, fileIndex := getFileIndex(fileName)
	jsonName, jsonIndex := getFileIndex(jsonName)
	if fileIndex != jsonIndex {
		return false
	}

	// supplemental-metadata check, same as matchNormal
	p2 := strings.LastIndex(jsonName, ".")
	if p2 > 1 {
		p1 := strings.LastIndex(jsonName[:p2], ".")
		if p1 > 1 {
			if strings.HasPrefix("supplemental-metadata", jsonName[p1+1:p2]) { //nolint:all
				jsonName = jsonName[:p1] + jsonName[p2:]
			}
		}
	}
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName)) // drop ".json", keeps the still image's extension

	// the sidecar must belong to the still-image half of the pair
	jsonExt := path.Ext(jsonName)
	if sm.TypeFromExt(jsonExt) != filetypes.TypeImage {
		return false
	}
	jsonName = strings.TrimSuffix(jsonName, jsonExt)

	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	return jsonName == fileName
}

func getFileIndex(name string) (string, string) {
	// Extract the index from the file name
	p1File := strings.LastIndex(name, "(")
	if p1File >= 0 {
		p2File := strings.LastIndex(name, ")")
		if p2File >= 0 && p2File > p1File {
			fileIndex := name[p1File+1 : p2File]
			if _, err := strconv.Atoi(fileIndex); err == nil {
				return name[:p1File] + name[p2File+1:], fileIndex
			}
		}
	}
	return name, ""
}
