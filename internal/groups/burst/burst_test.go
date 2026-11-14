package burst

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
	// Create assets with a DateTaken interval of 200 milliseconds
	testAssets := []*assets.Asset{
		mockAsset(ic, "IMG_001.jpg", baseTime),
		mockAsset(ic, "IMG_002.jpg", baseTime.Add(200*time.Millisecond)),  // group 1
		mockAsset(ic, "IMG_003.jpg", baseTime.Add(400*time.Millisecond)),  // group 1
		mockAsset(ic, "IMG_004.jpg", baseTime.Add(600*time.Millisecond)),  // group 1
		mockAsset(ic, "IMG_005.jpg", baseTime.Add(800*time.Millisecond)),  // group 1
		mockAsset(ic, "IMG_006.jpg", baseTime.Add(1000*time.Millisecond)), // group 1
		mockAsset(ic, "IMG_007.jpg", baseTime.Add(1200*time.Millisecond)), // group 1
		mockAsset(ic, "IMG_008.jpg", baseTime.Add(1400*time.Millisecond)), // group 1
		mockAsset(ic, "IMG_009.jpg", baseTime.Add(1600*time.Millisecond)),
		mockAsset(ic, "IMG_010.jpg", baseTime.Add(5*time.Second)),
		mockAsset(ic, "IMG_011.jpg", baseTime.Add(10*time.Second)),
		mockAsset(ic, "IMG_012.jpg", baseTime.Add(10*time.Second+200*time.Millisecond)), // group 2
		mockAsset(ic, "IMG_013.jpg", baseTime.Add(10*time.Second+400*time.Millisecond)), // group 2
		mockAsset(ic, "IMG_014.jpg", baseTime.Add(15*time.Second)),
		mockAsset(ic, "IMG_015.jpg", baseTime.Add(20*time.Second)),
		mockAsset(ic, "IMG_016.jpg", baseTime.Add(30*time.Second)),
		mockAsset(ic, "IMG_017.jpg", baseTime.Add(30*time.Second+200*time.Millisecond)), // group 3
		mockAsset(ic, "IMG_018.jpg", baseTime.Add(30*time.Second+400*time.Millisecond)), // group 3
	}

	expectedAssets := []*assets.Asset{
		mockAsset(ic, "IMG_010.jpg", baseTime.Add(5*time.Second)),
		mockAsset(ic, "IMG_014.jpg", baseTime.Add(15*time.Second)),
		mockAsset(ic, "IMG_015.jpg", baseTime.Add(20*time.Second)),
	}

	expectedGroup := []*assets.Group{
		assets.NewGroup(assets.GroupByBurst,
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
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "IMG_011.jpg", baseTime.Add(10*time.Second)),
			mockAsset(ic, "IMG_012.jpg", baseTime.Add(10*time.Second+200*time.Millisecond)),
			mockAsset(ic, "IMG_013.jpg", baseTime.Add(10*time.Second+400*time.Millisecond)),
		),
		assets.NewGroup(assets.GroupByBurst,
			mockAsset(ic, "IMG_016.jpg", baseTime.Add(30*time.Second)),
			mockAsset(ic, "IMG_017.jpg", baseTime.Add(30*time.Second+200*time.Millisecond)),
			mockAsset(ic, "IMG_018.jpg", baseTime.Add(30*time.Second+400*time.Millisecond)),
		),
	}

	in := make(chan *assets.Asset, len(testAssets))
	out := make(chan *assets.Asset)
	gOut := make(chan *assets.Group)

	go func() {
		Group(ctx, in, out, gOut)
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
		t.Errorf("Expected %d assets, got %d", len(expectedAssets), len(gotAssets))
	} else {
		for i := range gotAssets {
			if !reflect.DeepEqual(gotAssets[i], expectedAssets[i]) {
				t.Errorf("Expected asset \n%#v got asset \n%#v", expectedAssets[i], gotAssets[i])
			}
		}
	}
}

// A video, like the clip of a motion photo, is never part of a burst.
func TestGroupWithVideos(t *testing.T) {
	ctx := context.Background()
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	baseTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name           string
		assets         []*assets.Asset
		expectedGroups [][]string
		expectedAssets []string
	}{
		{
			name: "motion photo",
			assets: []*assets.Asset{
				mockAsset(ic, "IMG_001.MP.mp4", baseTime),
				mockAsset(ic, "IMG_001.MP.jpg", baseTime.Add(300*time.Millisecond)),
			},
			expectedAssets: []string{"IMG_001.MP.mp4", "IMG_001.MP.jpg"},
		},
		{
			name: "burst starting with a motion photo",
			assets: []*assets.Asset{
				mockAsset(ic, "IMG_001.MP.mp4", baseTime),
				mockAsset(ic, "IMG_001.MP.jpg", baseTime.Add(300*time.Millisecond)),
				mockAsset(ic, "IMG_002.jpg", baseTime.Add(600*time.Millisecond)),
			},
			expectedGroups: [][]string{{"IMG_001.MP.jpg", "IMG_002.jpg"}},
			expectedAssets: []string{"IMG_001.MP.mp4"},
		},
		{
			name: "video between two images",
			assets: []*assets.Asset{
				mockAsset(ic, "IMG_001.jpg", baseTime),
				mockAsset(ic, "IMG_002.mp4", baseTime.Add(100*time.Millisecond)),
				mockAsset(ic, "IMG_003.jpg", baseTime.Add(200*time.Millisecond)),
			},
			expectedAssets: []string{"IMG_001.jpg", "IMG_002.mp4", "IMG_003.jpg"},
		},
		{
			name: "video in a burst",
			assets: []*assets.Asset{
				mockAsset(ic, "IMG_001.jpg", baseTime),
				mockAsset(ic, "IMG_002.jpg", baseTime.Add(200*time.Millisecond)),
				mockAsset(ic, "IMG_003.mp4", baseTime.Add(300*time.Millisecond)),
				mockAsset(ic, "IMG_004.jpg", baseTime.Add(400*time.Millisecond)),
				mockAsset(ic, "IMG_005.jpg", baseTime.Add(600*time.Millisecond)),
			},
			expectedGroups: [][]string{{"IMG_001.jpg", "IMG_002.jpg"}, {"IMG_004.jpg", "IMG_005.jpg"}},
			expectedAssets: []string{"IMG_003.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan *assets.Asset, len(tt.assets))
			out := make(chan *assets.Asset, len(tt.assets))
			gOut := make(chan *assets.Group, len(tt.assets))
			for _, a := range tt.assets {
				in <- a
			}
			close(in)
			Group(ctx, in, out, gOut)
			close(out)
			close(gOut)

			gotGroups := [][]string{}
			for g := range gOut {
				names := []string{}
				for _, a := range g.Assets {
					names = append(names, a.File.Name())
				}
				gotGroups = append(gotGroups, names)
			}
			gotAssets := []string{}
			for a := range out {
				gotAssets = append(gotAssets, a.File.Name())
			}

			if tt.expectedGroups == nil {
				tt.expectedGroups = [][]string{}
			}
			if !reflect.DeepEqual(gotGroups, tt.expectedGroups) {
				t.Errorf("Expected groups %v, got %v", tt.expectedGroups, gotGroups)
			}
			if !reflect.DeepEqual(gotAssets, tt.expectedAssets) {
				t.Errorf("Expected assets %v, got %v", tt.expectedAssets, gotAssets)
			}
		})
	}
}
