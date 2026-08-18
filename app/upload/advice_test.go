package upload

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// localAsset builds an asset as the Google Photos adapter does for a takeout file: File is the name in
// the archive, OriginalFileName the title of the sidecar (which is the original's name, also for
// the "-edited" variant), and CaptureDate the sidecar's date.
func localAsset(fileName, title string, size int64, checksum string, date time.Time) *assets.Asset {
	return &assets.Asset{
		File:             fshelper.FSName(nil, "Takeout/Google Photos/album/"+fileName),
		OriginalFileName: title,
		FileSize:         int(size),
		Checksum:         checksum,
		CaptureDate:      date,
	}
}

func TestShouldUpload_editedSiblingIsNotAServerVariant(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	for _, tc := range []struct {
		name         string
		originalSize int64
		editedSize   int64
	}{
		{"original bigger than edited", 1141253, 34096},
		{"original smaller than edited", 35565, 1148341},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := localAsset("X.jpg", "X.jpg", tc.originalSize, "sha-original", date)
			edited := localAsset("X-edited.jpg", "X.jpg", tc.editedSize, "sha-edited", date)
			g := assets.NewGroup(assets.GroupByOther, edited, original)

			ii := newAssetIndex()
			uc := &UpCmd{}

			// the edited file is processed first: not on the server, uploaded and indexed
			advice, err := ii.ShouldUpload(edited, uc, g.Assets...)
			if err != nil {
				t.Fatal(err)
			}
			if advice.Advice != NotOnServer {
				t.Fatalf("edited: got %s, want NotOnServer", advice.Advice)
			}
			edited.ID = "id-edited"
			ii.addLocalAsset(edited)

			// the original must not be taken for a smaller/bigger version of its edited sibling
			advice, err = ii.ShouldUpload(original, uc, g.Assets...)
			if err != nil {
				t.Fatal(err)
			}
			if advice.Advice != NotOnServer {
				t.Errorf("original: got %s (%s), want NotOnServer", advice.Advice, advice.Message)
			}
		})
	}
}

func TestShouldUpload_sameNameOutsideTheGroupIsStillAVariant(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	// a copy of X.jpg from another directory of the takeout, uploaded earlier in the run
	other := localAsset("X.jpg", "X.jpg", 1000, "sha-other", date)
	other.ID = "id-other"
	ii := newAssetIndex()
	ii.addLocalAsset(other)

	original := localAsset("X.jpg", "X.jpg", 2000, "sha-original", date)
	sibling := localAsset("X-edited.jpg", "X.jpg", 500, "sha-edited", date)
	g := assets.NewGroup(assets.GroupByOther, sibling, original)

	advice, err := ii.ShouldUpload(original, &UpCmd{}, g.Assets...)
	if err != nil {
		t.Fatal(err)
	}
	if advice.Advice != SmallerOnServer || advice.ServerAsset != other {
		t.Errorf("got %s for %v, want SmallerOnServer for the other directory's copy", advice.Advice, advice.ServerAsset)
	}
}

func TestStackIDs(t *testing.T) {
	a := func(id string) *assets.Asset { return &assets.Asset{ID: id} }
	for _, tc := range []struct {
		name  string
		g     *assets.Group
		cover int
		want  []string
	}{
		{"cover first", assets.NewGroup(assets.GroupByBurst, a("1"), a("2"), a("3")), 1, []string{"2", "1", "3"}},
		{"empty ids are left out", assets.NewGroup(assets.GroupByOther, a(""), a("2"), a("")), 0, []string{"2"}},
		{"duplicate ids are listed once", assets.NewGroup(assets.GroupByOther, a("1"), a("1")), 0, []string{"1"}},
		{"cover index out of range", assets.NewGroup(assets.GroupByOther, a("1"), a("2")), 5, []string{"1", "2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.g.CoverIndex = tc.cover
			got := stackIDs(tc.g)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
