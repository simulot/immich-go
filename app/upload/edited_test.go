package upload

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

// markEdited turns a local asset into a Google Takeout "...-edited" variant, as the
// Google Photos adapter does.
func markEdited(a *assets.Asset) *assets.Asset {
	a.Kind = assets.KindEdited
	a.IsModified = true
	a.IsCover = true
	return a
}

// An edited file uploaded alone must be stacked over the server asset that has the
// original's name, even though the two files have different names on disk.
func TestStackIDs_editedAssetStacksOverServerBase(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	ii.add(serverAsset("id-base", 13749, "sha-base", date), false)

	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited) // a standalone file is a one-asset group
	g.CoverIndex = 0

	got := stackIDs(g, ii)
	want := []string{"id-edited", "id-base"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// When the server original already belongs to a stack, include that stack's
// primary so the server merges the stacks instead of splitting them.
func TestStackIDs_editedAssetMergesAnExistingStack(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	base := serverAsset("id-base", 13749, "sha-base", date)
	base.StackPrimaryID = "id-primary"
	ii.add(base, false)

	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited)

	got := stackIDs(g, ii)
	want := []string{"id-edited", "id-base", "id-primary"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// Several server assets with the original's name and date: do not guess.
func TestStackIDs_editedAssetDoesNotGuessBetweenServerCopies(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	ii.add(serverAsset("id-a", 1000, "sha-a", date), false)
	ii.add(serverAsset("id-b", 2000, "sha-b", date), false)

	edited := markEdited(localAsset("X-edited.jpg", 500, "sha-edited", date))
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited)

	got := stackIDs(g, ii)
	want := []string{"id-edited"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A server copy with a different name and date is not the original of the edit.
func TestStackIDs_editedAssetIgnoresOtherDates(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	ii.add(serverAsset("id-base", 13749, "sha-base", date.Add(time.Minute)), false)

	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited)

	got := stackIDs(g, ii)
	want := []string{"id-edited"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// When the server already has a version of the photo, an edit pair must still
// upload both files and keep the server version. The size heuristic must not turn
// one of the pair into a server copy.
func TestShouldUpload_editPairIsNeverAServerVariant(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	for _, tc := range []struct {
		name         string
		originalSize int64
		editedSize   int64
		serverSize   int64
	}{
		{"server smaller than the original", 13749, 1131, 1000},
		{"server bigger than the original", 1145, 14095, 20000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ii := newAssetIndex()
			// the server holds the edited version
			ii.add(serverAsset("id-server", tc.serverSize, "sha-edited", date), false)

			original := localAsset("X.jpg", tc.originalSize, "sha-original", date)
			edited := markEdited(localAsset("X-edited.jpg", tc.editedSize, "sha-edited", date))
			g := assets.NewGroup(assets.GroupByOther, edited, original)

			advice, err := ii.ShouldUpload(original, &UpCmd{}, g.Assets...)
			if err != nil {
				t.Fatal(err)
			}
			if advice.Advice != NotOnServer {
				t.Errorf("original: got %s (%s), want NotOnServer", advice.Advice, advice.Message)
			}
		})
	}
}

// A server copy of the original (its bytes are not the edit's) is still upgraded when
// the takeout has a bigger original. The edit must not disable that.
func TestShouldUpload_editPairStillUpgradesAServerCopyOfTheOriginal(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	server := serverAsset("id-server", 1000, "sha-small", date)
	ii.add(server, false)

	original := localAsset("X.jpg", 13749, "sha-original", date)
	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	g := assets.NewGroup(assets.GroupByOther, edited, original)

	advice, err := ii.ShouldUpload(original, &UpCmd{}, g.Assets...)
	if err != nil {
		t.Fatal(err)
	}
	if advice.Advice != SmallerOnServer || advice.ServerAsset != server {
		t.Errorf("original: got %s for %v, want SmallerOnServer for the server copy", advice.Advice, advice.ServerAsset)
	}
}
