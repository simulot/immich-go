package upload_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/app/root"
)

const folderAlbumConflict = "cannot use both --into-album and --folder-as-album flags"

func TestIntoAlbumAndFolderAsAlbumAreMutuallyExclusive(t *testing.T) {
	assertFolderAlbumConflict(t, []string{
		"upload", "from-folder",
		"--dry-run",
		"--into-album=target",
		"--folder-as-album=FOLDER",
		t.TempDir(),
	})
}

func TestLegacyIntoAlbumEnvironmentVariable(t *testing.T) {
	t.Setenv("IMMICH_GO_UPLOAD_INTO_ALBUM", "")
	t.Setenv("IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM", "legacy")

	assertFolderAlbumConflict(t, []string{
		"upload", "from-folder",
		"--dry-run",
		"--folder-as-album=FOLDER",
		t.TempDir(),
	})
}

func TestLegacyIntoAlbumConfigKey(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "legacy.toml")
	err := os.WriteFile(configFile, []byte("[upload.from-folder]\ninto-album = 'legacy'\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("IMMICH_GO_UPLOAD_INTO_ALBUM", "")
	t.Setenv("IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM", "")
	assertFolderAlbumConflict(t, []string{
		"--config=" + configFile,
		"upload", "from-folder",
		"--dry-run",
		"--folder-as-album=FOLDER",
		t.TempDir(),
	})
}

func TestLegacyIntoAlbumIsSourceSpecific(t *testing.T) {
	t.Setenv("IMMICH_GO_UPLOAD_INTO_ALBUM", "")
	t.Setenv("IMMICH_GO_UPLOAD_FROM_FOLDER_INTO_ALBUM", "legacy-folder")

	cmd, _ := root.RootImmichGoCommand(context.Background())
	cmd.SetArgs([]string{
		"upload", "from-icloud",
		"--dry-run",
		"--folder-as-album=FOLDER",
		"--no-ui",
		t.TempDir(),
	})
	err := cmd.Execute()
	if err != nil && strings.Contains(err.Error(), folderAlbumConflict) {
		t.Fatalf("folder legacy setting leaked into from-icloud: %v", err)
	}
}

func TestArchiveIntoAlbumRemainsAcceptedAsDeprecatedNoOp(t *testing.T) {
	source := t.TempDir()
	err := os.WriteFile(filepath.Join(source, "photo.jpg"), []byte("test"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cmd, _ := root.RootImmichGoCommand(context.Background())
	cmd.SetArgs([]string{
		"archive", "from-folder",
		"--write-to-folder=" + t.TempDir(),
		"--into-album=legacy",
		source,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("archive should accept the deprecated flag: %v", err)
	}
}

func assertFolderAlbumConflict(t *testing.T, args []string) {
	t.Helper()
	cmd, _ := root.RootImmichGoCommand(context.Background())
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), folderAlbumConflict) {
		t.Fatalf("expected %q, got %v", folderAlbumConflict, err)
	}
}
