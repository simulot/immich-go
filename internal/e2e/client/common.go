package client

import (
	"os"
	"strings"
	"testing"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

// Configuration from environment variables
var (
	immichURL    = getEnv("e2e_url", "http://localhost:2283")
	keysFile     = getEnv("e2e_users", findE2EUsersFile())
	sshHost      = getEnv("e2e_ssh", "")
	immichFolder = getEnv("e2e_folder", findE2EImmichDir())
	// dcPath       = getEnv("E2E_COMPOSE", path.Join(findE2EImmichDir(), "docker-compose.yml"))
	u1KeyPath  = "users/user1@immich.app/keys/e2eMinimal"
	admKeyPath = "users/admin@immich.app/keys/e2eAll"
)

func debug(t *testing.T) {
	e := os.Environ()
	for _, v := range e {
		if strings.HasPrefix(v, "e2e") {
			t.Logf("Env: %s", v)
		}
	}
}

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

var ictlr *e2eutils.ImmichController

// resetImmich resets the Immich database between tests
func resetImmich(t *testing.T) {
	var err error
	if ictlr == nil {
		if sshHost != "" {
			// Create a remote ImmichController
			ictlr, err = e2eutils.OpenImmichController(t.Context(), e2eutils.Remote(sshHost, immichURL, immichFolder))
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
