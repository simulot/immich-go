package gp

import (
	"context"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/kr/pretty"
	"github.com/simulot/immich-go/adapters"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
)

type fileLinks struct {
	image   string
	video   string
	sidecar string
}

func TestGPAssets(t *testing.T) {
	tc := []struct {
		name           string
		fsys           []fs.FS
		flags          ImportFlags
		expectedFiles  map[string]fileLinks
		expectedCounts []int64
		expectedAlbums map[string][]string
	}{
		{
			name: "motion picture",
			flags: ImportFlags{
				BannedFiles:    namematcher.MustList(`@eaDir/`, `.@__thumb`, `SYNOFILE_THUMB_*.*`),
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodNone,
				},
				KeepJSONLess:   true,
				InclusionFlags: cliflags.InclusionFlags{},
			},
			fsys: []fs.FS{
				newInMemFS("MemFS").
					addJSONImage("nomotion.MP4.JSON", "nomotion.MP4").
					addFile2("motion/nomotion.MP4").
					addJSONImage("motion/PXL_20210102_221126856.MP~2.jpg.JSON", "motion/PXL_20210102_221126856.MP~2.jpg").
					addFile2("motion/PXL_20210102_221126856.MP~2").
					addFile2("motion/PXL_20210102_221126856.MP~2.jpg").
					addJSONImage("motion/PXL_20210102_221126856.MP.jpg.JSON", "motion/PXL_20210102_221126856.MP.jpg").
					addFile2("motion/PXL_20210102_221126856.MP.jpg").
					addFile2("motion/PXL_20210102_221126856.MP").
					addJSONImage("motion/20231227_152817.jpg.JSON", "motion/20231227_152817.jpg").
					addFile2("motion/20231227_152817.jpg").
					addFile2("motion/20231227_152817.MP4"),
			},
			expectedFiles: map[string]fileLinks{
				"motion/PXL_20210102_221126856.MP.jpg":   {image: "motion/PXL_20210102_221126856.MP.jpg", video: "motion/PXL_20210102_221126856.MP"},
				"motion/PXL_20210102_221126856.MP~2.jpg": {image: "motion/PXL_20210102_221126856.MP~2.jpg", video: "motion/PXL_20210102_221126856.MP~2"},
				"motion/20231227_152817.jpg":             {image: "motion/20231227_152817.jpg", video: "motion/20231227_152817.MP4"},
				"motion/nomotion.MP4":                    {video: "motion/nomotion.MP4"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 3).
				Set(fileevent.DiscoveredVideo, 4).Set(fileevent.AnalysisAssociatedMetadata, 6).
				Set(fileevent.DiscoveredSidecar, 4).Set(fileevent.AnalysisMissingAssociatedMetadata, 1).Value(),
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			recorder := fileevent.NewRecorder(nil)
			b, err := NewTakeout(ctx, recorder, &c.flags, c.fsys...)
			if err != nil {
				t.Error(err)
			}

			groupChan, err := b.Browse(ctx)
			if err != nil {
				t.Error(err)
			}

			results := map[string]fileLinks{}
			albums := map[string][]string{}
			for g := range groupChan {
				if err = g.Validate(); err != nil {
					t.Error(err)
					return
				}
				fileName := g.Assets[0].FileName
				links := fileLinks{}
				for _, a := range g.Assets {
					ext := path.Ext(a.FileName)
					switch b.flags.SupportedMedia.TypeFromExt(ext) {
					case metadata.TypeImage:
						links.image = a.FileName
						if g.Kind == adapters.GroupKindMotionPhoto {
							fileName = a.FileName
						}
					case metadata.TypeVideo:
						links.video = a.FileName
					}
					if a.SideCar.FileName != "" {
						links.sidecar = a.SideCar.FileName
					}
					a.Close()
				}
				results[fileName] = links

				if len(c.expectedAlbums) > 0 {
					for _, album := range g.Albums {
						albums[album.Title] = append(albums[album.Title], fileName)
					}
				}
			}

			if !reflect.DeepEqual(results, c.expectedFiles) {
				t.Errorf("file list difference\n")
				pretty.Ldiff(t, c.expectedFiles, results)
			}
			if !reflect.DeepEqual(recorder.GetCounts(), c.expectedCounts) {
				t.Errorf("counters difference\n")
				pretty.Ldiff(t, c.expectedCounts, recorder.GetCounts())
			}
			if c.expectedAlbums != nil {
				compareAlbums(t, albums, c.expectedAlbums)
			}
		})
	}
}

func compareAlbums(t *testing.T, a, b map[string][]string) {
	a = sortAlbum(a)
	b = sortAlbum(b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("album list difference\n")
		pretty.Ldiff(t, a, b)
	}
}

func sortAlbum(a map[string][]string) map[string][]string {
	for k := range a {
		sort.Strings(a[k])
	}
	return a
}
