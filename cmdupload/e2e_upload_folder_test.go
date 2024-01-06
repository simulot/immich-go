//go:build e2e
// +build e2e

package cmdupload

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

type testCase struct {
	name        string
	args        []string
	resetImmich bool
	APITrace    bool
	expectError bool
}

func runCase(t *testing.T, tc testCase) {

	var myEnv map[string]string
	myEnv, err := godotenv.Read("../.env")
	if err != nil {
		t.Errorf("cant initialize environment variables: %s", err)
		return
	}

	host := myEnv["IMMICH_HOST"]
	if host == "" {
		host = "http://localhost:2283"
	}
	key := myEnv["IMMICH_KEY"]
	if key == "" {
		t.Fatal("you must provide the IMMICH's API KEY in the environnement variable DEBUG_IMMICH_KEY")
	}

	user := myEnv["IMMICH_DEBUGUSER"]
	if user == "" {
		user = "debug.example.com"
	}

	lf, err := os.Create(tc.name + ".log")
	if err != nil {
		t.Error(err)
		return
	}
	defer lf.Close()
	jnl := logger.NewJournal(logger.NewLogger(logger.Info, true, false).SetWriter(lf))

	ic, err := immich.NewImmichClient(host, key, false)
	if err != nil {
		t.Error(err)
		return
	}

	if tc.APITrace {
		ic.EnableAppTrace(true)
	}
	ctx := context.Background()

	if tc.resetImmich {
		err := resetImmich(ic, user)
		if err != nil {
			t.Error(err)
			return
		}
	}
	err = UploadCommand(ctx, ic, jnl, tc.args)
	if (tc.expectError && err == nil) || (!tc.expectError && err != nil) {
		t.Errorf("unexpected err: %v", err)
		return
	}

}

func TestE2eUpload(t *testing.T) {

	tests := []testCase{
		{
			name: "upload folder",
			args: []string{
				"../../test-data/low_high/high",
			},
			resetImmich: true,

			expectError: false,
		},
		{
			name: "upload folder",
			args: []string{
				"../../test-data/low_high/high",
			},

			// resetImmich: true,
			expectError: false,
		},
		{
			name: "upload folder *.jpg",
			args: []string{
				"-google-photos",
				"../../test-data/test_folder/*.jpg",
			},

			resetImmich: true,
			expectError: true,
		},
		{
			name: "upload folder *.jpg",
			args: []string{
				"../../test-data/test_folder/*/*.jpg",
			},

			// resetImmich: true,
			expectError: false,
		},

		{
			name: "upload folder *.jpg - dry run",
			args: []string{
				"-dry-run",
				"../../test-data/full_takeout (copy)/Takeout/Google Photos/Photos from 2023",
			},

			// resetImmich: true,
			expectError: false,
		},
		{
			name: "upload google photos",
			args: []string{
				"-google-photos",
				"../../test-data/low_high/Takeout",
			},
			// resetImmich: true,
			expectError: false,
		},
		{
			name: "upload burst Huawei",
			args: []string{
				"-stack-burst=FALSE",
				"-stack-jpg-raw=TRUE",
				"../../test-data/burst/Tel",
			},
			resetImmich: true,
			expectError: false,
		},
	}
	for _, tc := range tests {
		runCase(t, tc)
	}
}

// PXL_20231006_063536303 should be archived
// Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg.json is favorite and has a description
func Test_DescriptionAndFavorite(t *testing.T) {

	tc := testCase{
		name: "Test_DescriptionAndFavorite",
		args: []string{
			"-google-photos",
			"-discard-archived",
			"TEST_DATA/Takeout1",
		},
		resetImmich: true,
		expectError: false,
	}
	runCase(t, tc)
}

func Test_PermissionError(t *testing.T) {

	tc := testCase{
		name: "Test_PermissionError",
		args: []string{
			"../../test-data/low_high/high",
		},
		resetImmich: true,
		expectError: false,
	}
	runCase(t, tc)
}

func Test_CreateAlbumFolder(t *testing.T) {

	tc := testCase{
		name: "Test_CreateAlbumFolder",
		args: []string{
			"-create-album-folder",
			"../../test-data/albums",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_XMP(t *testing.T) {

	tc := testCase{
		name: "Test_XMP",
		args: []string{
			"../../test-data/xmp/files",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

// ResetImmich
// ⛔: will remove the content of the server.‼️
// Give the user of the connection to confirm the server instance: debug@example.com
//

func resetImmich(ic *immich.ImmichClient, user string) error {
	u, err := ic.ValidateConnection(context.Background())
	if err != nil {
		return err
	}
	if u.Email != user {
		return fmt.Errorf("Not the debug server")
	}

	albums, err := ic.GetAllAlbums(context.Background())
	if err != nil {
		return err
	}

	for _, album := range albums {
		err = ic.DeleteAlbum(context.Background(), album.ID)
		if err != nil {
			return err
		}
	}

	assets, err := ic.GetAllAssets(context.Background(), nil)
	if err != nil {
		return err
	}
	ids := []string{}
	for _, a := range assets {
		ids = append(ids, a.ID)
	}
	err = ic.DeleteAssets(context.Background(), ids, true)
	if err != nil {
		return nil
	}

	attempts := 5
	for attempts > 0 {
		assets, err := ic.GetAllAssets(context.Background(), nil)
		if err != nil {
			return err
		}
		if len(assets) == 0 {
			return nil
		}
		time.Sleep(5 * time.Second)
		attempts--
	}

	return errors.New("can't reset immich")
}
