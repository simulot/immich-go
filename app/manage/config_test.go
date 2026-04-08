package manage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	yaml := `people-album-sync:
  albums:
    - album: "Family Vacation"
      people:
        names: ["Alice", "Bob"]
        ids: ["uuid-1"]
    - album: "Kids"
      people:
        names: ["Alice"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "manage.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.PeopleAlbumSync == nil {
		t.Fatal("expected people-album-sync section")
	}

	if len(cfg.PeopleAlbumSync.Albums) != 2 {
		t.Fatalf("expected 2 albums, got %d", len(cfg.PeopleAlbumSync.Albums))
	}

	a0 := cfg.PeopleAlbumSync.Albums[0]
	if a0.Album != "Family Vacation" {
		t.Errorf("expected album name 'Family Vacation', got %q", a0.Album)
	}
	if len(a0.People.Names) != 2 {
		t.Errorf("expected 2 people names, got %d", len(a0.People.Names))
	}
	if len(a0.People.IDs) != 1 {
		t.Errorf("expected 1 people ID, got %d", len(a0.People.IDs))
	}

	a1 := cfg.PeopleAlbumSync.Albums[1]
	if a1.Album != "Kids" {
		t.Errorf("expected album name 'Kids', got %q", a1.Album)
	}
	if len(a1.People.Names) != 1 {
		t.Errorf("expected 1 people name, got %d", len(a1.People.Names))
	}
	if len(a1.People.IDs) != 0 {
		t.Errorf("expected 0 people IDs, got %d", len(a1.People.IDs))
	}
}

func TestParseConfig_Mode(t *testing.T) {
	yaml := `people-album-sync:
  albums:
    - album: "Everyone Together"
      people:
        names: ["Alice", "Bob"]
      mode: "all"
    - album: "Anyone"
      people:
        names: ["Alice", "Bob"]
      mode: "any"
    - album: "Default Mode"
      people:
        names: ["Alice"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "manage.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	// Before validation, entry with no mode should be empty
	if cfg.PeopleAlbumSync.Albums[2].Mode != "" {
		t.Errorf("expected empty mode before validation, got %q", cfg.PeopleAlbumSync.Albums[2].Mode)
	}

	if err := cfg.PeopleAlbumSync.validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	// After validation, all modes should be normalized
	if cfg.PeopleAlbumSync.Albums[0].Mode != ModeAll {
		t.Errorf("expected mode 'all', got %q", cfg.PeopleAlbumSync.Albums[0].Mode)
	}
	if cfg.PeopleAlbumSync.Albums[1].Mode != ModeAny {
		t.Errorf("expected mode 'any', got %q", cfg.PeopleAlbumSync.Albums[1].Mode)
	}
	if cfg.PeopleAlbumSync.Albums[2].Mode != ModeAll {
		t.Errorf("expected mode 'all' after validation (default), got %q", cfg.PeopleAlbumSync.Albums[2].Mode)
	}
}

func TestParseConfig_AlbumByID(t *testing.T) {
	yaml := `people-album-sync:
  albums:
    - album_id: "album-uuid-123"
      people:
        names: ["Alice"]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "manage.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.PeopleAlbumSync.Albums[0].AlbumID != "album-uuid-123" {
		t.Errorf("expected album_id 'album-uuid-123', got %q", cfg.PeopleAlbumSync.Albums[0].AlbumID)
	}
}

func TestParseConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "no album identifier",
			yaml: `people-album-sync:
  albums:
    - people:
        names: ["Alice"]
`,
		},
		{
			name: "no people specified",
			yaml: `people-album-sync:
  albums:
    - album: "Test"
      people:
`,
		},
		{
			name: "empty albums list",
			yaml: `people-album-sync:
  albums: []
`,
		},
		{
			name: "both album and album_id",
			yaml: `people-album-sync:
  albums:
    - album: "Test"
      album_id: "uuid-123"
      people:
        names: ["Alice"]
`,
		},
		{
			name: "invalid mode",
			yaml: `people-album-sync:
  albums:
    - album: "Test"
      people:
        names: ["Alice"]
      mode: "invalid"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "manage.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := ParseConfig(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PeopleAlbumSync == nil {
				t.Fatal("expected people-album-sync section")
			}
			err = cfg.PeopleAlbumSync.validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}
