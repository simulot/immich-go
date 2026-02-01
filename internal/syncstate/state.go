// Package syncstate manages persistent state for the sync command.
// State is stored per-directory in .immich-sync/state.json.
// It tracks which assets have been synced and prevents duplicate operations.
package syncstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	stateDir  = ".immich-sync"
	stateFile = "state.json"
	lockFile  = "lock"
	version   = 1
)

// AssetEntry tracks a single synced asset.
type AssetEntry struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
}

// State represents the persistent sync state for a directory.
type State struct {
	Version  int                   `json:"version"`
	Server   string                `json:"server"`
	UserID   string                `json:"user_id"`
	LastSync time.Time             `json:"last_sync"`
	Assets   map[string]AssetEntry `json:"assets"` // key: SHA1 checksum
}

// Manager handles loading, saving, and locking of sync state.
type Manager struct {
	dir    string // base sync directory
	state  *State
	lockFd *os.File
	dirty  int // count of modifications since last save
	dryRun bool
}

// NewManager creates a state manager for the given sync directory.
func NewManager(syncDir string, dryRun bool) *Manager {
	return &Manager{
		dir:    syncDir,
		dryRun: dryRun,
	}
}

func (m *Manager) stateDir() string {
	return filepath.Join(m.dir, stateDir)
}

func (m *Manager) statePath() string {
	return filepath.Join(m.stateDir(), stateFile)
}

func (m *Manager) lockPath() string {
	return filepath.Join(m.stateDir(), lockFile)
}

// Lock acquires an exclusive lock for this sync directory.
// Returns an error if another sync is already running.
func (m *Manager) Lock() error {
	if err := os.MkdirAll(m.stateDir(), 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	fd, err := os.OpenFile(m.lockPath(), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("another sync is already running in %s (lock file exists: %s). Remove it manually if this is incorrect", m.dir, m.lockPath())
		}
		return fmt.Errorf("creating lock file: %w", err)
	}
	fmt.Fprintf(fd, "%d", os.Getpid())
	m.lockFd = fd
	return nil
}

// Unlock releases the lock.
func (m *Manager) Unlock() error {
	if m.lockFd != nil {
		m.lockFd.Close()
		m.lockFd = nil
	}
	return os.Remove(m.lockPath())
}

// Load reads the state file. If no state exists, returns a fresh state.
func (m *Manager) Load() (*State, error) {
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			m.state = &State{
				Version: version,
				Assets:  make(map[string]AssetEntry),
			}
			return m.state, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if s.Version != version {
		return nil, fmt.Errorf("unsupported state version %d (expected %d)", s.Version, version)
	}
	if s.Assets == nil {
		s.Assets = make(map[string]AssetEntry)
	}
	m.state = &s
	return m.state, nil
}

// Validate checks that the state matches the given server and user.
// Returns an error on mismatch unless the state is empty (first sync).
func (m *Manager) Validate(server, userID string, force bool) error {
	if m.state == nil {
		return errors.New("state not loaded")
	}
	// First sync — set the server/user
	if m.state.Server == "" && m.state.UserID == "" {
		m.state.Server = server
		m.state.UserID = userID
		return nil
	}
	if m.state.Server != server || m.state.UserID != userID {
		if force {
			return nil
		}
		return fmt.Errorf("state mismatch: state has server=%q user=%q but current is server=%q user=%q. Use --force to override",
			m.state.Server, m.state.UserID, server, userID)
	}
	return nil
}

// State returns the current loaded state.
func (m *Manager) State() *State {
	return m.state
}

// TrackAsset records an asset in the state. Increments the dirty counter.
func (m *Manager) TrackAsset(checksum string, entry AssetEntry) {
	if m.state == nil {
		return
	}
	m.state.Assets[checksum] = entry
	m.dirty++
}

// RemoveAsset removes an asset from the state by checksum.
func (m *Manager) RemoveAsset(checksum string) {
	if m.state == nil {
		return
	}
	delete(m.state.Assets, checksum)
	m.dirty++
}

// NeedsSave returns true if there are unsaved modifications.
func (m *Manager) NeedsSave() bool {
	return m.dirty > 0
}

// DirtyCount returns the number of unsaved modifications.
func (m *Manager) DirtyCount() int {
	return m.dirty
}

// Save writes the state file atomically (temp → fsync → rename).
// In dry-run mode, this is a no-op.
func (m *Manager) Save() error {
	if m.dryRun {
		m.dirty = 0
		return nil
	}
	if m.state == nil {
		return errors.New("no state to save")
	}

	m.state.LastSync = time.Now().UTC()
	m.state.Version = version

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	tmpPath := m.statePath() + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp state file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("writing state: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("syncing state: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpPath, m.statePath()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming state file: %w", err)
	}

	m.dirty = 0
	return nil
}

// SaveIfNeeded saves the state if there are at least batchSize unsaved modifications.
func (m *Manager) SaveIfNeeded(batchSize int) error {
	if m.dirty >= batchSize {
		return m.Save()
	}
	return nil
}
