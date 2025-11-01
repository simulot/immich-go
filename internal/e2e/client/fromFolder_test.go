//go:build e2e

package client

import (
	"os"
	"path"
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// Configuration from environment variables
var (
	immichURL    = getEnv("E2E_SERVER", "http://localhost:2283")
	keysFile     = getEnv("E2E_USERS", findE2EUsersFile())
	sshHost      = getEnv("E2E_SSH", "")
	immichFolder = getEnv("E2E_FOLDER", findE2EImmichDir())
	dcPath       = getEnv("E2E_COMPOSE", path.Join(findE2EImmichDir(), "docker-compose.yml"))
	u1KeyPath    = "users/user1@immich.app/keys/e2eMinimal"
	admKeyPath   = "users/admin@immich.app/keys/e2eAll"
)

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// findE2EUsersFile searches for e2eusers.yml in multiple possible locations
func findE2EUsersFile() string {
	// Possible locations to check (in order of preference)

	candidates := []string{
		// CI environment - artifact downloaded to workspace root
		"e2e-immich/e2eusers.yml",
		// Local development - from test directory
		"../../../e2e-immich/e2eusers.yml",
		// Local development - from workspace root
		"./e2e-immich/e2eusers.yml",
		// Local development - from internal/e2e
		"../../e2e-immich/e2eusers.yml",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default fallback
	return "./e2e-immich/e2eusers.yml"
}

// resetImmich resets the Immich database between tests
func resetImmich(t *testing.T) {
	var ictlr *e2eutils.ImmichController
	var err error
	if sshHost != "" {
		// Create a remote ImmichController
		ictlr, err = e2eutils.OpenImmichController(t.Context(), e2eutils.Remote(sshHost, immichFolder))
		if err != nil {
			t.Fatalf("can't open the immich controller: %s", err.Error())
		}
		t.Logf("remote immich controller created, host:%s", sshHost)
	} else {
		// Create a local ImmichController
		ictlr, err = e2eutils.OpenImmichController(t.Context(), e2eutils.Local(immichFolder))
		if err != nil {
			t.Fatalf("can't open the immich controller: %s", err.Error())
		}
		t.Logf("local immich controller created, path:%s", immichFolder)
	}
	// Reset Immich using the controller (handles both local and remote)
	err = ictlr.ResetImmich(t.Context())
	if err != nil {
		t.Fatalf("failed to reset immich: %s", err.Error())
	}
	t.Logf("Immich reset successful")
}

// findE2EImmichDir searches for the e2e-immich directory
func findE2EImmichDir() string {
	candidates := []string{
		"../../../e2e-immich",
		"./e2e-immich",
		"../../e2e-immich",
	}

	for _, path := range candidates {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			return path
		}
	}

	return "./e2e-immich"
}

func Test_FromFolder(t *testing.T) {
	// Load user credentials
	t.Logf("Loading keys from: %s", keysFile)
	keys, err := e2eutils.KeysFromFile(keysFile)
	if err != nil {
		t.Fatalf("Can't get the keys from %s: %s", keysFile, err.Error())
	}

	// Reset Immich before test
	resetImmich(t)

	// Get API keys for users
	u1Key := keys.Get(u1KeyPath)
	admKey := keys.Get(admKeyPath)

	if u1Key == "" || admKey == "" {
		t.Fatalf("Missing required API keys in %s", keysFile)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + immichURL,
		"--api-key=" + u1Key,
		"--admin-api-key=" + admKey,
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"DATA/fromFolder/recursive",
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
		fileevent.Uploaded:         40,
		fileevent.UploadAddToAlbum: 0,
		fileevent.Tagged:           0,
	}, false, a.Jnl())
}
