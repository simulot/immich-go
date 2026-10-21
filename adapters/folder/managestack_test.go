package folder

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/spf13/cobra"
)

// The --manage-* options of `upload from-folder` stack fresh files (an SD card, a scanner's
// output) by name and capture time. Files restored from an immich-go archive are never grouped:
// their stacks come from their .JSON sidecars.

var shotAt = time.Date(2023, 10, 14, 18, 32, 46, 0, time.UTC)

// fixture writes files into a folder named "sdcard" (the adapter reads dates from paths, so the
// folder name holds none), every file with the modification time shotAt. It returns the folder.
func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sdcard")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, shotAt, shotAt); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// xmp is an XMP sidecar giving the capture time d.
func xmp(d time.Time) string {
	return `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description rdf:about="" xmlns:exif="http://ns.adobe.com/exif/1.0/">` +
		`<exif:DateTimeOriginal>` + d.Format("2006-01-02T15:04:05.000Z") + `</exif:DateTimeOriginal>` +
		`</rdf:Description></rdf:RDF></x:xmpmeta>`
}

// Fresh files, as a camera or a scanner writes them.
var (
	rawJPEG = map[string]string{
		"IMG_1.JPG": "jpeg",
		"IMG_1.CR2": "raw",
	}
	namedBurst = map[string]string{
		"IMG_20231014_183246_BURST001_COVER.jpg": "frame 1",
		"IMG_20231014_183246_BURST002.jpg":       "frame 2",
	}
	timedBurst = map[string]string{ // no burst in the names: frames 300 ms apart
		"DSC_0001.JPG":     "frame 1",
		"DSC_0001.JPG.xmp": xmp(shotAt.Add(100 * time.Millisecond)),
		"DSC_0002.JPG":     "frame 2",
		"DSC_0002.JPG.xmp": xmp(shotAt.Add(400 * time.Millisecond)),
	}
	epsonScans = map[string]string{
		"photo_0001.jpg":   "original scan",
		"photo_0001_a.jpg": "corrected scan",
		"photo_0001_b.jpg": "back of the photo",
	}
)

// archived returns the files plus, for each media file, the .JSON sidecar `archive` writes, naming
// one source stack of all of them, the first by name its primary.
func archived(t *testing.T, files map[string]string) map[string]string {
	t.Helper()
	var media []string
	for name := range files {
		if strings.EqualFold(path.Ext(name), ".xmp") {
			continue
		}
		media = append(media, name)
	}
	slices.Sort(media)
	out := map[string]string{}
	for name, content := range files {
		out[name] = content
	}
	for i, name := range media {
		taken := shotAt
		if x, ok := files[name+".xmp"]; ok {
			taken = xmpTime(t, x)
		}
		var b bytes.Buffer
		md := &assets.Metadata{
			FileName:  name,
			DateTaken: taken,
			Stack:     &assets.Stack{ID: "source-stack", Primary: i == 0, AssetCount: len(media)},
		}
		if err := jsonsidecar.Write(md, &b); err != nil {
			t.Fatal(err)
		}
		out[name+".JSON"] = b.String()
	}
	return out
}

func xmpTime(t *testing.T, x string) time.Time {
	t.Helper()
	const open = "<exif:DateTimeOriginal>"
	i := strings.Index(x, open)
	j := strings.Index(x, "</exif:DateTimeOriginal>")
	d, err := time.Parse("2006-01-02T15:04:05.000Z", x[i+len(open):j])
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// uploadRunner stands in for app/upload's UpCmd: it holds upload's --manage-* options and, like
// UpCmd.Run and handleGroup, applies their filters to each group the adapter sends.
type uploadRunner struct {
	shared.StackOptions
	groups []*assets.Group
}

func (r *uploadRunner) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	fs := []filters.Filter{r.ManageBurst.GroupFilter(), r.ManageRawJPG.GroupFilter(), r.ManageHEICJPG.GroupFilter()}
	for g := range adapter.Browse(cmd.Context()) {
		r.groups = append(r.groups, filters.ApplyFilters(g, fs...))
	}
	return nil
}

// group is what upload gets for one group: a stack when grouping is not "none".
type group struct {
	grouping string
	files    string // base names, sorted
	cover    string // the stack's cover; "" when grouping is "none"
	removed  string // base names the filters dropped, sorted
}

var groupings = map[assets.GroupBy]string{
	assets.GroupByNone:    "none",
	assets.GroupByBurst:   "burst",
	assets.GroupByRawJpg:  "raw+jpeg",
	assets.GroupByHeicJpg: "heic+jpeg",
	assets.GroupByOther:   "other",
}

type uploadResult struct {
	groups []group                  // sorted by files
	assets map[string]*assets.Asset // by base name
	log    string
}

