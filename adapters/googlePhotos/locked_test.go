package gp

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/fshelper"
)

func TestLockedFolderMetadata(t *testing.T) {
	tcs := []struct {
		name           string
		json           string
		wantVisibility assets.Visibility
		wantArchived   bool
	}{
		{
			name:           "regular",
			json:           `{"title":"a.jpg","photoTakenTime":{"timestamp":"1695394176"}}`,
			wantVisibility: assets.VisibilityUnknown,
		},
		{
			name:           "archived",
			json:           `{"title":"a.jpg","archived":true,"photoTakenTime":{"timestamp":"1695394176"}}`,
			wantVisibility: assets.VisibilityArchive,
			wantArchived:   true,
		},
		{
			name:           "locked",
			json:           `{"title":"a.jpg","inLockedFolder":true,"photoTakenTime":{"timestamp":"1695394176"}}`,
			wantVisibility: assets.VisibilityLocked,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var gmd GoogleMetaData
			if err := json.Unmarshal([]byte(tc.json), &gmd); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			md := gmd.AsMetadata(fshelper.FSName(fstest.MapFS{}, "a.jpg.json"), false)
			if md.Visibility != tc.wantVisibility {
				t.Errorf("visibility: got %q, want %q", md.Visibility, tc.wantVisibility)
			}
			if md.Archived != tc.wantArchived {
				t.Errorf("archived: got %v, want %v", md.Archived, tc.wantArchived)
			}

			a := &assets.Asset{}
			a.UseMetadata(md)
			if a.Visibility != tc.wantVisibility {
				t.Errorf("asset visibility: got %q, want %q", a.Visibility, tc.wantVisibility)
			}
		})
	}
}

func TestFilterOnMetadataLockedAndArchived(t *testing.T) {
	tcs := []struct {
		name        string
		keepLocked  bool
		keepArchive bool
		visibility  assets.Visibility
		archived    bool
		wantDropped bool
	}{
		{name: "locked kept", keepLocked: true, keepArchive: true, visibility: assets.VisibilityLocked},
		{name: "locked discarded", keepLocked: false, keepArchive: true, visibility: assets.VisibilityLocked, wantDropped: true},
		{name: "archived kept", keepLocked: true, keepArchive: true, visibility: assets.VisibilityArchive, archived: true},
		{name: "archived discarded", keepLocked: true, keepArchive: false, visibility: assets.VisibilityArchive, archived: true, wantDropped: true},
		{name: "locked not affected by archived flag", keepLocked: true, keepArchive: false, visibility: assets.VisibilityLocked},
		{name: "regular not affected", keepLocked: false, keepArchive: false},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			rec := fileevent.NewRecorder(slog.New(slog.DiscardHandler))
			toc := &TakeoutCmd{
				processor:    fileprocessor.New(assettracker.New(), rec),
				KeepLocked:   tc.keepLocked,
				KeepArchived: tc.keepArchive,
				KeepPartner:  true,
				KeepTrashed:  true,
			}
			a := &assets.Asset{
				File:       fshelper.FSName(fstest.MapFS{}, "a.jpg"),
				Visibility: tc.visibility,
				Archived:   tc.archived,
			}
			toc.processor.RecordAssetDiscovered(ctx, a.File, 0, fileevent.DiscoveredImage)

			code := toc.filterOnMetadata(ctx, a)
			dropped := code == fileevent.DiscardedFiltered
			if dropped != tc.wantDropped {
				t.Errorf("discarded: got %v, want %v", dropped, tc.wantDropped)
			}
		})
	}
}
