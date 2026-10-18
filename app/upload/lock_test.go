package upload

import (
	"context"
	"errors"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

type fakeLockClient struct {
	albums    []immich.AlbumSimplified
	albumsErr error
	updateErr error

	albumCalls int
	updates    []assets.Visibility
	updatedIDs []string
}

func (f *fakeLockClient) GetAssetAlbums(_ context.Context, _ string) ([]immich.AlbumSimplified, error) {
	f.albumCalls++
	return f.albums, f.albumsErr
}

func (f *fakeLockClient) SetAssetsVisibility(_ context.Context, ids []string, v assets.Visibility) error {
	f.updatedIDs = append(f.updatedIDs, ids...)
	f.updates = append(f.updates, v)
	return f.updateErr
}

func TestLockServerAsset(t *testing.T) {
	locked := &assets.Asset{Visibility: assets.VisibilityLocked}
	regular := &assets.Asset{}

	tcs := []struct {
		name        string
		input       *assets.Asset
		server      *assets.Asset
		client      *fakeLockClient
		want        lockOutcome
		wantUpdate  bool
		wantAlbumQs int
	}{
		{
			name:   "input not locked",
			input:  regular,
			server: &assets.Asset{ID: "s1"},
			client: &fakeLockClient{},
			want:   lockNotNeeded,
		},
		{
			name:   "server asset already locked",
			input:  locked,
			server: &assets.Asset{ID: "s1", Visibility: assets.VisibilityLocked},
			client: &fakeLockClient{},
			want:   lockNotNeeded,
		},
		{
			name:   "no server asset",
			input:  locked,
			server: nil,
			client: &fakeLockClient{},
			want:   lockNotNeeded,
		},
		{
			name:        "locked and no albums",
			input:       locked,
			server:      &assets.Asset{ID: "s1", Visibility: assets.VisibilityTimeline},
			client:      &fakeLockClient{},
			want:        lockApplied,
			wantUpdate:  true,
			wantAlbumQs: 1,
		},
		{
			name:   "server asset in album is left untouched, even if the cache knows no album",
			input:  locked,
			server: &assets.Asset{ID: "s1"},
			client: &fakeLockClient{
				albums: []immich.AlbumSimplified{{ID: "a1", AlbumName: "Holidays"}},
			},
			want:        lockSkipped,
			wantAlbumQs: 1,
		},
		{
			name:        "albums can't be checked",
			input:       locked,
			server:      &assets.Asset{ID: "s1"},
			client:      &fakeLockClient{albumsErr: errors.New("boom")},
			want:        lockSkipped,
			wantAlbumQs: 1,
		},
		{
			name:        "update fails",
			input:       locked,
			server:      &assets.Asset{ID: "s1"},
			client:      &fakeLockClient{updateErr: errors.New("boom")},
			want:        lockSkipped,
			wantUpdate:  true,
			wantAlbumQs: 1,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := lockServerAsset(context.Background(), tc.client, tc.input, tc.server)
			if got != tc.want {
				t.Fatalf("outcome: got %d, want %d (%s)", got, tc.want, reason)
			}
			if got == lockSkipped && reason == "" {
				t.Error("a skipped lock must explain why")
			}
			if tc.client.albumCalls != tc.wantAlbumQs {
				t.Errorf("album checks: got %d, want %d", tc.client.albumCalls, tc.wantAlbumQs)
			}
			if (len(tc.client.updates) > 0) != tc.wantUpdate {
				t.Fatalf("update calls: got %d, want update=%v", len(tc.client.updates), tc.wantUpdate)
			}
			if tc.wantUpdate {
				if tc.client.updates[0] != assets.VisibilityLocked {
					t.Errorf("update visibility: got %q", tc.client.updates[0])
				}
				if tc.client.updatedIDs[0] != tc.server.ID {
					t.Errorf("updated ID: got %q, want %q", tc.client.updatedIDs[0], tc.server.ID)
				}
			}
			if got == lockApplied && tc.server.Visibility != assets.VisibilityLocked {
				t.Error("the index entry must reflect the new visibility")
			}
		})
	}
}