// upload runs `immich-go upload from-folder args...` through the real command wiring: a root
// command with the configuration manager (as app/root), an `upload` command that registers the
// --manage-* options as persistent flags (as app/upload), and from-folder under it.
func upload(t *testing.T, args ...string) uploadResult {
	t.Helper()
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	ctx := context.Background()

	root := &cobra.Command{Use: "immich-go"}
	a := app.New(ctx, root)
	a.RegisterFlags(root.PersistentFlags())
	var logBuf bytes.Buffer
	a.SetLog(&app.Log{Logger: slog.New(slog.NewTextHandler(&logBuf, nil))})
	a.SetFileProcessor(fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))))
	a.SetTZ(time.UTC)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := a.Config.Init(a.CfgFile); err != nil {
			return err
		}
		return a.Config.ProcessCommand(cmd)
	}

	up := &cobra.Command{Use: "upload"}
	r := &uploadRunner{}
	r.RegisterFlags(up.PersistentFlags())
	up.AddCommand(NewFromFolderCommand(ctx, up, a, r))
	root.AddCommand(up)

	root.SetArgs(append([]string{"upload", "from-folder"}, args...))
	root.SetOut(io.Discard)
	if err := root.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}

	res := uploadResult{assets: map[string]*assets.Asset{}, log: logBuf.String()}
	for _, g := range r.groups {
		var v group
		v.grouping = groupings[g.Grouping]
		names := make([]string, 0, len(g.Assets))
		for _, x := range g.Assets {
			names = append(names, path.Base(x.File.Name()))
			res.assets[path.Base(x.File.Name())] = x
			t.Cleanup(func() { _ = x.Close() })
		}
		slices.Sort(names)
		v.files = strings.Join(names, " ")
		if g.Grouping != assets.GroupByNone && g.CoverIndex >= 0 && g.CoverIndex < len(g.Assets) {
			v.cover = path.Base(g.Assets[g.CoverIndex].File.Name())
		}
		removed := make([]string, 0, len(g.Removed))
		for _, x := range g.Removed {
			removed = append(removed, path.Base(x.Asset.File.Name()))
			res.assets[path.Base(x.Asset.File.Name())] = x.Asset
			t.Cleanup(func() { _ = x.Asset.Close() })
		}
		slices.Sort(removed)
		v.removed = strings.Join(removed, " ")
		res.groups = append(res.groups, v)
	}
	slices.SortFunc(res.groups, func(x, y group) int { return strings.Compare(x.files, y.files) })
	return res
}

func checkGroups(t *testing.T, got uploadResult, want ...group) {
	t.Helper()
	slices.SortFunc(want, func(x, y group) int { return strings.Compare(x.files, y.files) })
	if !slices.Equal(got.groups, want) {
		t.Errorf("groups:\n got  %+v\n want %+v", got.groups, want)
	}
}

// alone is the result for files each sent alone, none dropped.
func alone(names ...string) []group {
	gs := make([]group, 0, len(names))
	for _, n := range names {
		gs = append(gs, group{grouping: "none", files: n})
	}
	return gs
}

func TestManageRawJPEGStacksFreshFolder(t *testing.T) {
	got := upload(t, "--manage-raw-jpeg=StackCoverJPG", fixture(t, rawJPEG))
	checkGroups(t, got, group{grouping: "raw+jpeg", files: "IMG_1.CR2 IMG_1.JPG", cover: "IMG_1.JPG"})
}

func TestManageBurstStacksFreshFolder(t *testing.T) {
	t.Run("burst in the names", func(t *testing.T) {
		got := upload(t, "--manage-burst=Stack", fixture(t, namedBurst))
		checkGroups(t, got, group{
			grouping: "burst",
			files:    "IMG_20231014_183246_BURST001_COVER.jpg IMG_20231014_183246_BURST002.jpg",
			cover:    "IMG_20231014_183246_BURST001_COVER.jpg",
		})
	})
	// Only the burst grouper, which --manage-burst adds, groups frames by capture time.
	t.Run("burst by capture time", func(t *testing.T) {
		got := upload(t, "--manage-burst=Stack", fixture(t, timedBurst))
		checkGroups(t, got, group{grouping: "burst", files: "DSC_0001.JPG DSC_0002.JPG", cover: "DSC_0001.JPG"})
	})
}

// Only the Epson FastFoto grouper, which --manage-epson-fastfoto adds, groups a scan with its
// corrected copy and its back.
func TestManageEpsonFastFotoFreshFolder(t *testing.T) {
	got := upload(t, "--manage-epson-fastfoto", fixture(t, epsonScans))
	checkGroups(t, got, group{grouping: "other", files: "photo_0001.jpg photo_0001_a.jpg photo_0001_b.jpg", cover: "photo_0001_a.jpg"})
}

