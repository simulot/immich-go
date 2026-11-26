package upload

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// The Google Photos adapter marks a file HasNamesake when the input holds another file of the same
// name in the same folder, with other content: two Takeout series number their name clashes
// separately (#1473). Name, date and size cannot tell such files from versions of one photo, so only
// the checksum may match them with a server asset. They are uploaded, and never replace or skip a
// server asset because of its name.
func TestShouldUpload_namesakeIsComparedByChecksumOnly(t *testing.T) {
	date := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	for _, server := range []struct {
		name     string
		thisRun  bool
		wantSame AdviceCode // advice when the server asset holds the local file's bytes
	}{
		{"uploaded earlier in this run", true, AlreadyProcessed},
		{"on the server from a previous run", false, SameOnServer},
	} {
		for _, size := range []struct {
			name string
			size int64
		}{{"smaller", 1000}, {"same size", 2000}, {"bigger", 3000}} {
			for _, overwrite := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s, %s, overwrite %v", server.name, size.name, overwrite), func(t *testing.T) {
					sa := namesake(size.size, "sha-server", date)
					sa.ID = "id-server"
					ii := newAssetIndex()
					ii.add(sa, server.thisRun)

					la := namesake(2000, "sha-local", date.Add(2*time.Second))
					la.HasNamesake = true
					advice, err := ii.ShouldUpload(la, &UpCmd{Overwrite: overwrite})
					if err != nil {
						t.Fatal(err)
					}
					if advice.Advice != NamesakeOnServer || advice.ServerAsset != sa {
						t.Errorf("got %s (%s), want NamesakeOnServer: upload it and keep the server asset", advice.Advice, advice.Message)
					}
				})
			}
		}

		t.Run(server.name+", same bytes", func(t *testing.T) {
			sa := namesake(2000, "sha-local", date)
			sa.ID = "id-server"
			ii := newAssetIndex()
			ii.add(sa, server.thisRun)

			la := namesake(2000, "sha-local", date)
			la.HasNamesake = true
			advice, err := ii.ShouldUpload(la, &UpCmd{})
			if err != nil {
				t.Fatal(err)
			}
			if advice.Advice != server.wantSame {
				t.Errorf("got %s (%s), want %s: the server holds this very file", advice.Advice, advice.Message, server.wantSame)
			}
		})
	}
}

// namesake builds a takeout file named as the server asset, as the two files of #1473 are.
func namesake(size int64, checksum string, date time.Time) *assets.Asset {
	a := localAsset("IMG_3285.MOV", size, checksum, date)
	a.OriginalFileName = "IMG_3285.MOV"
	return a
}

// recordingImmich accepts every upload and records uploads and deletions. Any other call panics:
// the tests do not expect one.
type recordingImmich struct {
	immich.ImmichInterface
	uploaded    []string
	deleted     []string
	inspected   []string          // IDs read with GetAssetInfo
	unreachable string            // GetAssetInfo fails for this ID
	stacked     map[string]string // server asset ID → primary of the stack it is in
	stacks      [][]string        // IDs of the stacks created
}

func (c *recordingImmich) AssetUpload(_ context.Context, a *assets.Asset) (immich.AssetResponse, error) {
	c.uploaded = append(c.uploaded, a.File.Name())
	return immich.AssetResponse{ID: "id-" + a.File.Name(), Status: immich.UploadCreated}, nil
}

func (c *recordingImmich) CopyAsset(context.Context, string, string) error { return nil }

func (c *recordingImmich) DeleteAssets(_ context.Context, ids []string, _ bool) error {
	c.deleted = append(c.deleted, ids...)
	return nil
}

func (c *recordingImmich) GetAssetInfo(_ context.Context, id string) (*immich.Asset, error) {
	c.inspected = append(c.inspected, id)
	if id == c.unreachable {
		return nil, fmt.Errorf("asset %s: not found", id)
	}
	if p, ok := c.stacked[id]; ok {
		return &immich.Asset{ID: id, Stack: &immich.AssetStack{PrimaryAssetID: p}}, nil
	}
	return &immich.Asset{ID: id}, nil
}

