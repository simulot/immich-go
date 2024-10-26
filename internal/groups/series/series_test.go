package series

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/groupby"
	"github.com/simulot/immich-go/internal/metadata"
)

type mockedAsset struct {
	nameInfo  filenames.NameInfo
	dateTaken time.Time
}

func (m mockedAsset) NameInfo() filenames.NameInfo {
	return m.nameInfo
}

func (m mockedAsset) DateTaken() time.Time {
	return m.dateTaken
}

func mockAsset(ic *filenames.InfoCollector, name string, dateTaken time.Time) groups.Asset {
	return mockedAsset{
		nameInfo:  ic.GetInfo(name),
		dateTaken: dateTaken,
	}
}

func sortAssetFn(s []groups.Asset) func(i, j int) bool {
	return func(i, j int) bool {
		if s[i].NameInfo().Radical == s[j].NameInfo().Radical {
			if s[i].NameInfo().Index == s[j].NameInfo().Index {
				return s[i].DateTaken().Before(s[j].DateTaken())
			}
			return s[i].NameInfo().Index < s[j].NameInfo().Index
		}
		return s[i].NameInfo().Radical < s[j].NameInfo().Radical
	}
}

func sortGroupFn(s []*groups.AssetGroup) func(i, j int) bool {
	return func(i, j int) bool {
		if s[i].Assets[0].NameInfo().Radical == s[j].Assets[0].NameInfo().Radical {
			return s[i].Assets[0].DateTaken().Before(s[j].Assets[0].DateTaken())
		}
		return s[i].Assets[0].NameInfo().Radical < s[j].Assets[0].NameInfo().Radical
	}
}

func TestGroup(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, metadata.DefaultSupportedMedia)
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)

	assets := []groups.Asset{
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
	}

	expectedAssets := []groups.Asset{
		mockAsset(ic, "IMG_0001.jpg", baseTime),
		mockAsset(ic, "IMG_0005.raw", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0006.heic", baseTime.Add(4*time.Hour)),
		mockAsset(ic, "IMG_0007.raw", baseTime.Add(5*time.Hour)),
		mockAsset(ic, "IMG_0007.jpg", baseTime.Add(6*time.Hour)),
	}

	expectedGroup := []*groups.AssetGroup{
		groups.NewAssetGroup(groupby.GroupByBurst,
			mockAsset(ic, "IMG_20231014_183246_BURST001_COVER.jpg", baseTime.Add(1*time.Hour)), // group 1
			mockAsset(ic, "IMG_20231014_183246_BURST002.jpg", baseTime.Add(1*time.Hour)),       // group 1
			mockAsset(ic, "IMG_20231014_183246_BURST003.jpg", baseTime.Add(1*time.Hour)),       // group 1
		),
		groups.NewAssetGroup(groupby.GroupByRawJpg,
			mockAsset(ic, "IMG_0003.jpg", baseTime.Add(2*time.Hour)),
			mockAsset(ic, "IMG_0003.raw", baseTime.Add(2*time.Hour)),
		),
		groups.NewAssetGroup(groupby.GroupByHeicJpg,
			mockAsset(ic, "IMG_0004.heic", baseTime.Add(3*time.Hour)),
			mockAsset(ic, "IMG_0004.jpg", baseTime.Add(3*time.Hour)),
		),
	}

	sort.Slice(assets, sortAssetFn(assets))
	sort.Slice(expectedGroup, sortGroupFn(expectedGroup))
	source := make(chan groups.Asset, len(assets))
	out := make(chan groups.Asset)
	gOut := make(chan *groups.AssetGroup)

	go func() {
		for _, asset := range assets {
			source <- asset
		}
		close(source)
	}()

	go func() {
		Group(ctx, source, out, gOut)
		close(out)
		close(gOut)
	}()

	gotGroups := []*groups.AssetGroup{}
	gotAssets := []groups.Asset{}

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
				got := gotGroups[i].Assets[j].(mockedAsset)
				expected := expectedGroup[i].Assets[j].(mockedAsset)
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
