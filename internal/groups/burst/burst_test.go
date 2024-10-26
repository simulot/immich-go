package burst

import (
	"context"
	"reflect"
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

func TestGroup(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, metadata.DefaultSupportedMedia)

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	// Create assets with a DateTaken interval of 200 milliseconds
	assets := []groups.Asset{
		mockAsset(ic, "IMG_001.jpg", baseTime),
		mockAsset(ic, "IMG_002.jpg", baseTime.Add(200*time.Millisecond)),
		mockAsset(ic, "IMG_003.jpg", baseTime.Add(400*time.Millisecond)),
		mockAsset(ic, "IMG_004.jpg", baseTime.Add(600*time.Millisecond)),
		mockAsset(ic, "IMG_005.jpg", baseTime.Add(800*time.Millisecond)),
		mockAsset(ic, "IMG_006.jpg", baseTime.Add(1000*time.Millisecond)),
		mockAsset(ic, "IMG_007.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "IMG_008.jpg", baseTime.Add(1400*time.Millisecond)),
		mockAsset(ic, "IMG_009.jpg", baseTime.Add(1600*time.Millisecond)),
		mockAsset(ic, "IMG_010.jpg", baseTime.Add(5*time.Second)),
		mockAsset(ic, "IMG_011.jpg", baseTime.Add(10*time.Second)),
		mockAsset(ic, "IMG_012.jpg", baseTime.Add(10*time.Second+200*time.Millisecond)),
		mockAsset(ic, "IMG_013.jpg", baseTime.Add(10*time.Second+400*time.Millisecond)),
		mockAsset(ic, "IMG_014.jpg", baseTime.Add(15*time.Second)),
		mockAsset(ic, "IMG_015.jpg", baseTime.Add(20*time.Second)),
		mockAsset(ic, "IMG_016.jpg", baseTime.Add(30*time.Second)),
		mockAsset(ic, "IMG_017.jpg", baseTime.Add(30*time.Second+200*time.Millisecond)),
		mockAsset(ic, "IMG_018.jpg", baseTime.Add(30*time.Second+400*time.Millisecond)),
	}

	expectedAssets := []groups.Asset{
		mockAsset(ic, "IMG_010.jpg", baseTime.Add(5*time.Second)),
		mockAsset(ic, "IMG_014.jpg", baseTime.Add(15*time.Second)),
		mockAsset(ic, "IMG_015.jpg", baseTime.Add(20*time.Second)),
	}

	expectedGroup := []*groups.AssetGroup{
		groups.NewAssetGroup(groupby.GroupByBurst,
			mockAsset(ic, "IMG_001.jpg", baseTime),
			mockAsset(ic, "IMG_002.jpg", baseTime.Add(200*time.Millisecond)),
			mockAsset(ic, "IMG_003.jpg", baseTime.Add(400*time.Millisecond)),
			mockAsset(ic, "IMG_004.jpg", baseTime.Add(600*time.Millisecond)),
			mockAsset(ic, "IMG_005.jpg", baseTime.Add(800*time.Millisecond)),
			mockAsset(ic, "IMG_006.jpg", baseTime.Add(1000*time.Millisecond)),
			mockAsset(ic, "IMG_007.jpg", baseTime.Add(1200*time.Millisecond)),
			mockAsset(ic, "IMG_008.jpg", baseTime.Add(1400*time.Millisecond)),
			mockAsset(ic, "IMG_009.jpg", baseTime.Add(1600*time.Millisecond)),
		),
		groups.NewAssetGroup(groupby.GroupByBurst,
			mockAsset(ic, "IMG_011.jpg", baseTime.Add(10*time.Second)),
			mockAsset(ic, "IMG_012.jpg", baseTime.Add(10*time.Second+200*time.Millisecond)),
			mockAsset(ic, "IMG_013.jpg", baseTime.Add(10*time.Second+400*time.Millisecond)),
		),
		groups.NewAssetGroup(groupby.GroupByBurst,
			mockAsset(ic, "IMG_016.jpg", baseTime.Add(30*time.Second)),
			mockAsset(ic, "IMG_017.jpg", baseTime.Add(30*time.Second+200*time.Millisecond)),
			mockAsset(ic, "IMG_018.jpg", baseTime.Add(30*time.Second+400*time.Millisecond)),
		),
	}

	in := make(chan groups.Asset, len(assets))
	out := make(chan groups.Asset)
	gOut := make(chan *groups.AssetGroup)

	go func() {
		Group(ctx, in, out, gOut)
		close(out)
		close(gOut)
	}()

	for _, asset := range assets {
		in <- asset
	}
	close(in)

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
