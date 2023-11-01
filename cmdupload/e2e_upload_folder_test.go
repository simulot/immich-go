//go:build e2e
// +build e2e

package cmdupload

import (
	"context"
	"immich-go/immich"
	"immich-go/immich/logger"
	"testing"

	"github.com/joho/godotenv"
)

func TestE2eUpload(t *testing.T) {

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

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name: "upload google photos",
			args: []string{
				"-google-photos",
				"../../test-data/low_high/Takeout",
			},
			expectError: false,
		},
		{
			name: "upload folder",
			args: []string{
				"-google-photos",
				"../../test-data/low_high/high",
			},
			expectError: false,
		},
	}

	logger := logger.NewLogger(logger.Debug, true, false)
	app, err := immich.NewImmichClient(host, key, "e2e", false)

	if err != nil {
		t.Error(err)
		return
	}

	ctx := context.Background()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err = UploadCommand(ctx, app, logger, tc.args)
			if (tc.expectError && err == nil) || (!tc.expectError && err != nil) {
				t.Errorf("unexpected err: %v", err)
				return
			}
		})
	}
}
