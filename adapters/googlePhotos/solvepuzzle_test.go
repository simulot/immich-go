package gp

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
)

func TestSolvePuzzleMatchesAnEditToTheLongestSidecar(t *testing.T) {
	for _, tc := range []struct {
		name   string
		jsons  []string
		files  []string
		edited string
		want   string
	}{
		{
			name:   "numbered sibling",
			jsons:  []string{"IMG_0001.jpg.supplemental-metadata.json", "IMG_0001_1.jpg.supplemental-metadata.json"},
			files:  []string{"IMG_0001.jpg", "IMG_0001_1.jpg"},
			edited: "IMG_0001_1-edited.jpg",
			want:   "IMG_0001_1.jpg",
		},
		{
			name:   "effects sibling",
			jsons:  []string{"IMG_0002.jpg.supplemental-metadata.json", "IMG_0002-EFFECTS.jpg.supplemental-metadata.json"},
			files:  []string{"IMG_0002.jpg", "IMG_0002-EFFECTS.jpg"},
			edited: "IMG_0002-EFFECTS-edited.jpg",
			want:   "IMG_0002-EFFECTS.jpg",
		},
		{
			name:   "short name",
			jsons:  []string{"6.jpg.supplemental-metadata.json", "6ABC.jpg.supplemental-metadata.json"},
			files:  []string{"6.jpg", "6ABC.jpg"},
			edited: "6ABC-edited.jpg",
			want:   "6ABC.jpg",
		},
		{
			name:   "plain edit still matched",
			jsons:  []string{"Some Photo.jpg.supplemental-metadata.json"},
			files:  []string{"Some Photo.jpg"},
			edited: "Some Photo-edited.jpg",
			want:   "Some Photo.jpg",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for range 20 { // map iteration order varies
				cat := directoryCatalog{
					jsons:          map[string]*assets.Metadata{},
					unMatchedFiles: map[string]*assetFile{},
					matchedFiles:   map[string]*assets.Asset{},
				}
				for i, j := range tc.jsons {
					title := tc.files[i]
					cat.jsons[j] = &assets.Metadata{FileName: title}
				}
				for _, f := range append(append([]string{}, tc.files...), tc.edited) {
					cat.unMatchedFiles[f] = &assetFile{base: f, date: time.Now()}
				}
				toc := &TakeoutCmd{
					catalogs:       map[string]directoryCatalog{"album": cat},
					supportedMedia: filetypes.DefaultSupportedMedia,
					infoCollector:  filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia),
					processor: fileprocessor.New(assettracker.New(),
						fileevent.NewRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))),
				}
				if err := toc.solvePuzzle(t.Context()); err != nil {
					t.Fatal(err)
				}
				i, ok := toc.catalogs["album"].matchedFiles[tc.edited]
				if !ok {
					t.Fatalf("%s not matched", tc.edited)
				}
				if i.Kind != assets.KindEdited {
					t.Errorf("%s: Kind = %v, want edited", tc.edited, i.Kind)
				}
				if i.OriginalFileName != tc.want {
					t.Fatalf("%s matched to the sidecar of %q, want %q", tc.edited, i.OriginalFileName, tc.want)
				}
			}
		})
	}
}
