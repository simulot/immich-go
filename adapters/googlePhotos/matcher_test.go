package gp

import (
	"testing"

	"github.com/simulot/immich-go/internal/filetypes"
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
			want:     "matchFastTrack",
		},
		{
			jsonName: "PXL_20211013_220651983.jpg.json",
			fileName: "PXL_20211013_220651958.jpg",
			want:     "",
		},
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_090123740.PORTRAIT-modifié.jpg",
			want:     "matchEditedName",
		},
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_090123741.PORTRAIT-modifié.jpg",
			want:     "",
		},
		{
			jsonName: "DSC_0100.JPG.json",
			fileName: "DSC_0100.JPG",
			want:     "matchFastTrack",
		},

		{
			jsonName: "DSC_0101.JPG(1).json",
			fileName: "DSC_0101(1).JPG",
			want:     "matchNormal",
		},
		{
			jsonName: "DSC_0102.JPG(2).json",
			fileName: "DSC_0102(1).JPG",
			want:     "",
		},

		{
			jsonName: "DSC_0103.JPG(1).json",
			fileName: "DSC_0103.JPG",
			want:     "",
		},

		{
			jsonName: "DSC_0104.JPG.json",
			fileName: "DSC_0104(1).JPG",
			want:     "",
		},

		{
			jsonName: "IMG_2710.HEIC(1).json",
			fileName: "IMG_2710(1).HEIC",
			want:     "matchNormal",
		},
		{
			jsonName: "PXL_20231118_035751175.MP.jpg.json",
			fileName: "PXL_20231118_035751175.MP.jpg",
			want:     "matchFastTrack",
		},
		{
			jsonName: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json",
			fileName: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg",
			want:     "matchNormal",
		},
		{
			jsonName: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json",
			fileName: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg",
			want:     "matchNormal",
		},

		{
			jsonName: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json",
			fileName: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg",
			want:     "matchNormal",
		},
		{
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_(494).json",
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg",
			want:     "matchNormal",
		},
		{
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_(494).json",
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(185).jpg",
			want:     "",
		},
		{
			jsonName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json",
			fileName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg",
			want:     "matchNormal",
		},

		{
			jsonName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json",
			fileName: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg",
			want:     "matchForgottenDuplicates",
		},
		{ // #405
			jsonName: "PXL_20210102_221126856.MP~2.jpg.json",
			fileName: "PXL_20210102_221126856.MP~2.jpg",
			want:     "matchFastTrack",
		},

		{ //#613
			// 13039_327707840323_537645323_9470255_27214_n.j.json
			// 13039_327707840323_537645323_9470255_27214_n.jpg

			jsonName: "13039_327707840323_537645323_9470255_27214_n.j(1).json",
			fileName: "13039_327707840323_537645323_9470255_27214_n(1).jpg",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "20161105_170829.jpg.supplemental-metadata.json",
			fileName: "20161105_170829.jpg",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "Screenshot_20231027_123303_Facebook.jpg.supple.json",
			fileName: "Screenshot_20231027_123303_Facebook.jpg",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "Screenshot_20231027_123303_Facebook.jpg.supple(1).json",
			fileName: "Screenshot_20231027_123303_Facebook(1).jpg",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "MVIMG_20191230_232926.jpg.supplemental-metadat.json",
			fileName: "MVIMG_20191230_232926.jpg",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "Screenshot_20200301-161151.png.supplemental-me.json",
			fileName: "Screenshot_20200301-161151.png",
			want:     "matchNormal",
		},
		{ //#652 Google Photos: add support for supplemental-metadata.json files #652
			jsonName: "MVIMG_20200207_134534~2.jpg.supplemental-metad.json",
			fileName: "MVIMG_20200207_134534~2.jpg",
			want:     "matchNormal",
		},
		{ //#673 Google photos supplemental metadata files
			fileName: "Scan35(1).jpg",
			jsonName: "Scan35.jpg.supplemental-metadata(1).json",
			want:     "matchNormal",
		},
		{ //#673 Google photos supplemental metadata files
			fileName: "CLIP0001(10).AVI",
			jsonName: "CLIP0001.AVI.supplemental-metadata(10).json",
			want:     "matchNormal",
		},
		{ //#698 Google photos supplemental metadata files
			fileName: "IMAG0061-edited.JPG",
			jsonName: "IMAG0061.JPG.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{ //#674 Google photos supplemental metadata files
			fileName: "IMG-20230325-WA0122~2-edited.jpg",
			jsonName: "IMG-20230325-WA0122~2.jpg.supplemental-metadat.json",
			want:     "matchEditedName",
		},
		{ //#674 Google photos supplemental metadata files
			fileName: "2234089303984509579-edited.jpg",
			jsonName: "2234089303984509579.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{ //#652 Google photos supplemental metadata files
			fileName: "Screenshot_20231027_123303_Facebook-edited.jpg",
			jsonName: "Screenshot_20231027_123303_Facebook.jpg.supple.json",
			want:     "matchEditedName",
		},
		{ //#652 Google photos supplemental metadata files
			fileName: "Screenshot_20231027_123303_Facebook(1).jpg",
			jsonName: "Screenshot_20231027_123303_Facebook.jpg.supple(1).json",
			want:     "matchNormal",
		},
		{
			fileName: "Screenshot_2024-08-31-00-53-48-647_com.snapchat.jpg",
			jsonName: "Screenshot_2024-08-31-00-53-48-647_com.snapcha.json",
			want:     "matchNormal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			matcher := ""
			tmatchers := []struct {
				name string
				fn   matcherFn
			}{
				{name: "matchFastTrack", fn: matchFastTrack},
				{name: "matchNormal", fn: matchNormal},
				{name: "matchForgottenDuplicates", fn: matchForgottenDuplicates},
				{name: "matchEditedName", fn: matchEditedName},
			}

			for _, m := range tmatchers {
				if m.fn(tt.jsonName, tt.fileName, filetypes.DefaultSupportedMedia) {
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
