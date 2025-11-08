package client

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// getSHA1ByFilename reads all files in the given folder and returns a map
// of SHA1 hashes indexed by filename (not full path, just the basename).
// Only regular files are processed; directories are skipped.
func getSHA1ByFilename(folderPath string) (map[string]string, error) {
	result := make(map[string]string)

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", folderPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(folderPath, entry.Name())
		hash, err := computeSHA1(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to compute SHA1 for %s: %w", filePath, err)
		}

		result[entry.Name()] = hash
	}

	return result, nil
}

// computeSHA1 calculates the SHA1 hash of a file and returns it as base64
func computeSHA1(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

/*
	This test verifies how immich-go replaces photos using the 2.2.0 API specifications
	1- log in with a new user
	2- upload low quality photos
	3- get the asset ID of the uploaded photos
	4- upload higher quality photos with the same name
	5- verify that the photos are replaced
*/

func Test_Replace(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	// A fresh user for a new test
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}
	t.Logf("user: email:%s, password:%s, key:%s", u1.Email, u1.Password, u1.APIKey)

	// lowQ, err := getSHA1ByFilename("DATA/replace/low_quality")
	// if err != nil {
	// 	t.Fatalf("can't get SHA1: %v", err)
	// }
	highQ, err := getSHA1ByFilename("DATA/replace/high_quality")
	if err != nil {
		t.Fatalf("can't get SHA1: %v", err)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"DATA/replace/low_quality",
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
		fileevent.Uploaded:         5,
		fileevent.UploadAddToAlbum: 0,
		fileevent.Tagged:           0,
	}, false, a.Jnl())

	time.Sleep(2 * time.Second)

	c, a = root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"--concurrent-tasks=0", // 0 to enable debuging
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"DATA/replace/high_quality",
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
		fileevent.Uploaded:         0,
		fileevent.UploadAddToAlbum: 0,
		fileevent.Tagged:           0,
		fileevent.UploadUpgraded:   5,
	}, false, a.Jnl())

	assets, err := e2eutils.GetAllAssets(u1.Email, u1.Password)
	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	// Compare assets with high quality files
	t.Logf("Total assets retrieved: %d", len(assets))
	t.Logf("Total high quality files: %d", len(highQ))

	// The server's assets should have been replaced by the high quality files
	if len(assets) != len(highQ) {
		t.Errorf("Number of assets (%d) does not match number of high quality files (%d)", len(assets), len(highQ))
	}

	// Check if all high quality files are present in assets
	var missingFiles []string
	var mismatchedChecksums []string
	var matchedFiles int

	for filename, expectedChecksum := range highQ {
		asset, found := assets[filename]
		if !found {
			missingFiles = append(missingFiles, filename)
			continue
		}

		if asset.Checksum != expectedChecksum {
			mismatchedChecksums = append(mismatchedChecksums, fmt.Sprintf("%s: expected %s, got %s", filename, expectedChecksum, asset.Checksum))
		} else {
			matchedFiles++
		}
	}

	// Check for extra files in assets that aren't in highQ
	var extraFiles []string
	for filename := range assets {
		if _, found := highQ[filename]; !found {
			extraFiles = append(extraFiles, filename)
		}
	}

	// Report results
	if len(missingFiles) > 0 {
		t.Errorf("Missing files in assets: %v", missingFiles)
	}

	if len(mismatchedChecksums) > 0 {
		t.Errorf("Checksum mismatches:\n%s", mismatchedChecksums)
	}

	if len(extraFiles) > 0 {
		t.Logf("Extra files in assets (not in high quality folder): %v", extraFiles)
	}

	t.Logf("Matched files with correct checksums: %d/%d", matchedFiles, len(highQ))

	// Final assertion: all high quality files should be present with correct checksums
	if len(missingFiles) > 0 || len(mismatchedChecksums) > 0 {
		t.Error("Assets do not match expected high quality files")
	} else {
		t.Log("✓ All high quality files were correctly uploaded and replaced the low quality versions")
	}
}
