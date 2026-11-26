package folder

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

func TestArchivedModificationTimes(t *testing.T) {
	capture := time.Date(2011, 6, 15, 9, 0, 0, 0, time.UTC)
	fileDate := capture.Add(48 * time.Hour)
	for _, tc := range []struct {
		name, dir               string
		capture, fileDate, want time.Time
	}{
		{"capture date", "2011/2011-06/", capture, fileDate, capture},
		{"file date", "no-date/", time.Time{}, fileDate, fileDate},
		{"no date", "no-date/", time.Time{}, time.Time{}, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := fstest.MapFS{"photo.jpg": {Data: []byte("photo")}, "photo.xmp": {Data: []byte("xmp")}}
			dst := osfs.DirFS(t.TempDir())
			writer, err := NewLocalAssetWriter(dst, ".")
			if err != nil {
				t.Fatal(err)
			}
			a := &assets.Asset{File: fshelper.FSName(src, "photo.jpg"), NameInfo: assets.NameInfo{Base: "photo.jpg"}, CaptureDate: tc.capture, FileDate: tc.fileDate, FromSideCar: &assets.Metadata{File: fshelper.FSName(src, "photo.xmp")}, FromApplication: &assets.Metadata{Description: "photo"}}
			defer a.Close()
			before := time.Now().Truncate(time.Second)
			if err := writer.WriteAsset(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			for _, ext := range []string{"", ".XMP", ".JSON"} {
				name := tc.dir + "photo.jpg" + ext
				info, err := fs.Stat(dst, name)
				if err != nil {
					t.Fatal(err)
				}
				got := info.ModTime()
				if tc.want.IsZero() {
					if got.Before(before) {
						t.Errorf("%s: expected filesystem timestamp, got %v", name, got)
					}
				} else if !got.Equal(tc.want) {
					t.Errorf("%s: got %v, want %v", name, got, tc.want)
				}
			}
		})
	}
}
