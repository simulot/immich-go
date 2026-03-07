package upload

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// State tracks the progress of a batched upload session.
// It persists to disk so that interrupted uploads can be resumed.
type State struct {
	Version         int            `json:"version"`
	ArchivePath     string         `json:"archive_path"`
	ServerURL       string         `json:"server_url"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DateRange       StateDateRange `json:"date_range"`
	CompletedMonths []string       `json:"completed_months"`
	InProgress      InProgress     `json:"in_progress"`

	// stateDir is the directory where the state file is stored.
	// Not serialized to JSON.
	stateDir string
	mu       sync.Mutex `json:"-"`
}

// StateDateRange represents the overall date range of the archive.
type StateDateRange struct {
	Min time.Time `json:"min"`
	Max time.Time `json:"max"`
}

// InProgress tracks the current month being processed and the files uploaded so far.
type InProgress struct {
	Month         string         `json:"month"`
	UploadedFiles []UploadedFile `json:"uploaded_files"`
}

// UploadedFile records a file that has been successfully uploaded during the current month.
type UploadedFile struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
}

const stateVersion = 1

// DefaultStateDir computes the default state directory path for a given server URL and archive path.
// The directory is ~/.config/immich-go/state/<hash>/ where hash = sha256(archivePath + "|" + serverURL)[:16].
func DefaultStateDir(serverURL, archivePath string) string {
	h := sha256.Sum256([]byte(archivePath + "|" + serverURL))
	hash := fmt.Sprintf("%x", h[:])[:16]
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(configDir, "immich-go", "state", hash)
}

// LoadState loads the state from stateDir/state.json. If the file does not exist,
// a new state is created with the given serverURL and archivePath.
func LoadState(stateDir, serverURL, archivePath string) (*State, error) {
	stateFile := filepath.Join(stateDir, "state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			now := time.Now().UTC().Truncate(time.Second)
			return &State{
				Version:         stateVersion,
				ArchivePath:     archivePath,
				ServerURL:       serverURL,
				CreatedAt:       now,
				UpdatedAt:       now,
				CompletedMonths: []string{},
				InProgress: InProgress{
					UploadedFiles: []UploadedFile{},
				},
				stateDir: stateDir,
			}, nil
		}
		return nil, fmt.Errorf("can't read state file %s: %w", stateFile, err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("can't parse state file %s: %w", stateFile, err)
	}
	s.stateDir = stateDir

	// Ensure slices are non-nil after deserialization.
	if s.CompletedMonths == nil {
		s.CompletedMonths = []string{}
	}
	if s.InProgress.UploadedFiles == nil {
		s.InProgress.UploadedFiles = []UploadedFile{}
	}

	return &s, nil
}

// SaveState writes the state to disk atomically.
// It writes to a temporary file in the same directory and renames it into place.
func (s *State) SaveState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	if err := os.MkdirAll(s.stateDir, 0o700); err != nil {
		return fmt.Errorf("can't create state directory %s: %w", s.stateDir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("can't marshal state: %w", err)
	}

	stateFile := filepath.Join(s.stateDir, "state.json")
	tmpFile := stateFile + ".tmp"

	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("can't write temporary state file %s: %w", tmpFile, err)
	}

	if err := os.Rename(tmpFile, stateFile); err != nil {
		return fmt.Errorf("can't rename temporary state file: %w", err)
	}

	return nil
}

// IsMonthCompleted returns true if the given month has already been fully processed.
func (s *State) IsMonthCompleted(month string) bool {
	return slices.Contains(s.CompletedMonths, month)
}

// CompleteMonth marks the in-progress month as completed.
// It adds the month to CompletedMonths and clears the InProgress uploaded files.
func (s *State) CompleteMonth(month string) {
	if !s.IsMonthCompleted(month) {
		s.CompletedMonths = append(s.CompletedMonths, month)
	}
	if s.InProgress.Month == month {
		s.InProgress.Month = ""
		s.InProgress.UploadedFiles = []UploadedFile{}
	}
}

// IsFileUploaded checks whether a file with the given source, path, and size
// has already been recorded as uploaded in the current in-progress month.
func (s *State) IsFileUploaded(source, path string, size int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, f := range s.InProgress.UploadedFiles {
		if f.Source == source && f.Path == path && f.Size == size {
			return true
		}
	}
	return false
}

// RecordFileUploaded adds a file to the in-progress uploaded files list.
func (s *State) RecordFileUploaded(source, path string, size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.InProgress.UploadedFiles = append(s.InProgress.UploadedFiles, UploadedFile{
		Source: source,
		Path:   path,
		Size:   size,
	})
}

// SetInProgressMonth sets the current in-progress month.
func (s *State) SetInProgressMonth(month string) {
	s.InProgress.Month = month
	s.InProgress.UploadedFiles = []UploadedFile{}
}

// ResetState clears all progress, resetting the state to its initial condition.
func (s *State) ResetState() {
	now := time.Now().UTC().Truncate(time.Second)
	s.CompletedMonths = []string{}
	s.InProgress = InProgress{
		UploadedFiles: []UploadedFile{},
	}
	s.DateRange = StateDateRange{}
	s.UpdatedAt = now
}