// The --manage-* values can come from the configuration file instead of the command line.
func TestManageBurstFromConfigFile(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "immich-go.yaml")
	if err := os.WriteFile(cfg, []byte("upload:\n  manage-burst: Stack\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := upload(t, "--config="+cfg, fixture(t, timedBurst))
	checkGroups(t, got, group{grouping: "burst", files: "DSC_0001.JPG DSC_0002.JPG", cover: "DSC_0001.JPG"})
}

// KeepJPG uploads the JPEG and drops the RAW (documented behavior).
func TestManageRawJPEGKeepJPGFreshFolder(t *testing.T) {
	got := upload(t, "--manage-raw-jpeg=KeepJPG", fixture(t, rawJPEG))
	checkGroups(t, got, group{grouping: "none", files: "IMG_1.JPG", removed: "IMG_1.CR2"})
}

// Files with an immich-go .JSON sidecar are sent alone whatever the options: none is stacked by
// name or capture time, none is dropped, and the stack their sidecar names is read.
func TestArchivedAssetsNotGrouped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		flags []string
	}{
		{"raw+jpeg, StackCoverJPG", rawJPEG, []string{"--manage-raw-jpeg=StackCoverJPG"}},
		{"raw+jpeg, KeepJPG", rawJPEG, []string{"--manage-raw-jpeg=KeepJPG"}},
		{"raw+jpeg, KeepRaw", rawJPEG, []string{"--manage-raw-jpeg=KeepRaw"}},
		{"burst in the names, Stack", namedBurst, []string{"--manage-burst=Stack"}},
		{"burst in the names, StackKeepJPEG", namedBurst, []string{"--manage-burst=StackKeepJPEG"}},
		{"burst by capture time, Stack", timedBurst, []string{"--manage-burst=Stack"}},
		{"epson fastfoto", epsonScans, []string{"--manage-epson-fastfoto"}},
		{"no options", rawJPEG, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := archived(t, tc.files)
			got := upload(t, append(tc.flags, fixture(t, files))...)

			var media []string
			for name := range files {
				if ext := strings.ToLower(path.Ext(name)); ext != ".json" && ext != ".xmp" {
					media = append(media, name)
				}
			}
			checkGroups(t, got, alone(media...)...)
			for _, name := range media {
				a := got.assets[name]
				if a == nil {
					t.Errorf("%s: not sent", name)
					continue
				}
				if !a.FromArchive {
					t.Errorf("%s: not taken as restored from an archive", name)
				}
				if a.FromApplication == nil || a.FromApplication.Stack == nil || a.FromApplication.Stack.ID != "source-stack" {
					t.Errorf("%s: the sidecar's stack was not read: %+v", name, a.FromApplication)
				}
			}
			if n := archiveLines(got.log, len(media)); n != 1 {
				t.Errorf("%d info lines counting the %d archived files, want 1; log:\n%s", n, len(media), got.log)
			}
		})
	}
}

// An archived file and a fresh file with the same name and capture time: never one stack, and
// no Keep* filter drops the archived one.
func TestMixedArchiveAndFreshNotGroupedTogether(t *testing.T) {
	files := map[string]string{
		"IMG_1.JPG":      "jpeg",
		"IMG_1.JPG.JSON": archived(t, map[string]string{"IMG_1.JPG": "jpeg"})["IMG_1.JPG.JSON"],
		"IMG_1.CR2":      "raw",
	}
	for _, flag := range []string{"--manage-raw-jpeg=StackCoverJPG", "--manage-raw-jpeg=KeepRaw"} {
		t.Run(flag, func(t *testing.T) {
			got := upload(t, flag, fixture(t, files))
			checkGroups(t, got, alone("IMG_1.CR2", "IMG_1.JPG")...)
			if a := got.assets["IMG_1.JPG"]; a == nil || !a.FromArchive {
				t.Errorf("IMG_1.JPG: not taken as restored from an archive")
			}
			if a := got.assets["IMG_1.CR2"]; a == nil || a.FromArchive {
				t.Errorf("IMG_1.CR2 has no sidecar: taken as restored from an archive")
			}
			if n := archiveLines(got.log, 1); n != 1 {
				t.Errorf("%d info lines counting 1 archived file, want 1; log:\n%s", n, got.log)
			}
		})
	}
}

// Without options, a fresh folder is grouped as upstream main groups it: the series grouper alone,
// and upload's filters turn its RAW+JPEG and burst groups into plain uploads.
func TestDefaultFlagsFreshFolderUnchanged(t *testing.T) {
	files := map[string]string{}
	for _, set := range []map[string]string{rawJPEG, namedBurst, timedBurst, epsonScans} {
		for name, content := range set {
			files[name] = content
		}
	}
	got := upload(t, fixture(t, files))
	want := append(alone("DSC_0001.JPG", "DSC_0002.JPG", "photo_0001.jpg", "photo_0001_a.jpg", "photo_0001_b.jpg"),
		group{grouping: "none", files: "IMG_1.CR2 IMG_1.JPG"},
		group{grouping: "none", files: "IMG_20231014_183246_BURST001_COVER.jpg IMG_20231014_183246_BURST002.jpg"},
	)
	checkGroups(t, got, want...)
	if strings.Contains(got.log, "immich-go archive") {
		t.Errorf("a fresh folder logged about archived files:\n%s", got.log)
	}
}

// archiveLines counts the info lines that report n files taken as restored from an archive.
func archiveLines(log string, n int) int {
	c := 0
	for _, l := range strings.Split(log, "\n") {
		if strings.Contains(l, "level=INFO") && strings.Contains(l, "immich-go archive") && strings.Contains(l, "files="+strconv.Itoa(n)) {
			c++
		}
	}
	return c
}
