package series

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

func sortAssetFn(s []*assets.Asset) func(i, j int) bool {
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

func sortGroupFn(s []*assets.Group) func(i, j int) bool {
	return func(i, j int) bool {
		if s[i].Assets[0].Radical == s[j].Assets[0].Radical {
			return s[i].Assets[0].CaptureDate.Before(s[j].Assets[0].CaptureDate)
		}
		return s[i].Assets[0].Radical < s[j].Assets[0].Radical
	}
}

func TestGroup(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)

	as := []*assets.Asset{
		mockAsset(ic, "IMG_0001.jpg", baseTime),
		mockAsset(ic, "IMG_20231014_183246_BURST001_COVER.jpg", baseTime.Add(1*time.Hour)), // group 1
		mockAsset(ic, "IMG_20231014_183246_BURST002.jpg", baseTime.Add(1*time.Hour)),       // group 1
		mockAsset(ic, "IMG_20231014_183246_BURST003.jpg", baseTime.Add(1*time.Hour)),       // group 1
		mockAsset(ic, "IMG_0003.jpg", baseTime.Add(2*time.Hour)),                           // group 2
		mockAsset(ic, "IMG_0003.raw", baseTime.Add(2*time.Hour)),                           // group 2
		mockAsset(ic, "IMG_0004.heic", baseTime.Add(3*time.Hour)),                          // group 3
		mockAsset(ic, "IMG_0004.jpg", baseTime.Add(3*time.Hour)),                           // group 3
		mockAsset(ic, "IMG_0005.raw", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0006.heic", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0007.raw", baseTime.Add(5*time.Hour)),
		mockAsset(ic, "IMG_0007.jpg", baseTime.Add(6*time.Hour)),
		mockAsset(ic, "IMG_030.mp4", baseTime.Add(140*time.Hour)),
		mockAsset(ic, "IMG_030.mov", baseTime.Add(150*time.Hour)),
		mockAsset(ic, "IMG_030.jpg", baseTime.Add(160*time.Hour)),
		mockAsset(ic, "IMG_030.heic", baseTime.Add(170*time.Hour)),
	}

	expectedAssets := []*assets.Asset{
		mockAsset(ic, "IMG_0001.jpg", baseTime),
		mockAsset(ic, "IMG_0005.raw", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0006.heic", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0007.raw", baseTime.Add(5*time.Hour)),
		mockAsset(ic, "IMG_0007.jpg", baseTime.Add(6*time.Hour)),
		mockAsset(ic, "IMG_030.mp4", baseTime.Add(140*time.Hour)),
		mockAsset(ic, "IMG_030.mov", baseTime.Add(150*time.Hour)),
		mockAsset(ic, "IMG_030.jpg", baseTime.Add(160*time.Hour)),
		mockAsset(ic, "IMG_030.heic", baseTime.Add(170*time.Hour)),
	}

	expectedGroup := []*assets.Group{
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "IMG_20231014_183246_BURST001_COVER.jpg", baseTime.Add(1*time.Hour)), // group 1
			mockAsset(ic, "IMG_20231014_183246_BURST002.jpg", baseTime.Add(1*time.Hour)),       // group 1
			mockAsset(ic, "IMG_20231014_183246_BURST003.jpg", baseTime.Add(1*time.Hour)),       // group 1
		),
		assets.NewGroup(assets.GroupByRawJpg,
			mockAsset(ic, "IMG_0003.raw", baseTime.Add(2*time.Hour)),
			mockAsset(ic, "IMG_0003.jpg", baseTime.Add(2*time.Hour)),
		),
		assets.NewGroup(assets.GroupByHeicJpg,
			mockAsset(ic, "IMG_0004.jpg", baseTime.Add(3*time.Hour)),
			mockAsset(ic, "IMG_0004.heic", baseTime.Add(3*time.Hour)),
		),
	}

	sort.Slice(as, sortAssetFn(as))
	sort.Slice(expectedGroup, sortGroupFn(expectedGroup))
	source := make(chan *assets.Asset, len(as))
	out := make(chan *assets.Asset)
	gOut := make(chan *assets.Group)

	go func() {
		for _, asset := range as {
			source <- asset
		}
		close(source)
	}()

	go func() {
		Group(ctx, source, out, gOut)
		close(out)
		close(gOut)
	}()

	gotGroups := []*assets.Group{}
	gotAssets := []*assets.Asset{}

	doneGroup := false
	doneAsset := false
	for !doneGroup || !doneAsset {
		select {
		case group, ok := <-gOut:
			if !ok {
				doneGroup = true
				continue
			}
			gotGroups = append(gotGroups, group)
		case asset, ok := <-out:
			if !ok {
				doneAsset = true
				continue
			}
			gotAssets = append(gotAssets, asset)
		}
	}

	sort.Slice(gotGroups, sortGroupFn(gotGroups))

	if len(gotGroups) != len(expectedGroup) {
		t.Errorf("Expected %d groups, got %d", len(expectedGroup), len(gotGroups))
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
