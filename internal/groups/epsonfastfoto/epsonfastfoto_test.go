package epsonfastfoto

import (
	"context"
	"reflect"
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

func TestGroup(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)

	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	testAssets := []*assets.Asset{
		mockAsset(ic, "SceneryAndWildlife_0001_a.jpg", baseTime),
		mockAsset(ic, "SceneryAndWildlife_0001_b.jpg", baseTime.Add(200*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0001.jpg", baseTime.Add(400*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0002_a.jpg", baseTime.Add(600*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0002_b.jpg", baseTime.Add(800*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0002.jpg", baseTime.Add(1000*time.Millisecond)),
		mockAsset(ic, "img_0001.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "img_0002.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0003_a.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0003.jpg", baseTime.Add(1400*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0004_a.jpg", baseTime.Add(1600*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0004.jpg", baseTime.Add(1800*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0005_a.jpg", baseTime.Add(2000*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0005_b.jpg", baseTime.Add(2200*time.Millisecond)),
		mockAsset(ic, "SceneryAndWildlife_0005.jpg", baseTime.Add(2400*time.Millisecond)),
		mockAsset(ic, "img_0005.jpg", baseTime.Add(1200*time.Millisecond)),
	}

	expectedAssets := []*assets.Asset{
		mockAsset(ic, "img_0001.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "img_0002.jpg", baseTime.Add(1200*time.Millisecond)),
		mockAsset(ic, "img_0005.jpg", baseTime.Add(1200*time.Millisecond)),
	}

	expectedGroup := []*assets.Group{
		assets.NewGroup(assets.GroupByOther,
			mockAsset(ic, "SceneryAndWildlife_0001_a.jpg", baseTime),
			mockAsset(ic, "SceneryAndWildlife_0001_b.jpg", baseTime.Add(200*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0001.jpg", baseTime.Add(400*time.Millisecond)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByOther,
			mockAsset(ic, "SceneryAndWildlife_0002_a.jpg", baseTime.Add(600*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0002_b.jpg", baseTime.Add(800*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0002.jpg", baseTime.Add(1000*time.Millisecond)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByOther,
			mockAsset(ic, "SceneryAndWildlife_0003_a.jpg", baseTime.Add(1200*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0003.jpg", baseTime.Add(1400*time.Millisecond)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByOther,
			mockAsset(ic, "SceneryAndWildlife_0004_a.jpg", baseTime.Add(1600*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0004.jpg", baseTime.Add(1800*time.Millisecond)),
		).SetCover(0),
		assets.NewGroup(assets.GroupByOther,
			mockAsset(ic, "SceneryAndWildlife_0005_a.jpg", baseTime.Add(2000*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0005_b.jpg", baseTime.Add(2200*time.Millisecond)),
			mockAsset(ic, "SceneryAndWildlife_0005.jpg", baseTime.Add(2400*time.Millisecond)),
		).SetCover(0),
	}

	in := make(chan *assets.Asset, len(testAssets))
	out := make(chan *assets.Asset)
	gOut := make(chan *assets.Group)

	go func() {
		g := &Group{}
		g.Group(ctx, in, out, gOut)
		close(out)
		close(gOut)
	}()

	for _, a := range testAssets {
		in <- a
	}
	close(in)

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
		t.Errorf("Expected 0 assets, got %d", len(gotAssets))
	} else {
		for i := range gotAssets {
			if !reflect.DeepEqual(gotAssets[i], expectedAssets[i]) {
				t.Errorf("Expected asset \n%#v got asset \n%#v", expectedAssets[i], gotAssets[i])
			}
		}
	}
}