func (c *recordingImmich) CreateStack(_ context.Context, ids []string) (string, error) {
	c.stacks = append(c.stacks, ids)
	return "stack", nil
}

// A new run over a server that already holds one of two namesakes uploads the other one next to it.
// Unmarked, the bigger file replaces the server asset, which is then deleted for good.
func TestHandleAsset_namesakeDoesNotReplaceTheServerAsset(t *testing.T) {
	date := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	const name = "Takeout/Google Photos/Photos from 2025/IMG_3285.MOV"

	for _, tc := range []struct {
		marked      bool
		wantDeleted []string
	}{
		{marked: true, wantDeleted: nil},
		{marked: false, wantDeleted: []string{"id-server"}},
	} {
		t.Run(fmt.Sprintf("marked %v", tc.marked), func(t *testing.T) {
			uc, _ := newTestUpCmd(t)
			client := &recordingImmich{}
			uc.client.Immich = client
			ctx := context.Background()

			server := namesake(1000, "sha-server", date)
			server.ID = "id-server"
			uc.assetIndex.add(server, false) // found on the server, from a previous run

			fsys := fstest.MapFS{name: {Data: bytes.Repeat([]byte("b"), 2000)}}
			la := &assets.Asset{
				File:             fshelper.FSName(fsys, name),
				OriginalFileName: "IMG_3285.MOV",
				FileSize:         2000,
				CaptureDate:      date,
				HasNamesake:      tc.marked,
			}
			uc.app.FileProcessor().RecordAssetDiscovered(ctx, la.File, 2000, fileevent.DiscoveredVideo)

			if err := uc.handleAsset(ctx, la, nil); err != nil {
				t.Fatalf("handleAsset: %v", err)
			}
			if !slices.Equal(client.uploaded, []string{name}) {
				t.Errorf("uploaded %q, want the local file", client.uploaded)
			}
			if !slices.Equal(client.deleted, tc.wantDeleted) {
				t.Errorf("deleted %q, want %q", client.deleted, tc.wantDeleted)
			}
		})
	}
}

// The mark of a namesake uploaded earlier in the run also keeps it from being replaced or
// skipped for by an unmarked file of its name and date.
func TestShouldUpload_namesakeOfThisRunIsNotReplaced(t *testing.T) {
	date := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	sa := namesake(1000, "sha-namesake", date)
	sa.ID = "id-namesake"
	sa.HasNamesake = true
	ii := newAssetIndex()
	ii.add(sa, true)

	la := namesake(2000, "sha-local", date.Add(3*time.Second))
	advice, err := ii.ShouldUpload(la, &UpCmd{})
	if err != nil {
		t.Fatal(err)
	}
	if advice.Advice != NamesakeOnServer || advice.ServerAsset != sa {
		t.Errorf("got %s (%s), want NamesakeOnServer", advice.Advice, advice.Message)
	}
}

