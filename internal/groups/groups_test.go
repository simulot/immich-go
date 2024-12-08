package groups_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/series"
)

func mockAsset(ic *filenames.InfoCollector, name string, dateTaken time.Time) *assets.Asset {
	a := assets.Asset{
		File:        fshelper.FSName(nil, name),
		FileDate:    dateTaken,
		CaptureDate: dateTaken,
	}
	a.SetNameInfo(ic.GetInfo(name))
	return &a
}

func TestGroup(t *testing.T) {
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)

	testAssets := []*assets.Asset{
		mockAsset(ic, "photo1.jpg", t0.Add(50*time.Hour)),
		mockAsset(ic, "photo2.jpg", t0.Add(55*time.Hour)),
		mockAsset(ic, "IMG_001.jpg", t0),                            // Group 1
		mockAsset(ic, "IMG_002.jpg", t0.Add(200*time.Millisecond)),  // Group 1
		mockAsset(ic, "IMG_003.jpg", t0.Add(400*time.Millisecond)),  // Group 1
		mockAsset(ic, "IMG_004.jpg", t0.Add(600*time.Millisecond)),  // Group 1
		mockAsset(ic, "IMG_005.jpg", t0.Add(800*time.Millisecond)),  // Group 1
		mockAsset(ic, "IMG_006.jpg", t0.Add(1000*time.Millisecond)), // Group 1
		mockAsset(ic, "IMG_007.jpg", t0.Add(1200*time.Millisecond)), // Group 1
		mockAsset(ic, "IMG_008.jpg", t0.Add(1400*time.Millisecond)), // Group 1
		mockAsset(ic, "IMG_009.jpg", t0.Add(1600*time.Millisecond)), // Group 1
		mockAsset(ic, "photo3.jpg", t0.Add(5*time.Hour)),
		mockAsset(ic, "photo4.jpg", t0.Add(6*time.Hour)),
		mockAsset(ic, "IMG_001.jpg", t0.Add(7*time.Hour)),
		mockAsset(ic, "IMG_20231014_183246_BURST001_COVER.jpg", time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local)), // Group 2
		mockAsset(ic, "IMG_20231014_183246_BURST002.jpg", time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local)),       // Group 2
		mockAsset(ic, "IMG_003.jpg", t0.Add(9*time.Hour)),                                                           // Group 3
		mockAsset(ic, "IMG_003.raw", t0.Add(9*time.Hour)),                                                           // Group 3
		mockAsset(ic, "IMG_004.heic", t0.Add(10*time.Hour)),                                                         // Group 4
		mockAsset(ic, "IMG_004.jpg", t0.Add(10*time.Hour+100*time.Millisecond)),                                     // Group 4
		mockAsset(ic, "IMG_005.raw", t0.Add(100*time.Hour)),
		mockAsset(ic, "IMG_005.jpg", t0.Add(101*time.Hour)),
		mockAsset(ic, "00001IMG_00001_BURST20210101153000.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)),       // Group 5
		mockAsset(ic, "00002IMG_00002_BURST20210101153000_COVER.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)), // Group 5
		mockAsset(ic, "00003IMG_00003_BURST20210101153000.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)),       // Group 5
		mockAsset(ic, "IMG_006.heic", t0.Add(110*time.Hour)),
		mockAsset(ic, "photo5.jpg", t0.Add(120*time.Hour)),
		mockAsset(ic, "photo6.jpg", t0.Add(130*time.Hour)),
		mockAsset(ic, "IMG_030.mp4", t0.Add(140*time.Hour)),
		mockAsset(ic, "IMG_030.mov", t0.Add(150*time.Hour)),
		mockAsset(ic, "IMG_030.jpg", t0.Add(160*time.Hour)),
		mockAsset(ic, "IMG_030.heic", t0.Add(170*time.Hour)),
	}

	expectedGroup := []*assets.Group{
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "00001IMG_00001_BURST20210101153000.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)),
			mockAsset(ic, "00002IMG_00002_BURST20210101153000_COVER.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)),
			mockAsset(ic, "00003IMG_00003_BURST20210101153000.jpg", time.Date(2021, 1, 1, 15, 30, 0, 0, time.Local)),
		).SetCover(1),
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "IMG_001.jpg", t0),
			mockAsset(ic, "IMG_002.jpg", t0.Add(200*time.Millisecond)),
			mockAsset(ic, "IMG_003.jpg", t0.Add(400*time.Millisecond)),
			mockAsset(ic, "IMG_004.jpg", t0.Add(600*time.Millisecond)),
			mockAsset(ic, "IMG_005.jpg", t0.Add(800*time.Millisecond)),
			mockAsset(ic, "IMG_006.jpg", t0.Add(1000*time.Millisecond)),
			mockAsset(ic, "IMG_007.jpg", t0.Add(1200*time.Millisecond)),
			mockAsset(ic, "IMG_008.jpg", t0.Add(1400*time.Millisecond)),
			mockAsset(ic, "IMG_009.jpg", t0.Add(1600*time.Millisecond)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "IMG_20231014_183246_BURST001_COVER.jpg", time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local)),
			mockAsset(ic, "IMG_20231014_183246_BURST002.jpg", time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByHeicJpg,
			mockAsset(ic, "IMG_004.heic", t0.Add(10*time.Hour)),
			mockAsset(ic, "IMG_004.jpg", t0.Add(10*time.Hour+100*time.Millisecond)),
		),
		assets.NewGroup(assets.GroupByRawJpg,
			mockAsset(ic, "IMG_003.jpg", t0.Add(9*time.Hour)),
			mockAsset(ic, "IMG_003.raw", t0.Add(9*time.Hour)),
		),
	}

	expectedAssets := []*assets.Asset{
		mockAsset(ic, "photo1.jpg", t0.Add(50*time.Hour)),
		mockAsset(ic, "photo2.jpg", t0.Add(55*time.Hour)),
		mockAsset(ic, "photo3.jpg", t0.Add(5*time.Hour)),
		mockAsset(ic, "photo4.jpg", t0.Add(6*time.Hour)),
		mockAsset(ic, "IMG_001.jpg", t0.Add(7*time.Hour)),
		mockAsset(ic, "IMG_005.raw", t0.Add(100*time.Hour)),
		mockAsset(ic, "IMG_005.jpg", t0.Add(101*time.Hour)),
		mockAsset(ic, "IMG_006.heic", t0.Add(110*time.Hour)),
		mockAsset(ic, "photo5.jpg", t0.Add(120*time.Hour)),
		mockAsset(ic, "photo6.jpg", t0.Add(130*time.Hour)),
		mockAsset(ic, "IMG_030.mp4", t0.Add(140*time.Hour)),
		mockAsset(ic, "IMG_030.mov", t0.Add(150*time.Hour)),
		mockAsset(ic, "IMG_030.jpg", t0.Add(160*time.Hour)),
		mockAsset(ic, "IMG_030.heic", t0.Add(170*time.Hour)),
	}

	// inject assets in the input channel
	in := make(chan *assets.Asset)
	go func() {
		for _, a := range testAssets {
			in <- a
		}
		close(in)
	}()

	// collect the outputs in gotGroups and gotAssets
	var gotGroups []*assets.Group
	var gotAssets []*assets.Asset
	ctx := context.Background()

	gOut := groups.NewGrouperPipeline(ctx, burst.Group, series.Group).PipeGrouper(ctx, in)
	for g := range gOut {
		switch g.Grouping {
		case assets.GroupByNone:
			gotAssets = append(gotAssets, g.Assets...)
		default:
			gotGroups = append(gotGroups, g)
		}
	}

	sortGroupFn := func(s []*assets.Group) func(i, j int) bool {
		return func(i, j int) bool {
			if s[i].Assets[0].Radical == s[j].Assets[0].Radical {
				return s[i].Assets[0].CaptureDate.Before(s[j].Assets[0].CaptureDate)
			}
			return s[i].Assets[0].Radical < s[j].Assets[0].Radical
		}
	}

	sort.Slice(expectedGroup, sortGroupFn(expectedGroup))
	sort.Slice(gotGroups, sortGroupFn(gotGroups))
	if len(gotGroups) != len(expectedGroup) {
		t.Errorf("Expected %d group, got %d", len(expectedGroup), len(gotGroups))
	} else {
		for i := range gotGroups {
			for j := range gotGroups[i].Assets {
				got := gotGroups[i].Assets[j]
				expected := expectedGroup[i].Assets[j]
				if !reflect.DeepEqual(got, expected) {
					t.Errorf("Expected group %d asset %d \n%#v got\n%#v", i, j, expected, got)
				}
			}
		}
	}

	sortAssetFn := func(s []*assets.Asset) func(i, j int) bool {
		return func(i, j int) bool {
			if s[i].Radical == s[j].Radical {
				if s[i].Index == s[j].Index {
					return s[i].CaptureDate.Before(s[j].CaptureDate)
				}
				return s[i].Index < s[j].Index
			}
			return s[i].Radical < s[j].Radical
		}
	}

	sort.Slice(expectedAssets, sortAssetFn(expectedAssets))
	sort.Slice(gotAssets, sortAssetFn(gotAssets))
	if len(gotAssets) != len(expectedAssets) {
		t.Errorf("Expected %d assets, got %d", len(expectedAssets), len(gotAssets))
	} else {
		for i := range gotAssets {
			if !reflect.DeepEqual(gotAssets[i], expectedAssets[i]) {
				t.Errorf("Expected asset \n%#v got asset \n%#v", expectedAssets[i], gotAssets[i])
			}
		}
	}
}
