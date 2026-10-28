//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/immich"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// A Picasa album file (.picasa.ini) is not a media type. Before the fix, the
// useless file filter ran before the Picasa check, so the file was recorded as
// an unknown file and the album name was lost. The Picasa check must run first.
//
// Test data: DATA/fromPicasa/album holds a .picasa.ini file with a [Picasa]
// section (name=Concert, description=Live photos) and two jpg photos.
func Test_FromPicasa(t *testing.T) {
	t.Run("picasa-album", func(t *testing.T) {
		adm, err := getUser("admin@immich.app")
		if err != nil {
			t.Fatalf("can't get admin user: %v", err)
		}
		// A fresh user for a new test
		u1, err := createUser("minimal")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			// "--concurrent-tasks=0", // for debugging
			"upload", "from-picasa",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromPicasa/album",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		// The .picasa.ini file is read as album metadata, not as a useless file.
		e2eutils.CheckResults(t, map[fileevent.Code]int64{
			fileevent.DiscoveredMetadata:     1,
			fileevent.DiscoveredUnknown:      0,
			fileevent.ProcessedUploadSuccess: 2,
			fileevent.ProcessedAlbumAdded:    2,
		}, false, a.FileProcessor())

		// The album name and description come from the [Picasa] section.
		ic, err := immich.NewImmichClient(ImmichURL, u1.APIKey)
		if err != nil {
			t.Fatalf("can't create the client: %v", err)
		}
		albums, err := ic.GetAllAlbums(ctx)
		if err != nil {
			t.Fatalf("can't get the user's albums: %v", err)
		}
		if len(albums) != 1 {
			t.Errorf("expected one album, got %d: %v", len(albums), albums)
			return
		}
		if albums[0].AlbumName != "Concert" || albums[0].Description != "Live photos" {
			t.Errorf("expected album [Concert] with description [Live photos], got %v", albums[0])
		}
	})

	// The same data with the Picasa option off. The file must still be recorded
	// as a useless file, and no album must be created. This checks the command
	// wiring: from-picasa sets --album-picasa to true by default.
	t.Run("picasa-album-off", func(t *testing.T) {
		adm, err := getUser("admin@immich.app")
		if err != nil {
			t.Fatalf("can't get admin user: %v", err)
		}
		// A fresh user for a new test
		u1, err := createUser("minimal")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			// "--concurrent-tasks=0", // for debugging
			"upload", "from-picasa",
			"--album-picasa=false",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromPicasa/album",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2eutils.CheckResults(t, map[fileevent.Code]int64{
			fileevent.DiscoveredMetadata:     0,
			fileevent.DiscoveredUnknown:      1,
			fileevent.ProcessedUploadSuccess: 2,
			fileevent.ProcessedAlbumAdded:    0,
		}, false, a.FileProcessor())
	})
}
