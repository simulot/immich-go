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
