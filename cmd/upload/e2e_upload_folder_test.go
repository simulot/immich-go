//go:build e2e
// +build e2e

package upload

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/immich"
)

var myEnv map[string]string

func initMyEnv(t *testing.T) {
	if len(myEnv) > 0 {
		return
	}
	var err error
	e, err := godotenv.Read("../../.env")
	if err != nil {
		t.Fatalf("cant initialize environment variables: %s", err)
	}
	myEnv = e
	if myEnv["IMMICH_TESTFILES"] == "" {
		t.Fatal("missing IMMICH_TESTFILES in .env file")
	}
}

type immichSetupFunc func(ctx context.Context, t *testing.T, ic *immich.ImmichClient) func(t *testing.T)

type testCase struct {
	name        string
	args        []string
	resetImmich bool
	setup       immichSetupFunc
	APITrace    bool
	expectError bool
}

func runCase(t *testing.T, tc testCase) {

	initMyEnv(t)
	host := myEnv["IMMICH_E2E_HOST"]
	if host == "" {
		host = "http://localhost:2283"
	}
	key := myEnv["IMMICH_E2E_KEY"]
	if key == "" {
		t.Fatal("you must provide the IMMICH's API KEY in the environnement variable IMMICH_E2E_KEY")
	}

	user := myEnv["IMMICH_E2E_USER"]
	if user == "" {
		user = "debug.example.com"
	}

	ctx := context.Background()
	ic, err := immich.NewImmichClient(host, key, false)

	if tc.resetImmich {
		err := resetImmich(ic, user)
		if err != nil {
			t.Error(err)
			return
		}
	}

	if tc.setup != nil {
		teardown := tc.setup(ctx, t, ic)
		if teardown != nil {
			defer teardown(t)
		}
	}

	argc := []string{"-server=" + host, "-key=" + key, "-log-file=" + tc.name + ".log"}

	if tc.APITrace {
		argc = append(argc, "-api-trace=TRUE")
	}

	argc = append(argc, tc.args...)

	app := cmd.SharedFlags{}

	err = UploadCommand(ctx, &app, tc.args)
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
				myEnv["IMMICH_TESTFILES"] + "//low_high/high",
			},
			resetImmich: true,

			expectError: false,
		},
		{
			name: "upload folder",
			args: []string{
				myEnv["IMMICH_TESTFILES"] + "/low_high/high",
			},

			// resetImmich: true,
			expectError: false,
		},
		{
			name: "upload folder *.jpg",
			args: []string{
				"-google-photos",
				myEnv["IMMICH_TESTFILES"] + "/test_folder/*.jpg",
			},

			resetImmich: true,
			expectError: true,
		},
		{
			name: "upload folder *.jpg",
			args: []string{
				myEnv["IMMICH_TESTFILES"] + "/test_folder/*/*.jpg",
			},

			// resetImmich: true,
			expectError: false,
		},

		// {
		// 	name: "upload folder *.jpg - dry run",
		// 	args: []string{
		// 		"-dry-run",
		// 		myEnv["IMMICH_TESTFILES"] + "/full_takeout (copy)/Takeout/Google Photos/Photos from 2023",
		// 	},

		// 	// resetImmich: true,
		// 	expectError: false,
		// },

		{
			name: "upload google photos",
			args: []string{
				"-google-photos",
				myEnv["IMMICH_TESTFILES"] + "/low_high/Takeout",
			},
			// resetImmich: true,
			expectError: false,
		},
		{
			name: "upload burst Huawei",
			args: []string{
				"-stack-burst=FALSE",
				"-stack-jpg-raw=TRUE",
				myEnv["IMMICH_TESTFILES"] + "/burst/Tel",
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
			myEnv["IMMICH_TESTFILES"] + "/low_high/high",
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
			myEnv["IMMICH_TESTFILES"] + "/albums",
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
			"-create-stacks=false",
			myEnv["IMMICH_TESTFILES"] + "/xmp",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_Album_Issue_119(t *testing.T) {
	tc := []testCase{
		{
			name: "Test_Album 1",
			args: []string{
				"-album", "The Album",
				myEnv["IMMICH_TESTFILES"] + "/xmp/files",
			},
			setup: func(ctx context.Context, t *testing.T, ic *immich.ImmichClient) func(t *testing.T) {
				_, err := ic.CreateAlbum(ctx, "The Album", nil)
				if err != nil {
					t.Error(err)
				}
				return nil
			},
			resetImmich: true,
			expectError: false,
			APITrace:    false,
		},
		{
			name: "Test_Album 2",
			args: []string{
				"-album", "The Album",
				myEnv["IMMICH_TESTFILES"] + "/albums/Album test 6-10-23",
			},
			resetImmich: false,
			expectError: false,
			APITrace:    false,
		},
	}
	runCase(t, tc[0])
	runCase(t, tc[1])
}

func Test_Issue_126A(t *testing.T) {
	tc := testCase{
		name: "Test_Issue_126A",
		args: []string{
			"-exclude-types",
			".dng,.cr2,.arw,.rw2,.tif,.tiff,.gif,.psd",
			myEnv["IMMICH_TESTFILES"] + "/burst/PXL6",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_Issue_126B(t *testing.T) {
	tc := testCase{
		name: "Test_Issue_126B",
		args: []string{
			"-select-types",
			".jpg",
			myEnv["IMMICH_TESTFILES"] + "/burst/PXL6",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_Issue_129(t *testing.T) {
	tc := testCase{
		name: "Test_Issue_129",
		args: []string{
			"-google-photos",
			myEnv["IMMICH_TESTFILES"] + "/Weird file names #88",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_Issue_128(t *testing.T) {
	tc := testCase{
		name: "Test_Issue_128",
		args: []string{
			"-google-photos",
			myEnv["IMMICH_TESTFILES"] + "/Issue 128",
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
