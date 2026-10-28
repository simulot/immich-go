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

// A file whose name looks like a scan's but that the grouper does not group (a burst frame, a
// video) is passed on, never dropped.
func TestGroupPassesOnWhatItDoesNotGroup(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	t0 := time.Date(2023, 12, 7, 10, 16, 5, 0, time.Local)
	testAssets := []*assets.Asset{
		mockAsset(ic, "20231207_101605_001.jpg", t0), // a Samsung burst frame
		mockAsset(ic, "20231207_101605_002.jpg", t0),
		mockAsset(ic, "clip_0001.mov", t0),
		mockAsset(ic, "photo_0001.jpg", t0),
		mockAsset(ic, "photo_0001_a.jpg", t0),
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

	var passed, grouped []string
	for out != nil || gOut != nil {
		select {
		case a, ok := <-out:
			if !ok {
				out = nil
				continue
			}
			passed = append(passed, a.File.Name())
		case g, ok := <-gOut:
			if !ok {
				gOut = nil
				continue
			}
			for _, a := range g.Assets {
				grouped = append(grouped, a.File.Name())
			}
		}
	}
	if want := []string{"20231207_101605_001.jpg", "20231207_101605_002.jpg", "clip_0001.mov"}; !reflect.DeepEqual(passed, want) {
		t.Errorf("passed on %v, want %v", passed, want)
	}
	if want := []string{"photo_0001.jpg", "photo_0001_a.jpg"}; !reflect.DeepEqual(grouped, want) {
		t.Errorf("grouped %v, want %v", grouped, want)
	}
}
