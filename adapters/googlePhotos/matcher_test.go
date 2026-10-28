package gp

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filenames"
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

		// Live Photo pairs: Google Takeout only writes a JSON for the still image;
		// the video half must be matched to the same sidecar.
		{ // still image, base pair
			fileName: "IMG_4488.HEIC",
			jsonName: "IMG_4488.HEIC.supplemental-metadata.json",
			want:     "matchNormal",
		},
		{ // video half, base pair, no JSON of its own
			fileName: "IMG_4488.MP4",
			jsonName: "IMG_4488.HEIC.supplemental-metadata.json",
			want:     "matchLivePhotoVideo",
		},
		{ // still image, duplicate index 1
			fileName: "IMG_4488(1).HEIC",
			jsonName: "IMG_4488.HEIC.supplemental-metadata(1).json",
			want:     "matchNormal",
		},
		{ // video half, duplicate index 1, no JSON of its own
			fileName: "IMG_4488(1).MP4",
			jsonName: "IMG_4488.HEIC.supplemental-metadata(1).json",
			want:     "matchLivePhotoVideo",
		},
		{ // still image, duplicate index 2
			fileName: "IMG_4488(2).HEIC",
			jsonName: "IMG_4488.HEIC.supplemental-metadata(2).json",
			want:     "matchNormal",
		},
		{ // video half, duplicate index 2, no JSON of its own
			fileName: "IMG_4488(2).MP4",
			jsonName: "IMG_4488.HEIC.supplemental-metadata(2).json",
			want:     "matchLivePhotoVideo",
		},
		{ // video half must NOT bind to a sidecar of a different duplicate index
			fileName: "IMG_4488(1).MP4",
			jsonName: "IMG_4488.HEIC.supplemental-metadata(2).json",
			want:     "",
		},
		{ // a video with its own dedicated sidecar keeps using it, not the live-photo fallback
			fileName: "IMG_4488.MP4",
			jsonName: "IMG_4488.MP4.supplemental-metadata.json",
			want:     "matchNormal",
		},
		{ // video half where the "(1)" is part of the still image's base name, not a duplicate index
			fileName: "PXL_20220615_101533000(1).MP4",
			jsonName: "PXL_20220615_101533000(1).jpg.supplemental-metadata.json",
			want:     "matchLivePhotoVideo",
		},
		{ // localized suffixes are matched through the sidecar title, not a suffix list
			fileName: "X-bearbeitet.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "X-edytowany.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "X-editado.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "X-bewerkt.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "X-編集済み.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{ // a truncated suffix still matches; a fixed suffix list would miss it
			fileName: "X-edi.jpg",
			jsonName: "X.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},

		// "(n)" that is part of the original name, not a duplicate index
		{
			fileName: "image(2)-edited.jpg",
			jsonName: "image(2).jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "(25219)(3)1368646893405-edited.jpg",
			jsonName: "(25219)(3)1368646893405.jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "HI (3)-edited.jpg",
			jsonName: "HI (3).jpg.supplemental-metadata.json",
			want:     "matchEditedName",
		},
		{
			fileName: "image(2).jpg",
			jsonName: "image(2).jpg.supplemental-metadata.json",
			want:     "matchNormal",
		},
		{ // the literal name must not use the 46-character truncation rule
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg",
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_.json",
			want:     "matchForgottenDuplicates",
		},

		// title without extension
		{
			fileName: "Mail(3).jpg",
			jsonName: "Mail.supplemental-metadata(3).json",
			want:     "matchNormal",
		},
		{
			fileName: "Mail(3).jpg",
			jsonName: "Mail.supplemental-metadata(2).json",
			want:     "",
		},
		{ // an original, not an edited copy
			fileName: "Mail.jpg",
			jsonName: "Mail.supplemental-metadata.json",
			want:     "matchNormal",
		},
		{
			fileName: "s_361ad37c-5556-4bf2-a107-6b579418d6d7-0(1).jpg",
			jsonName: "s_361ad37c-5556-4bf2-a107-6b579418d6d7-0.supplemental-metadata(1).json",
			want:     "matchNormal",
		},
		{
			fileName: "image(2).jpg",
			jsonName: "image(2).supplemental-metadata.json",
			want:     "matchNormal",
		},
		{
			fileName: "Mail-edited.jpg",
			jsonName: "Mail.supplemental-metadata.json",
			want:     "matchEditedName",
		},

		// edited copy of a duplicate
		{
			fileName: "IMG_20131117_164736-edited(1).jpg",
			jsonName: "IMG_20131117_164736.jpg.supplemental-metadata(1).json",
			want:     "matchEditedName",
		},
		{
			fileName: "IMG_20131117_164736-edited(1).jpg",
			jsonName: "IMG_20131117_164736.jpg.supplemental-metadata.json",
			want:     "",
		},
		{
			fileName: "IMG_20131117_164736-edited.jpg",
			jsonName: "IMG_20131117_164736.jpg.supplemental-metadata(1).json",
			want:     "",
		},
		{
			fileName: "s_361ad37c-5556-4bf2-a107-6b579418d6d7-0-edited(1).jpg",
			jsonName: "s_361ad37c-5556-4bf2-a107-6b579418d6d7-0.supplemental-metadata(1).json",
			want:     "matchEditedName",
		},
		{
			fileName: "DSC_0105-edited(1).JPG",
			jsonName: "DSC_0105.JPG(1).json",
			want:     "matchEditedName",
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
				{name: "matchLivePhotoVideo", fn: matchLivePhotoVideo},
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

func Test_getFileIndex(t *testing.T) {
	for _, tc := range []struct {
		name, wantName, wantIndex string
	}{
		{"X(1).jpg", "X.jpg", "1"},
		{"X-edited(1).jpg", "X-edited.jpg", "1"},
		{"X(12)", "X", "12"},
		{"X.jpg(1).json", "X.jpg.json", "1"},
		{"X.jpg.supplemental-metadata(1).json", "X.jpg.supplemental-metadata.json", "1"},
		{"X.jpg", "X.jpg", ""},
		{"image(2)-edited.jpg", "image(2)-edited.jpg", ""},
		{"(25219)(3)1368646893405-edited.jpg", "(25219)(3)1368646893405-edited.jpg", ""},
		{"X(1).jpg.supplemental-metadata.json", "X(1).jpg.supplemental-metadata.json", ""},
		{"X(a).jpg", "X(a).jpg", ""},
		{"X().jpg", "X().jpg", ""},
		{"X).jpg", "X).jpg", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name, index := getFileIndex(tc.name)
			if name != tc.wantName || index != tc.wantIndex {
				t.Errorf("getFileIndex(%q) = (%q, %q), want (%q, %q)", tc.name, name, index, tc.wantName, tc.wantIndex)
			}
		})
	}
}

// The edited flag comes from the matcher that associated the file with the sidecar, not
// from a suffix list: it covers every language and truncated names.
func TestMakeAssetMarksTheEditedFileFromTheMatcher(t *testing.T) {
	toc := &TakeoutCmd{infoCollector: filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)}

	for _, tc := range []struct {
		matcher string
		base    string
		want    bool
	}{
		{"matchEditedName", "X-bearbeitet.jpg", true},
		{"matchEditedName", "X-edi.jpg", true},
		{"matchEditedName", "X-edited(1).jpg", true},
		{"matchNormal", "X(1).jpg", false},
		{"matchLivePhotoVideo", "X.mp4", false},
	} {
		t.Run(tc.matcher+"/"+tc.base, func(t *testing.T) {
			f := &assetFile{base: tc.base, matcher: tc.matcher}
			a := toc.makeAsset(context.Background(), "", f, nil)
			if (a.Kind == assets.KindEdited) != tc.want {
				t.Errorf("%s matched by %s: Kind = %v, want edited = %v", tc.base, tc.matcher, a.Kind, tc.want)
			}
			if tc.want && !a.IsCover {
				t.Errorf("%s matched by %s: not the cover", tc.base, tc.matcher)
			}
		})
	}
}
