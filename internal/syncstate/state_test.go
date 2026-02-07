package syncstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockUnlock(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	if err := m.Lock(); err != nil {
		t.Fatalf("Lock failed: %v", err)
	}

	// Second lock should fail
	m2 := NewManager(dir, false)
	if err := m2.Lock(); err == nil {
		t.Fatal("Expected second lock to fail")
	}

	if err := m.Unlock(); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}

	// Now lock should succeed again
	m3 := NewManager(dir, false)
	if err := m3.Lock(); err != nil {
		t.Fatalf("Lock after unlock failed: %v", err)
	}
	m3.Unlock()
}

func TestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	if err := m.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	defer m.Unlock()

	state, err := m.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if state.Version != 1 {
		t.Fatalf("expected version 1, got %d", state.Version)
	}
	if len(state.Assets) != 0 {
		t.Fatalf("expected empty assets, got %d", len(state.Assets))
	}

	// Track assets
	m.TrackAsset("abc123", AssetEntry{
		ID:       "asset-1",
		Filename: "photo.jpg",
		Path:     "2026/2026-01/photo.jpg",
		Size:     12345,
	})
	m.TrackAsset("def456", AssetEntry{
		ID:       "asset-2",
		Filename: "video.mp4",
		Path:     "2026/2026-02/video.mp4",
		Size:     67890,
	})

	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload
	m2 := NewManager(dir, false)
	state2, err := m2.Load()
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}

	if len(state2.Assets) != 2 {
		t.Fatalf("expected 2 assets, got %d", len(state2.Assets))
	}
	if state2.Assets["abc123"].Filename != "photo.jpg" {
		t.Fatalf("expected photo.jpg, got %s", state2.Assets["abc123"].Filename)
	}
	if state2.Server != "" {
		// Server not set yet (no Validate called)
	}
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	m.Lock()
	defer m.Unlock()

	m.Load()

	// First validate sets server/user
	if err := m.Validate("https://immich.example", "user-1", false); err != nil {
		t.Fatalf("first validate: %v", err)
	}
	if m.State().Server != "https://immich.example" {
		t.Fatal("server not set")
	}

	// Same server/user OK
	if err := m.Validate("https://immich.example", "user-1", false); err != nil {
		t.Fatalf("same validate: %v", err)
	}

	// Different server should fail
	if err := m.Validate("https://other.example", "user-1", false); err == nil {
		t.Fatal("expected mismatch error")
	}

	// Force should bypass
	if err := m.Validate("https://other.example", "user-1", true); err != nil {
		t.Fatalf("force validate: %v", err)
	}
}

func TestDryRunNoWrite(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, true) // dry-run
	m.Lock()
	defer m.Unlock()

	m.Load()
	m.TrackAsset("abc", AssetEntry{ID: "1", Filename: "f.jpg", Path: "f.jpg", Size: 1})
	m.Save()

	// State file should not exist
	if _, err := os.Stat(filepath.Join(dir, stateDir, stateFile)); !os.IsNotExist(err) {
		t.Fatal("state file should not exist in dry-run mode")
	}
}

func TestRemoveAsset(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	m.Lock()
	defer m.Unlock()

	m.Load()
	m.TrackAsset("abc", AssetEntry{ID: "1", Filename: "f.jpg", Path: "f.jpg", Size: 1})
	m.TrackAsset("def", AssetEntry{ID: "2", Filename: "g.jpg", Path: "g.jpg", Size: 2})
	m.RemoveAsset("abc")

	if len(m.State().Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(m.State().Assets))
	}
	if _, ok := m.State().Assets["def"]; !ok {
		t.Fatal("expected 'def' to remain")
	}
}

func TestSaveIfNeeded(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	m.Lock()
	defer m.Unlock()

	m.Load()
	m.TrackAsset("abc", AssetEntry{ID: "1", Filename: "f.jpg", Path: "f.jpg", Size: 1})

	// Below threshold: no save
	if err := m.SaveIfNeeded(5); err != nil {
		t.Fatal(err)
	}
	if m.DirtyCount() != 1 {
		t.Fatalf("expected dirty=1, got %d", m.DirtyCount())
	}

	// At threshold: should save
	for i := range 4 {
		m.TrackAsset(string(rune('a'+i+1))+"x", AssetEntry{ID: string(rune('0' + i)), Filename: "x.jpg", Path: "x.jpg", Size: 1})
	}
	if err := m.SaveIfNeeded(5); err != nil {
		t.Fatal(err)
	}
	if m.DirtyCount() != 0 {
		t.Fatalf("expected dirty=0 after save, got %d", m.DirtyCount())
	}
}

func TestAtomicSave(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	m.Lock()
	defer m.Unlock()

	m.Load()
	m.TrackAsset("abc", AssetEntry{ID: "1", Filename: "f.jpg", Path: "f.jpg", Size: 1})
	m.Save()

	// Verify no temp file left
	statePath := filepath.Join(dir, stateDir, stateFile)
	tmpPath := statePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatal("temp file should not exist after save")
	}

	// Verify state file exists and is valid
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file should exist: %v", err)
	}
}

func TestMigration_OldStateWithoutCaptureDate(t *testing.T) {
	// Simulate a state file from before CaptureDate was added
	oldJSON := `{
  "version": 1,
  "server": "https://immich.example",
  "user_id": "user-1",
  "last_sync": "2025-01-01T00:00:00Z",
  "assets": {
    "abc123": {
      "id": "asset-1",
      "filename": "photo.jpg",
      "path": "2025/2025-01/photo.jpg",
      "size": 12345
    }
  }
}`

	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".immich-sync")
	os.MkdirAll(stateDir, 0o755)
	os.WriteFile(filepath.Join(stateDir, "state.json"), []byte(oldJSON), 0o644)

	m := NewManager(dir, false)
	state, err := m.Load()
	if err != nil {
		t.Fatalf("Load old state: %v", err)
	}

	entry := state.Assets["abc123"]
	if entry.Filename != "photo.jpg" {
		t.Fatalf("expected photo.jpg, got %s", entry.Filename)
	}
	if !entry.CaptureDate.IsZero() {
		t.Fatalf("expected zero CaptureDate for old state, got %v", entry.CaptureDate)
	}
}

func TestCaptureDate_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	m := NewManager(dir, false)
	if err := m.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}
	defer m.Unlock()

	m.Load()

	captureDate := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	m.TrackAsset("abc123", AssetEntry{
		ID:          "asset-1",
		Filename:    "photo.jpg",
		Path:        "2024/2024-06/photo.jpg",
		Size:        12345,
		CaptureDate: captureDate,
	})

	if err := m.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload and verify CaptureDate survives
	m2 := NewManager(dir, false)
	state2, err := m2.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	entry := state2.Assets["abc123"]
	if !entry.CaptureDate.Equal(captureDate) {
		t.Fatalf("expected CaptureDate %v, got %v", captureDate, entry.CaptureDate)
	}
}

func TestCaptureDate_OmittedWhenZero(t *testing.T) {
	entry := AssetEntry{
		ID:       "asset-1",
		Filename: "photo.jpg",
		Path:     "photo.jpg",
		Size:     100,
		// CaptureDate left as zero
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// The JSON should not contain "capture_date" when zero
	if contains := string(data); contains != "" {
		var m map[string]any
		json.Unmarshal(data, &m)
		if _, ok := m["capture_date"]; ok {
			t.Fatal("capture_date should be omitted when zero")
		}
	}
}
