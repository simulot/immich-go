package gp

import (
	"testing"

	"github.com/simulot/immich-go/internal/metadata"
)

func Test_matchVeryLongNameWithNumber(t *testing.T) {
	tests := []struct {
		jsonName string
		fileName string
		want     bool
	}{
		{
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_(494).json",
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg",
			want:     true,
		},
		{
			jsonName: "Backyard_ceremony_wedding_photography_xxxxxxx_(494).json",
			fileName: "Backyard_ceremony_wedding_photography_xxxxxxx_m(185).jpg",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			if got := matchVeryLongNameWithNumber(tt.jsonName, tt.fileName, metadata.DefaultSupportedMedia); got != tt.want {
				t.Errorf("matchVeryLongNameWithNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_matchDuplicateInYear(t *testing.T) {
	tests := []struct {
		name     string
		jsonName string
		fileName string
		want     bool
	}{
		{
			name:     "match",
			jsonName: "IMG_3479.JPG(2).json",
			fileName: "IMG_3479(2).JPG",
			want:     true,
		},
		{
			name:     "doesn't match",
			jsonName: "IMG_3479.JPG(2).json",
			fileName: "IMG_3479(3).JPG",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchDuplicateInYear(tt.jsonName, tt.fileName, metadata.DefaultSupportedMedia); got != tt.want {
				t.Errorf("matchDuplicateInYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_matchForgottenDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		jsonName string
		fileName string
		want     bool
	}{
		{
			name:     "match1",
			jsonName: "1556189729458-8d2e2d13-bca5-467e-a242-9e4cb238.json",
			fileName: "1556189729458-8d2e2d13-bca5-467e-a242-9e4cb238e.jpg",
			want:     true,
		},
		{
			name:     "match2",
			jsonName: "1556189729458-8d2e2d13-bca5-467e-a242-9e4cb238.json",
			fileName: "1556189729458-8d2e2d13-bca5-467e-a242-9e4cb238e(1).jpg",
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchForgottenDuplicates(tt.jsonName, tt.fileName, metadata.DefaultSupportedMedia); got != tt.want {
				t.Errorf("matchDuplicateInYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

/* indexes, but false
goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/browser/gp
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_matchDuplicateInYear-12    	27067428	        52.06 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/simulot/immich-go/browser/gp	1.458s

goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/browser/gp
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_matchDuplicateInYear-12    	  881652	      1491 ns/op	     240 B/op	       4 allocs/op
PASS
ok  	github.com/simulot/immich-go/browser/gp	1.332s


goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/browser/gp
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_matchDuplicateInYear-12    	25737067	        43.88 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/simulot/immich-go/browser/gp	1.180s

*/

func Benchmark_matchDuplicateInYear(b *testing.B) {
	for i := 0; i < b.N; i++ {
		matchDuplicateInYear("IMG_3479.JPG(2).json", "IMG_3479(2).JPG", nil)
	}
}

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
