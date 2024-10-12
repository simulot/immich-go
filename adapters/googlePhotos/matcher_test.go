package gp

import (
	"testing"

	"github.com/simulot/immich-go/internal/metadata"
)

func Test_matchers(t *testing.T) {
	tests := []struct {
		jsonName string
		fileName string
		want     string
	}{
		{
			jsonName: "PXL_20211013_220651983.jpg.json",
			fileName: "PXL_20211013_220651983.jpg",
			want:     "normalMatch",
		},
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_090123740.PORTRAIT-modifié.jpg",
			want:     "matchEditedName",
		},
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_100123740.PORTRAIT-modifié.jpg",
			want:     "",
		},
		{
			jsonName: "DSC_0238.JPG.json",
			fileName: "DSC_0238.JPG",
			want:     "normalMatch",
		},
		{
			jsonName: "DSC_0238.JPG(1).json",
			fileName: "DSC_0238(1).JPG",
			want:     "matchDuplicateInYear",
		},
		{
			jsonName: "IMG_2710.HEIC(1).json",
			fileName: "IMG_2710(1).HEIC",
			want:     "matchDuplicateInYear",
		},
		{
			jsonName: "PXL_20231118_035751175.MP.jpg.json",
			fileName: "PXL_20231118_035751175.MP.jpg",
			want:     "normalMatch",
		},
		{
			jsonName: "PXL_20231118_035751175.MP.jpg.json",
			fileName: "PXL_20231118_035751175.MP",
			want:     "livePhotoMatch",
		},
		{
			jsonName: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json",
			fileName: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg",
			want:     "matchWithOneCharOmitted",
		},
		{
			jsonName: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json",
			fileName: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg",
			want:     "livePhotoMatch",
		},
		{
			jsonName: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json",
			fileName: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg",
			want:     "matchWithOneCharOmitted",
		},
		{
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_(494).json",
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg",
			want:     "matchVeryLongNameWithNumber",
		},
		{
			jsonName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json",
			fileName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg",
			want:     "matchWithOneCharOmitted",
		},
		{
			jsonName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json",
			fileName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg",
			want:     "matchForgottenDuplicates",
		},
		{ // #405
			jsonName: "PXL_20210102_221126856.MP~2.jpg.json",
			fileName: "PXL_20210102_221126856.MP~2",
			want:     "livePhotoMatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			matcher := ""
			for _, m := range matchers {
				if m.fn(tt.jsonName, tt.fileName, metadata.DefaultSupportedMedia) {
					matcher = m.name
					break
				}
			}
			if matcher != tt.want {
				t.Errorf("matcher is '%s', want %v", matcher, tt.want)
			}
		})
	}
}
