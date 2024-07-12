//go:build e2e
// +build e2e

package upload

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	e, err := godotenv.Read("../../e2e.env")
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
	changeCWD   string
}

func runCase(t *testing.T, tc testCase) {
	fmt.Println("Test case", tc.name)
	host := myEnv["IMMICH_HOST"]
	if host == "" {
		host = "http://localhost:2283"
	}
	key := myEnv["IMMICH_KEY"]
	if key == "" {
		t.Fatal("you must provide the IMMICH's API KEY in the environnement variable IMMICH_KEY")
	}

	user := myEnv["IMMICH_USER"]
	if user == "" {
		user = "demo@immich.app"
	}

	if tc.changeCWD != "" {
		cwd, _ := os.Getwd()
		defer func() {
			os.Chdir(cwd)
		}()
		_ = os.Chdir(tc.changeCWD)
	}

	ctx := context.Background()
	ic, err := immich.NewImmichClient(host, key)
	if err != nil {
		t.Error(err)
		return
	}
	u, err := ic.ValidateConnection(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	_ = u
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

	args := []string{"-server=" + host, "-key=" + key, "-log-file=" + tc.name + ".log", "-log-level=INFO", "-no-ui"}

	if tc.APITrace {
		args = append(args, "-api-trace=TRUE")
	}

	args = append(args, tc.args...)

	app := cmd.SharedFlags{
		Immich: ic,
	}

	os.Remove(tc.name + ".log")

	err = UploadCommand(ctx, &app, args)
	if (tc.expectError && (err == nil)) || (!tc.expectError && (err != nil)) {
		t.Errorf("unexpected err: %v", err)
		return
	}
}

func TestE2eUpload(t *testing.T) {
	initMyEnv(t)

	tests := []testCase{
		{
			name: "upload folder",
			args: []string{
				myEnv["IMMICH_TESTFILES"] + "/low_high/high",
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
			expectError: false,
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
	initMyEnv(t)

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

func Test_duplicate_albums_355(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_duplicate_albums_355_p1",
		args: []string{
			"-google-photos",
			"/home/jfcassan/Dev/test-data/#355 album of duplicates/takeout1",
		},
		resetImmich: true,
		expectError: false,
	}
	runCase(t, tc)
	tc = testCase{
		name: "Test_duplicate_albums_355_p2",
		args: []string{
			"-google-photos",
			"/home/jfcassan/Dev/test-data/#355 album of duplicates/takeout2",
		},
		resetImmich: false,
		expectError: false,
	}
	runCase(t, tc)
}

func Test_PermissionError(t *testing.T) {
	initMyEnv(t)

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
	initMyEnv(t)

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
	initMyEnv(t)

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

func Test_XMP2(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_XMP2",
		args: []string{
			"-create-stacks=false",
			"-create-album-folder",
			// myEnv["IMMICH_TESTFILES"] + "/xmp/files",
			// myEnv["IMMICH_TESTFILES"] + "/xmp/files/*.CR2",
			myEnv["IMMICH_TESTFILES"] + "/xmp/files*/*.CR2",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_Album_Issue_119(t *testing.T) {
	initMyEnv(t)

	tc := []testCase{
		{
			name: "Test_Album 1",
			args: []string{
				"-album", "The Album",
				myEnv["IMMICH_TESTFILES"] + "/xmp/files",
			},
			setup: func(ctx context.Context, t *testing.T, ic *immich.ImmichClient) func(t *testing.T) {
				_, err := ic.CreateAlbum(ctx, "The Album", "Description", nil)
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
	initMyEnv(t)

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
	initMyEnv(t)

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
	initMyEnv(t)

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

// Test_Issue_128
// Manage GP with no names
func Test_Issue_128(t *testing.T) {
	initMyEnv(t)

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

// Test_GP_MultiZip test the new way to pars arguments (post 0.12.0)
func Test_GP_MultiZip(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_Issue_128",
		args: []string{
			"-google-photos",
			myEnv["IMMICH_TESTFILES"] + "/google-photos/zip*.zip",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_ExtensionsFromTheServer(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "ExtensionsFromTheServer",
		args: []string{
			// "-log-json",
			myEnv["IMMICH_TESTFILES"] + "/low_high/high",
		},

		// resetImmich: true,
		expectError: false,
	}
	runCase(t, tc)
}

// Test_Issue_173: date of take is the file modification date
func Test_Issue_173(t *testing.T) {
	mtime := time.Date(2020, 1, 1, 15, 30, 45, 0, time.Local)
	err := os.Chtimes("TEST_DATA/nodate/NO_DATE.jpg", time.Now(), mtime)
	if err != nil {
		t.Error(err.Error())
		return
	}
	initMyEnv(t)

	tc := testCase{
		name: "Test_Issue_173",
		args: []string{
			"-when-no-date=FILE",
			"TEST_DATA/nodate",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

// Test_Issue_159: Albums from subdirectories with matching names
func Test_Issue_159(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_Issue_159",
		args: []string{
			"-create-album-folder=true",
			// "TEST_DATA/folder/high/Album*",
			"TEST_DATA/folder/high",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

// #304  Not all images being uploaded #304
func Test_CreateAlbumFolder_304(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_#304_UploadFiles",
		args: []string{
			"-album", "Album Name",
			"*.JPG",
		},
		resetImmich: true,
		expectError: false,
		changeCWD:   myEnv["IMMICH_TESTFILES"] + "/Error Upload #304",
	}
	runCase(t, tc)
}

func Test_CreateAlbumFolder_304_2(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_#304_UploadFiles",
		args: []string{
			"-create-album-folder",
			"*.JPG",
		},
		resetImmich: true,
		expectError: false,
		changeCWD:   myEnv["IMMICH_TESTFILES"] + "/Error Upload #304",
	}
	runCase(t, tc)
}

func Test_EnrichedAlbum_297(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_EnrichedAlbum_297",
		args: []string{
			"-google-photos",
			myEnv["IMMICH_TESTFILES"] + "/#297 Album enrichis #329 #297/Album texts #287/takeout-20240613T094535Z-001.zip",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_BannedFiles_(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_BannedFiles_",
		args: []string{
			"-exclude-files=backup/",
			"-exclude-files=copy).*",
			"TEST_DATA/banned",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

func Test_MissedJSON(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_MissedJSON",
		args: []string{
			"-google-photos",
			"-exclude-files=backup/",
			"-exclude-files=copy).*",
			"TEST_DATA/banned",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    false,
	}
	runCase(t, tc)
}

// Check if the small version of the photos loaded with the takeout
// is replaced by the better version
func Test_SmallTakeout_Better_p1(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_SmallTakeout_Better_p1",
		args: []string{
			"-google-photos",
			myEnv["IMMICH_TESTFILES"] + "/low_high/Takeout",
		},
		resetImmich: true,
		expectError: false,
		APITrace:    true,
	}
	runCase(t, tc)
}

func Test_SmallTakeout_Better_p2(t *testing.T) {
	initMyEnv(t)

	tc := testCase{
		name: "Test_SmallTakeout_Better_p2",
		args: []string{
			myEnv["IMMICH_TESTFILES"] + "/low_high/high",
		},
		resetImmich: false,
		expectError: false,
		APITrace:    true,
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

	assets, err := ic.GetAllAssets(context.Background())
	if err != nil {
		return err
	}
	ids := []string{}
	for _, a := range assets {
		ids = append(ids, a.ID)
	}
	err = ic.DeleteAssets(context.Background(), ids, true)
	if err != nil {
		return err
	}

	attempts := 5
	for attempts > 0 {
		assets, err := ic.GetAllAssets(context.Background())
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
