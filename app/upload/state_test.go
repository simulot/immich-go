package upload

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateCreation(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir, "https://immich.example.com", "/path/to/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if s.Version != stateVersion {
		t.Errorf("expected version %d, got %d", stateVersion, s.Version)
	}
	if s.ArchivePath != "/path/to/archive" {
		t.Errorf("expected archive_path %q, got %q", "/path/to/archive", s.ArchivePath)
	}
	if s.ServerURL != "https://immich.example.com" {
		t.Errorf("expected server_url %q, got %q", "https://immich.example.com", s.ServerURL)
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}
	if len(s.CompletedMonths) != 0 {
		t.Errorf("expected empty completed_months, got %v", s.CompletedMonths)
	}
	if s.InProgress.Month != "" {
		t.Errorf("expected empty in_progress month, got %q", s.InProgress.Month)
	}
	if len(s.InProgress.UploadedFiles) != 0 {
		t.Errorf("expected empty uploaded_files, got %v", s.InProgress.UploadedFiles)
	}
}

func TestStateSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	original, err := LoadState(dir, "https://immich.example.com", "/path/to/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Populate all fields
	original.DateRange = StateDateRange{
		Min: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Max: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	original.CompletedMonths = []string{"2020-01", "2020-02", "2020-03"}
	original.InProgress = InProgress{
		Month: "2020-04",
		UploadedFiles: []UploadedFile{
			{Source: "archive.zip", Path: "Photos/IMG_001.HEIC", Size: 1234567},
			{Source: "archive.zip", Path: "Photos/IMG_002.JPG", Size: 7654321},
		},
	}

	if err := original.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify the file was written
	stateFile := filepath.Join(dir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("state.json file was not created")
	}

	// Load back
	loaded, err := LoadState(dir, "https://immich.example.com", "/path/to/archive")
	if err != nil {
		t.Fatalf("LoadState after save failed: %v", err)
	}

	// Verify all fields survived serialization
	if loaded.Version != original.Version {
		t.Errorf("version: expected %d, got %d", original.Version, loaded.Version)
	}
	if loaded.ArchivePath != original.ArchivePath {
		t.Errorf("archive_path: expected %q, got %q", original.ArchivePath, loaded.ArchivePath)
	}
	if loaded.ServerURL != original.ServerURL {
		t.Errorf("server_url: expected %q, got %q", original.ServerURL, loaded.ServerURL)
	}
	if !loaded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("created_at: expected %v, got %v", original.CreatedAt, loaded.CreatedAt)
	}
	if !loaded.DateRange.Min.Equal(original.DateRange.Min) {
		t.Errorf("date_range.min: expected %v, got %v", original.DateRange.Min, loaded.DateRange.Min)
	}
	if !loaded.DateRange.Max.Equal(original.DateRange.Max) {
		t.Errorf("date_range.max: expected %v, got %v", original.DateRange.Max, loaded.DateRange.Max)
	}
	if len(loaded.CompletedMonths) != len(original.CompletedMonths) {
		t.Fatalf("completed_months length: expected %d, got %d", len(original.CompletedMonths), len(loaded.CompletedMonths))
	}
	for i, m := range original.CompletedMonths {
		if loaded.CompletedMonths[i] != m {
			t.Errorf("completed_months[%d]: expected %q, got %q", i, m, loaded.CompletedMonths[i])
		}
	}
	if loaded.InProgress.Month != original.InProgress.Month {
		t.Errorf("in_progress.month: expected %q, got %q", original.InProgress.Month, loaded.InProgress.Month)
	}
	if len(loaded.InProgress.UploadedFiles) != len(original.InProgress.UploadedFiles) {
		t.Fatalf("uploaded_files length: expected %d, got %d", len(original.InProgress.UploadedFiles), len(loaded.InProgress.UploadedFiles))
	}
	for i, f := range original.InProgress.UploadedFiles {
		lf := loaded.InProgress.UploadedFiles[i]
		if lf.Source != f.Source || lf.Path != f.Path || lf.Size != f.Size {
			t.Errorf("uploaded_files[%d]: expected %+v, got %+v", i, f, lf)
		}
	}
}