// An edit pair whose original has a namesake takes no other base than its own original: the
// namesake has the original's title and date, but is another photo. The server copy of the
// group's own original still counts, so that its existing stack is merged.
func TestStackIDs_editOfANamesakeTakesNoOtherBase(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		otherThisRun  bool // the other namesake was uploaded earlier in this run
		originalStack bool // the group's original is on the server, in a stack
		want          []string
	}{
		{"other uploaded earlier in this run", true, false, []string{"id-edited", "id-original"}},
		{"other on the server from a previous run", false, false, []string{"id-edited", "id-original"}},
		{"rerun, the original stacked on the server", false, true, []string{"id-edited", "id-original", "id-primary"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ii := newAssetIndex()
			other := serverAsset("id-other", 3000, "sha-other", date.Add(3*time.Second))
			other.HasNamesake = tc.otherThisRun // a server asset carries no mark
			ii.add(other, tc.otherThisRun)

			original := localAsset("X.jpg", 2000, "sha-original", date)
			original.HasNamesake = true
			original.ID = "id-original"
			if tc.originalStack {
				// found on the server: the group's file got the server asset's ID
				ii.add(serverAsset("id-original", 2000, "sha-original", date), false)
				ii.stackPrimaryByID.Store("id-original", "id-primary")
			} else {
				ii.add(original, true)
			}
			edited := markEdited(localAsset("X-edited.jpg", 1000, "sha-edited", date))
			edited.HasNamesake = true
			edited.ID = "id-edited"
			ii.add(edited, true)
			g := assets.NewGroup(assets.GroupByOther, edited, original)
			g.CoverIndex = 0

			got, blocked := stackIDs(g, ii)
			if blocked {
				t.Fatalf("stackIDs reported an ambiguity: got %v", got)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// An edit uploaded alone, whose title belongs to namesakes, is stacked with none of them: the one
// on the server may be the other photo.
func TestStackIDs_editOfANamesakeAloneIsNotStacked(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)
	ii := newAssetIndex()
	ii.add(serverAsset("id-other", 3000, "sha-other", date.Add(3*time.Second)), false)

	edited := markEdited(localAsset("X-edited.jpg", 1000, "sha-edited", date))
	edited.HasNamesake = true
	edited.ID = "id-edited"
	g := assets.NewGroup(assets.GroupByNone, edited)
	g.CoverIndex = 0

	got, blocked := stackIDs(g, ii)
	if blocked {
		t.Fatalf("stackIDs reported an ambiguity: got %v", got)
	}
	if !slices.Equal(got, []string{"id-edited"}) {
		t.Errorf("got %v, want the edit alone", got)
	}
}

// An edit pair of namesakes is stacked with its own files. The other namesake on the server, of the
// same title and date, is not one of its versions: it is not even read, so that a failure to read
// it can't cancel the stack.
func TestHandleGroup_editPairOfNamesakesIgnoresTheOtherNamesake(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)
	uc, _ := newTestUpCmd(t)
	client := &recordingImmich{unreachable: "id-other"}
	uc.client.Immich = client
	ctx := context.Background()

	uc.assetIndex.add(serverAsset("id-other", 3000, "sha-other", date.Add(3*time.Second)), false)

	const dir = "Takeout/Google Photos/Photos from 2023/"
	fsys := fstest.MapFS{
		dir + "X.jpg":        {Data: bytes.Repeat([]byte("o"), 2000)},
		dir + "X-edited.jpg": {Data: bytes.Repeat([]byte("e"), 1000)},
	}
	original := &assets.Asset{File: fshelper.FSName(fsys, dir+"X.jpg"), OriginalFileName: "X.jpg", FileSize: 2000, CaptureDate: date, HasNamesake: true}
	edited := markEdited(&assets.Asset{File: fshelper.FSName(fsys, dir+"X-edited.jpg"), OriginalFileName: "X.jpg", FileSize: 1000, CaptureDate: date, HasNamesake: true})
	for _, a := range []*assets.Asset{original, edited} {
		uc.app.FileProcessor().RecordAssetDiscovered(ctx, a.File, int64(a.FileSize), fileevent.DiscoveredImage)
	}
	g := assets.NewGroup(assets.GroupByOther, edited, original)
	g.CoverIndex = 0

	if err := uc.handleGroup(ctx, g); err != nil {
		t.Fatalf("handleGroup: %v", err)
	}
	if slices.Contains(client.inspected, "id-other") {
		t.Errorf("the other namesake was read as a possible version of the pair")
	}
	want := [][]string{{"id-" + dir + "X-edited.jpg", "id-" + dir + "X.jpg"}}
	if len(client.stacks) != 1 || !slices.Equal(client.stacks[0], want[0]) {
		t.Errorf("stacks %q, want %q", client.stacks, want)
	}
}

// The rule for namesakes is the marked file's own: an unmarked file of the same group (here the
// RAW next to a marked JPG and its edit) keeps the protection of its server versions. Its smaller
// server copy is in a stack, so it is a version: the local RAW is uploaded next to it.
func TestHandleGroup_unmarkedFileOfAGroupWithNamesakesKeepsItsServerVersion(t *testing.T) {
	date := time.Date(2018, 5, 5, 10, 0, 0, 0, time.UTC)
	uc, _ := newTestUpCmd(t)
	client := &recordingImmich{stacked: map[string]string{"id-server-raw": "id-primary"}}
	uc.client.Immich = client
	ctx := context.Background()

	serverRAW := serverAsset("id-server-raw", 1000, "sha-server-raw", date)
	serverRAW.OriginalFileName = "IMG_0001.CR2"
	uc.assetIndex.add(serverRAW, false)

	const dir = "Takeout/Google Photos/Photos from 2018/"
	fsys := fstest.MapFS{
		dir + "IMG_0001.JPG":        {Data: bytes.Repeat([]byte("j"), 300)},
		dir + "IMG_0001-edited.JPG": {Data: bytes.Repeat([]byte("e"), 200)},
		dir + "IMG_0001.CR2":        {Data: bytes.Repeat([]byte("r"), 2000)},
	}
	jpg := &assets.Asset{File: fshelper.FSName(fsys, dir+"IMG_0001.JPG"), OriginalFileName: "IMG_0001.JPG", FileSize: 300, CaptureDate: date, HasNamesake: true}
	edited := markEdited(&assets.Asset{File: fshelper.FSName(fsys, dir+"IMG_0001-edited.JPG"), OriginalFileName: "IMG_0001.JPG", FileSize: 200, CaptureDate: date, HasNamesake: true})
	raw := &assets.Asset{File: fshelper.FSName(fsys, dir+"IMG_0001.CR2"), OriginalFileName: "IMG_0001.CR2", FileSize: 2000, CaptureDate: date}
	for _, a := range []*assets.Asset{edited, raw, jpg} {
		uc.app.FileProcessor().RecordAssetDiscovered(ctx, a.File, int64(a.FileSize), fileevent.DiscoveredImage)
	}
	g := assets.NewGroup(assets.GroupByOther, edited, raw, jpg)
	g.CoverIndex = 0

	if err := uc.handleGroup(ctx, g); err != nil {
		t.Fatalf("handleGroup: %v", err)
	}
	if len(client.deleted) != 0 {
		t.Errorf("deleted %q: the server RAW is a version, never replaced", client.deleted)
	}
	if !slices.Contains(client.uploaded, dir+"IMG_0001.CR2") {
		t.Errorf("uploaded %q, want the local RAW too", client.uploaded)
	}
}

// Likewise an unmarked edit pair still merges the existing stack of its original's server copy,
// though another file of its group is a namesake.
func TestStackIDs_unmarkedEditOfAGroupWithNamesakesMergesItsStack(t *testing.T) {
	date := time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC)
	ii := newAssetIndex()
	ii.add(serverAsset("id-server-x", 5000, "sha-server-x", date), false)
	ii.stackPrimaryByID.Store("id-server-x", "id-primary")

	original := localAsset("X.jpg", 2000, "sha-original", date)
	original.ID = "id-original"
	ii.add(original, true)
	edited := markEdited(localAsset("X-edited.jpg", 1000, "sha-edited", date))
	edited.ID = "id-edited"
	ii.add(edited, true)
	raw := localAsset("X.CR2", 9000, "sha-raw", date)
	raw.OriginalFileName = "X.CR2"
	raw.HasNamesake = true
	raw.ID = "id-raw"
	ii.add(raw, true)
	g := assets.NewGroup(assets.GroupByOther, edited, original, raw)
	g.CoverIndex = 0

	got, blocked := stackIDs(g, ii)
	if blocked {
		t.Fatalf("stackIDs reported an ambiguity: got %v", got)
	}
	if !slices.Contains(got, "id-primary") {
		t.Errorf("got %v, want the existing stack of the original merged (id-primary)", got)
	}
}
