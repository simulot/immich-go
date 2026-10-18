package immich

import (
	"testing"
	"testing/fstest"

	"github.com/simulot/immich-go/internal/assets"
)

func TestPrepareCallValuesVisibility(t *testing.T) {
	fsys := fstest.MapFS{"a.jpg": &fstest.MapFile{Data: []byte("x")}}
	info, err := fsys.Stat("a.jpg")
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		name  string
		asset assets.Asset
		want  string
	}{
		{name: "default", asset: assets.Asset{}, want: "timeline"},
		{name: "archived", asset: assets.Asset{Archived: true}, want: "archive"},
		{name: "locked", asset: assets.Asset{Visibility: assets.VisibilityLocked}, want: "locked"},
		{name: "locked wins over archived", asset: assets.Asset{Visibility: assets.VisibilityLocked, Archived: true}, want: "locked"},
		{name: "other visibilities are not propagated", asset: assets.Asset{Visibility: assets.VisibilityHidden}, want: "timeline"},
	}
	ic := &ImmichClient{}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.asset.OriginalFileName = "a.jpg"
			got := ic.prepareCallValues(&tc.asset, info, ".jpg", "image")["visibility"]
			if got != tc.want {
				t.Errorf("visibility: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAsAssetVisibility(t *testing.T) {
	ia := Asset{ID: "id", Visibility: "locked"}
	if got := ia.AsAsset().Visibility; got != assets.VisibilityLocked {
		t.Errorf("got %q, want %q", got, assets.VisibilityLocked)
	}
}