func TestStateMonthCompletionFlow(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir, "https://immich.example.com", "/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	month := "2023-06"

	// Month should not be completed initially
	if s.IsMonthCompleted(month) {
		t.Error("month should not be completed initially")
	}

	// Set in-progress
	s.SetInProgressMonth(month)
	if s.InProgress.Month != month {
		t.Errorf("expected in_progress month %q, got %q", month, s.InProgress.Month)
	}

	// Record some files
	s.RecordFileUploaded("archive.zip", "Photos/IMG_001.HEIC", 1000)
	s.RecordFileUploaded("archive.zip", "Photos/IMG_002.JPG", 2000)

	if len(s.InProgress.UploadedFiles) != 2 {
		t.Fatalf("expected 2 uploaded files, got %d", len(s.InProgress.UploadedFiles))
	}

	// Complete the month
	s.CompleteMonth(month)

	if !s.IsMonthCompleted(month) {
		t.Error("month should be completed after CompleteMonth")
	}
	if s.InProgress.Month != "" {
		t.Errorf("expected empty in_progress month after completion, got %q", s.InProgress.Month)
	}
	if len(s.InProgress.UploadedFiles) != 0 {
		t.Errorf("expected empty uploaded_files after completion, got %d", len(s.InProgress.UploadedFiles))
	}

	// Completing the same month again should not duplicate it
	s.CompleteMonth(month)
	count := 0
	for _, m := range s.CompletedMonths {
		if m == month {
			count++
		}
	}
	if count != 1 {
		t.Errorf("month should appear exactly once in completed_months, got %d", count)
	}
}

func TestStateIsFileUploaded(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir, "https://immich.example.com", "/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	s.SetInProgressMonth("2023-06")
	s.RecordFileUploaded("archive.zip", "Photos/IMG_001.HEIC", 1000)

	// Exact match
	if !s.IsFileUploaded("archive.zip", "Photos/IMG_001.HEIC", 1000) {
		t.Error("expected file to be found as uploaded")
	}

	// Different source
	if s.IsFileUploaded("other.zip", "Photos/IMG_001.HEIC", 1000) {
		t.Error("expected file with different source to not match")
	}

	// Different path
	if s.IsFileUploaded("archive.zip", "Photos/IMG_999.HEIC", 1000) {
		t.Error("expected file with different path to not match")
	}

	// Different size
	if s.IsFileUploaded("archive.zip", "Photos/IMG_001.HEIC", 9999) {
		t.Error("expected file with different size to not match")
	}
}

func TestStateHashUniqueness(t *testing.T) {
	dir1 := DefaultStateDir("https://server1.com", "/archive/a")
	dir2 := DefaultStateDir("https://server2.com", "/archive/a")
	dir3 := DefaultStateDir("https://server1.com", "/archive/b")
	dir4 := DefaultStateDir("https://server1.com", "/archive/a") // same as dir1

	if dir1 == dir2 {
		t.Error("different server URLs should produce different state dirs")
	}
	if dir1 == dir3 {
		t.Error("different archive paths should produce different state dirs")
	}
	if dir1 != dir4 {
		t.Error("same inputs should produce the same state dir")
	}
}

func TestStateResetState(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir, "https://immich.example.com", "/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Populate state
	s.DateRange = StateDateRange{
		Min: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		Max: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	s.CompletedMonths = []string{"2020-01", "2020-02"}
	s.SetInProgressMonth("2020-03")
	s.RecordFileUploaded("archive.zip", "Photos/IMG_001.HEIC", 1000)

	// Reset
	s.ResetState()

	if len(s.CompletedMonths) != 0 {
		t.Errorf("expected empty completed_months after reset, got %v", s.CompletedMonths)
	}
	if s.InProgress.Month != "" {
		t.Errorf("expected empty in_progress month after reset, got %q", s.InProgress.Month)
	}
	if len(s.InProgress.UploadedFiles) != 0 {
		t.Errorf("expected empty uploaded_files after reset, got %d", len(s.InProgress.UploadedFiles))
	}
	if !s.DateRange.Min.IsZero() {
		t.Error("expected zero date_range.min after reset")
	}
	if !s.DateRange.Max.IsZero() {
		t.Error("expected zero date_range.max after reset")
	}

	// Version and identity fields should be preserved
	if s.Version != stateVersion {
		t.Errorf("expected version %d after reset, got %d", stateVersion, s.Version)
	}
	if s.ArchivePath != "/archive" {
		t.Errorf("expected archive_path preserved after reset, got %q", s.ArchivePath)
	}
	if s.ServerURL != "https://immich.example.com" {
		t.Errorf("expected server_url preserved after reset, got %q", s.ServerURL)
	}
}

func TestStateSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested", "deep", "state")

	s, err := LoadState(nestedDir, "https://immich.example.com", "/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if err := s.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	stateFile := filepath.Join(nestedDir, "state.json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatal("state.json file was not created in nested directory")
	}
}

func TestStateAtomicWrite(t *testing.T) {
	dir := t.TempDir()

	s, err := LoadState(dir, "https://immich.example.com", "/archive")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	// Save initial state
	if err := s.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify no temp file is left behind
	tmpFile := filepath.Join(dir, "state.json.tmp")
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("temporary file should not exist after successful save")
	}
}
