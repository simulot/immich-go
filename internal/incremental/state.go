package incremental

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateVersion = 3

type Scope struct {
	Server   string
	UserID   string
	ExportID string
}

// Media holds one receipt per checksum, regardless of the number of source paths.
// Attempted is persisted before upload: a lost acknowledgement must never confer ownership.
type Media struct {
	Checksum      string
	Size          uint64
	CanonicalName string
	Attempted     bool
	AssetID       string
	Owned         bool
	Baseline      *Values
	Applied       *Values
	Reason        string
	Staged        bool `json:",omitempty"`
}

type Occurrence struct {
	Checksum             string
	ConflictingChecksums []string
}

type Values struct {
	Date        time.Time
	Description string
}

// Metadata keeps distinct candidates so neither arrival order nor absent fields
// can silently resolve a disagreement. It contains no media or archive bytes.
type Metadata struct {
	Names        []string
	Dates        []time.Time
	Descriptions []string
	Problems     []string
}

type State struct {
	Version     int
	Scope       Scope
	Media       map[string]Media
	Occurrences map[string]Occurrence
	Metadata    map[string]Metadata
	Problems    map[string][]string
	Staging     *Staging     `json:",omitempty"`
	Archives    *ArchivePlan `json:",omitempty"`
}

// Open exclusively locks an export state, binding it to one server and account.
func Open(filename string, scope Scope, client Client, options ...Option) (*Importer, error) {
	var config stagingOptions
	for _, option := range options {
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	if scope.Server == "" || scope.UserID == "" || scope.ExportID == "" {
		return nil, errors.New("server, user and export ID must be set")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return nil, err
	}
	filename, err := canonicalLocation(filename)
	if err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filename+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("state is locked (after a crash, verify no importer is running before removing %s.lock): %w", filename, err)
	}
	i := &Importer{filename: filename, client: client, lock: lock, state: State{
		Version: stateVersion, Scope: scope, Media: map[string]Media{}, Occurrences: map[string]Occurrence{},
		Metadata: map[string]Metadata{}, Problems: map[string][]string{},
	}}
	i.availableBytes, i.storageObserver = config.available, config.observer
	if i.availableBytes == nil {
		i.availableBytes = availableStagingBytes
	}
	i.storageLevels = map[string]bool{}
	data, err := os.ReadFile(filename)
	if err == nil {
		// Check the version before decoding the changed receipt and metadata shapes.
		var header struct{ Version int }
		err = json.Unmarshal(data, &header)
		if err == nil && header.Version != stateVersion && header.Version != 2 {
			err = fmt.Errorf("unsupported incremental state version %d; version 1 lacks canonical filenames and may contain permanently misnamed uploads. Preserve this state, audit existing uploads manually, and use a new state file/export ID; automatic migration is unsafe", header.Version)
		}
		if err == nil {
			var persisted State
			err = json.Unmarshal(data, &persisted)
			i.state = persisted
		}
		if err == nil {
			err = i.validate(scope)
		}
		if err == nil && header.Version == 2 {
			if i.state.Staging != nil {
				err = errors.New("version 2 state must not contain staging configuration")
			}
			i.state.Version = stateVersion
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = i.save()
	}
	if err == nil {
		err = i.openStaging(config)
	}
	if err == nil && i.staging != nil {
		var free uint64
		free, err = i.observeStorage()
		if err == nil && free <= storageReserve {
			err = i.selectFallback(fmt.Sprintf("available storage is %d bytes, at or below the 10 GB reserve; catalog/re-present remaining media. Second-pass uploads require guarded workspace with the media size plus 10 GB free", free))
			if isStorageFallback(err) {
				err = nil
			}
		}
	}
	if err == nil {
		err = i.save()
	}
	if err != nil {
		_ = i.Close()
		return nil, err
	}
	return i, nil
}

func (i *Importer) validate(scope Scope) error {
	if i.state.Scope != scope || i.state.Media == nil || i.state.Occurrences == nil || i.state.Metadata == nil || i.state.Problems == nil {
		return errors.New("state scope or contents do not match this import")
	}
	for checksum, m := range i.state.Media {
		if checksum == "" || m.Checksum != checksum || (m.CanonicalName != "" && !safeName(m.CanonicalName)) ||
			(m.Attempted && m.CanonicalName == "") || (m.AssetID != "" && !m.Attempted) ||
			(m.Owned && (m.AssetID == "" || m.Baseline == nil)) || (m.Applied != nil && (!m.Owned || !validDate(m.Applied.Date))) {
			return fmt.Errorf("invalid persisted logical media receipt for %q", checksum)
		}
		if m.Staged && (i.state.Version != stateVersion || i.state.Staging == nil) {
			return fmt.Errorf("staged media %q has no valid staging binding", checksum)
		}
	}
	for name, occurrence := range i.state.Occurrences {
		if !safePath(name) || occurrence.Checksum == "" {
			return fmt.Errorf("invalid persisted source occurrence for %q", name)
		}
		if _, ok := i.state.Media[occurrence.Checksum]; !ok {
			return fmt.Errorf("missing logical media receipt for %q", name)
		}
	}
	for name, md := range i.state.Metadata {
		if !safePath(name) {
			return fmt.Errorf("invalid persisted metadata path %q", name)
		}
		for _, title := range md.Names {
			if !safeName(title) {
				return fmt.Errorf("invalid persisted canonical filename for %q", name)
			}
		}
		for _, date := range md.Dates {
			if !validDate(date) {
				return fmt.Errorf("invalid persisted metadata date for %q", name)
			}
		}
	}
	return nil
}

func (i *Importer) Close() error {
	if i.lock == nil {
		return nil
	}
	var stageErr error
	if i.staging != nil {
		stageErr = i.staging.close()
		i.staging = nil
	}
	err := errors.Join(stageErr, i.lock.Close(), os.Remove(i.filename+".lock"))
	i.lock = nil
	return err
}

func (i *Importer) save() error {
	data, err := json.MarshalIndent(i.state, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(i.filename), ".incremental-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return replaceState(name, i.filename)
}
