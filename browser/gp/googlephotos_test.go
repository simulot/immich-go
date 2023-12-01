package gp

import "testing"

func Test_matchEditedName(t *testing.T) {
	tests := []struct {
		jsonName string
		fileName string
		want     bool
	}{
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_090123740.PORTRAIT-modifié.jpg",
			want:     true,
		},
		{
			jsonName: "PXL_20220405_090123740.PORTRAIT.jpg.json",
			fileName: "PXL_20220405_100123740.PORTRAIT-modifié.jpg",
			want:     false,
		},
		{
			jsonName: "DSC_0238.JPG.json",
			fileName: "DSC_0238.JPG",
			want:     true,
		},
		// {
		// 	jsonName: "DSC_0238.JPG.json",
		// 	fileName: "DSC_0238(1).JPG",
		// 	want:     false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			if got := matchEditedName(tt.jsonName, tt.fileName); got != tt.want {
				t.Errorf("matchEditedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			if got := matchVeryLongNameWithNumber(tt.jsonName, tt.fileName); got != tt.want {
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
			if got := matchDuplicateInYear(tt.jsonName, tt.fileName); got != tt.want {
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
			if got := matchForgottenDuplicates(tt.jsonName, tt.fileName); got != tt.want {
				t.Errorf("matchDuplicateInYear() = %v, want %v", got, tt.want)
			}
		})
	}
}
