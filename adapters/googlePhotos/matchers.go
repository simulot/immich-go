package gp

import (
	"path"
	"strings"
	"unicode/utf8"

	"github.com/simulot/immich-go/internal/filetypes"
)

// normalMatch
//
//	PXL_20230922_144936660.jpg.json
//	PXL_20230922_144936660.jpg
func normalMatch(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	return base == fileName
}

// livePhotoMatch
// 20231227_152817.jpg.json
// 20231227_152817.MP4
//
// PXL_20231118_035751175.MP.jpg.json
// PXL_20231118_035751175.MP.jpg
// PXL_20231118_035751175.MP
func livePhotoMatch(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	fileExt := path.Ext(fileName)
	fileName = strings.TrimSuffix(fileName, fileExt)
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	base = strings.TrimSuffix(base, path.Ext(base))
	if base == fileName {
		return true
	}
	base = strings.TrimSuffix(base, path.Ext(base))
	return base == fileName
}

// matchWithOneCharOmitted
//
//	PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json
//	PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg
//
//	05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json <-- match also with LivePhoto matcher
//	05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg
//
//  😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json
//  😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg

func matchWithOneCharOmitted(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	baseJSON := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	ext := path.Ext(baseJSON)
	if sm.IsExtensionPrefix(ext) {
		baseJSON = strings.TrimSuffix(baseJSON, ext)
	}
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	if fileName == baseJSON {
		return true
	}
	if strings.HasPrefix(fileName, baseJSON) {
		a, b := utf8.RuneCountInString(fileName), utf8.RuneCountInString(baseJSON)
		if a-b <= 1 {
			return true
		}
	}
	return false
}

// matchVeryLongNameWithNumber
//
//	Backyard_ceremony_wedding_photography_xxxxxxx_(494).json
//	Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg
func matchVeryLongNameWithNumber(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))

	p1JSON := strings.Index(jsonName, "(")
	if p1JSON < 0 {
		return false
	}
	p2JSON := strings.Index(jsonName, ")")
	if p2JSON < 0 || p2JSON != len(jsonName)-1 {
		return false
	}
	p1File := strings.Index(fileName, "(")
	if p1File < 0 || p1File != p1JSON+1 {
		return false
	}
	if jsonName[:p1JSON] != fileName[:p1JSON] {
		return false
	}
	p2File := strings.Index(fileName, ")")
	return jsonName[p1JSON+1:p2JSON] == fileName[p1File+1:p2File]
}

// matchDuplicateInYear
//
//	IMG_3479.JPG(2).json
//	IMG_3479(2).JPG
//

// Fast implementation, but does't work with live photos
func matchDuplicateInYear(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	p1JSON := strings.Index(jsonName, "(")
	if p1JSON < 1 {
		return false
	}
	p1File := strings.Index(fileName, "(")
	if p1File < 0 {
		return false
	}
	jsonExt := path.Ext(jsonName[:p1JSON])

	p2JSON := strings.Index(jsonName, ")")
	if p2JSON < 0 || p2JSON != len(jsonName)-1 {
		return false
	}

	p2File := strings.Index(fileName, ")")
	if p2File < 0 || p2File < p1File {
		return false
	}

	fileExt := path.Ext(fileName)

	if fileExt != jsonExt {
		return false
	}

	jsonBase := strings.TrimSuffix(jsonName[:p1JSON], path.Ext(jsonName[:p1JSON]))

	if jsonBase != fileName[:p1File] {
		return false
	}

	if fileName[p1File+1:p2File] != jsonName[p1JSON+1:p2JSON] {
		return false
	}

	return true
}

/*
// Regexp implementation, work with live photos, 10 times slower
var (
	reDupInYearJSON = regexp.MustCompile(`(.*)\.(.{2,4})\((\d+)\)\..{2,4}$`)
	reDupInYearFile = regexp.MustCompile(`(.*)\((\d+)\)\..{2,4}$`)
)

func matchDuplicateInYear(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	mFile := reDupInYearFile.FindStringSubmatch(fileName)
	if len(mFile) < 3 {
		return false
	}
	mJSON := reDupInYearJSON.FindStringSubmatch(jsonName)
	if len(mJSON) < 4 {
		return false
	}
	if mFile[1] == mJSON[1] && mFile[2] == mJSON[3] {
		return true
	}
	return false
}
*/

// matchEditedName
//   PXL_20220405_090123740.PORTRAIT.jpg.json
//   PXL_20220405_090123740.PORTRAIT.jpg
//   PXL_20220405_090123740.PORTRAIT-modifié.jpg

func matchEditedName(jsonName string, fileName string, sm filetypes.SupportedMedia) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	ext := path.Ext(base)
	if ext != "" && sm.IsMedia(ext) {
		base = strings.TrimSuffix(base, ext)
		fname := strings.TrimSuffix(fileName, path.Ext(fileName))
		return strings.HasPrefix(fname, base)
	}
	return false
}

// TODO: This one interferes with matchVeryLongNameWithNumber

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
