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

	got, blocked := stackIDs(g, ii)
	if blocked {
		t.Fatalf("stackIDs reported an ambiguity")
	}
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

// When the server original already belongs to a stack, the stack's primary is enough:
// the server merges the whole existing stack, so the base itself is not listed.
func TestStackIDs_editedAssetMergesAnExistingStack(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	base := serverAsset("id-base", 13749, "sha-base", date)
	ii.stackPrimaryByID.Store("id-base", "id-primary")
	ii.add(base, false)

	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited)

	got, blocked := stackIDs(g, ii)
	if blocked {
		t.Fatalf("stackIDs reported an ambiguity")
	}
	want := []string{"id-edited", "id-primary"}
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

	got, blocked := stackIDs(g, ii)
	if !blocked {
		t.Fatalf("stackIDs did not report the ambiguity between the server copies")
	}
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

	got, blocked := stackIDs(g, ii)
	if blocked {
		t.Fatalf("stackIDs reported an ambiguity")
	}
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

// A server asset that is a version (stacked, or edited on the server) is never a size
// variant: the local original must upload and be stacked with it.
func TestShouldUpload_editPairSkipsAStackedServerVersion(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)

	ii := newAssetIndex()
	server := serverAsset("id-server", 500, "sha-old-edit", date)
	ii.add(server, false)
	ii.serverVersions.Add("id-server")
	ii.stackPrimaryByID.Store("id-server", "id-primary")

	edited := markEdited(localAsset("X-edited.jpg", 1131, "sha-edited", date))
	original := localAsset("X.jpg", 13749, "sha-original", date)
	g := assets.NewGroup(assets.GroupByOther, edited, original)

	advice, err := ii.ShouldUpload(original, &UpCmd{}, g.Assets...)
	if err != nil {
		t.Fatal(err)
	}
	if advice.Advice != NotOnServer {
		t.Errorf("original: got %s for %v, want NotOnServer for the stacked server version", advice.Advice, advice.ServerAsset)
	}
}
