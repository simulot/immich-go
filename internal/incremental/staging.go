package incremental

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Staging binds a private directory to the exact state-file location. Both must
// stay in place. Omitting WithStaging on subsequent opens restores this config.
// To increase capacity, supply the same directory with a larger limit.
type Staging struct {
	Directory      string
	StateFile      string
	ID             string
	MaxBytes       int64
	Fallback       bool
	FallbackReason string
}

type stagingOptions struct {
	dir       string
	limit     int64
	observer  func(StorageStatus)
	available func(string) (uint64, error)
}

type Option func(*stagingOptions) error

// WithStaging retains unresolved media until a durable upload receipt exists.
// The directory must be private and empty on first use, and is exclusively locked.
// The limit covers retained payloads and the staging write in progress, excluding
// input ZIPs and ordinary upload temporary files. No payloads are evicted.
// A persisted cap may stay the same or increase; decreases are rejected.
func WithStaging(directory string, maxBytes int64) Option {
	return func(options *stagingOptions) error {
		if strings.TrimSpace(directory) == "" || maxBytes <= 0 {
			return errors.New("staging requires a nonempty directory and a positive max-staging-bytes limit")
		}
		if options.dir != "" {
			return errors.New("staging option supplied more than once")
		}
		options.dir, options.limit = directory, maxBytes
		return nil
	}
}

type stagingStore struct {
	root *os.Root
	lock *os.File
	used int64
}

const (
	stagingOwner = ".immich-go-owner.json"
	stagingLock  = ".immich-go.lock"
)

// Resolve ancestor aliases (for example the system temporary directory), but
// refuse a symlink at the actual state/directory location.
func canonicalLocation(name string) (string, error) {
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(abs); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing symlink at %s", abs)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

func (i *Importer) openStaging(options stagingOptions) error {
	if options.dir == "" && i.state.Staging == nil {
		return nil
	}
	if err := stagingSupported(); err != nil {
		return err
	}
	config := i.state.Staging
	if options.dir != "" {
		// Create only parents here. An existing staging directory is inspected
		// before use and must never be replaced or followed through a symlink.
		if err := os.MkdirAll(filepath.Dir(options.dir), 0o700); err != nil {
			return err
		}
		dir, err := canonicalLocation(options.dir)
		if err != nil {
			return err
		}
		if config != nil && dir != config.Directory {
			return errors.New("staging directory differs from the persisted binding; keep the state and staging directory at their original locations")
		}
		if config == nil {
			config = &Staging{Directory: dir, StateFile: i.filename, ID: rand.Text(), MaxBytes: options.limit}
		} else {
			if options.limit < config.MaxBytes {
				return fmt.Errorf("max-staging-bytes cannot decrease from the persisted cap of %d bytes to %d bytes; use the same cap or increase it", config.MaxBytes, options.limit)
			}
			updatedConfig := *config
			config = &updatedConfig
			config.MaxBytes = options.limit
		}
	}
	if config.StateFile != i.filename || !filepath.IsAbs(config.Directory) || config.MaxBytes <= 0 ||
		len(config.ID) != 26 || strings.ContainsAny(config.ID, "/\\.") {
		return errors.New("invalid staging binding or copied/moved state; restore the original state-file and staging locations")
	}
	dir, err := canonicalLocation(config.Directory)
	if err != nil {
		return err
	}
	if dir != config.Directory {
		return errors.New("staging directory location changed")
	}
	newBinding := i.state.Staging == nil
	if newBinding {
		if err := os.Mkdir(config.Directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := syncStagingParent(config.Directory); err != nil {
			return err
		}
	}
	info, err := os.Stat(config.Directory)
	if err != nil {
		return fmt.Errorf("staging directory unavailable; restore it before continuing: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("staging directory must be a private directory (permissions 0700)")
	}
	root, err := os.OpenRoot(config.Directory)
	if err != nil {
		return err
	}
	i.staging = &stagingStore{root: root}
	lock, err := root.OpenFile(stagingLock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("staging is locked; after a crash verify no importer is running before removing %s: %w", filepath.Join(config.Directory, stagingLock), err)
	}
	i.staging.lock = lock
	if newBinding {
		entries, err := readStageDir(root)
		if err != nil {
			return err
		}
		if len(entries) != 1 || entries[0].Name() != stagingLock {
			return errors.New("new staging directory must be empty; existing contents may belong to another state")
		}
		// Persist the identity before publishing its marker. If interrupted with
		// an empty directory, the same state can safely finish initialization.
		i.state.Staging = config
		if err := i.save(); err != nil {
			return err
		}
	}
	owner, err := regularStageFile(root, stagingOwner)
	if errors.Is(err, os.ErrNotExist) {
		entries, listErr := readStageDir(root)
		if listErr != nil {
			return listErr
		}
		if len(entries) != 1 || entries[0].Name() != stagingLock {
			return errors.New("staging ownership marker missing from nonempty directory; preserve contents for recovery")
		}
		data, marshalErr := json.Marshal(struct{ ID, StateFile string }{config.ID, config.StateFile})
		if marshalErr != nil {
			return marshalErr
		}
		f, createErr := root.OpenFile(stagingOwner, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			return createErr
		}
		_, writeErr := f.Write(data)
		if err := errors.Join(writeErr, f.Sync(), f.Close(), syncStage(root)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if owner.Size() > 4096 {
			return errors.New("invalid staging ownership marker")
		}
		data, err := root.ReadFile(stagingOwner)
		if err != nil {
			return err
		}
		var binding struct{ ID, StateFile string }
		if err := json.Unmarshal(data, &binding); err != nil || binding.ID != config.ID || binding.StateFile != config.StateFile {
			return errors.New("staging ownership does not match this state; refusing to use or delete its contents")
		}
	}
	i.state.Staging = config
	if err := i.recoverStaging(); err != nil {
		return err
	}
	if i.staging.used > config.MaxBytes {
		return fmt.Errorf("retained %d bytes exceed staging cap %d bytes; preserve the cache and reopen with the same --staging-dir and a larger --max-staging-bytes", i.staging.used, config.MaxBytes)
	}
	return nil
}

func readStageDir(root *os.Root) ([]os.DirEntry, error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.ReadDir(-1)
}

func regularStageFile(root *os.Root, name string) (os.FileInfo, error) {
	info, err := root.Lstat(name)
	if err == nil && !info.Mode().IsRegular() {
		err = fmt.Errorf("staging entry %q must be a regular file; symlinks and directories are refused", name)
	}
	return info, err
}

func (s *stagingStore) close() error {
	var err error
	if s.lock != nil {
		err = errors.Join(s.lock.Close(), s.root.Remove(stagingLock))
	}
	return errors.Join(err, s.root.Close())
}

func payloadName(checksum string) (string, error) {
	digest, err := base64.StdEncoding.DecodeString(checksum)
	if err != nil || len(digest) != sha1.Size {
		return "", errors.New("invalid staging checksum")
	}
	return hex.EncodeToString(digest) + ".payload", nil
}

func (i *Importer) recoverStaging() error {
	if err := i.cleanupStagingTemps(); err != nil {
		return err
	}
	entries, err := readStageDir(i.staging.root)
	if err != nil {
		return err
	}
	names := map[string]string{}
	for checksum := range i.state.Media {
		name, err := payloadName(checksum)
		if err != nil {
			return err
		}
		names[name] = checksum
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == stagingLock || name == stagingOwner {
			continue
		}
		if _, err := regularStageFile(i.staging.root, name); err != nil {
			return err
		}
		checksum, known := names[name]
		if !known {
			return fmt.Errorf("unrecognized staging entry %q; preserved, not deleted", name)
		}
		m := i.state.Media[checksum]
		// A publish followed by a crash before reference-save is recoverable.
		// Verification happens before upload, including for recovered references.
		m.Staged = true
		i.state.Media[checksum] = m
	}
	return syncStage(i.staging.root)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

// ZIP's reader verifies CRC and advertised size at EOF. Independently compare
// actual bytes with the advertisement before using that size for any budget.
func checksumZIP(ctx context.Context, f *zip.File) (string, error) {
	r, err := f.Open()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	n, readErr := io.CopyBuffer(h, contextReader{ctx, r}, make([]byte, 64<<10))
	if err := errors.Join(readErr, r.Close()); err != nil {
		return "", err
	}
	if uint64(n) != f.UncompressedSize64 {
		return "", errors.New("ZIP advertised media size differs from actual bytes")
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (i *Importer) verifyStaged(ctx context.Context, m Media) error {
	name, err := payloadName(m.Checksum)
	if err != nil {
		return err
	}
	info, err := regularStageFile(i.staging.root, name)
	if err != nil {
		return fmt.Errorf("staged payload %s unavailable; restore the cache or re-present the retained media ZIP: %w", name, err)
	}
	if uint64(info.Size()) != m.Size {
		return fmt.Errorf("staged payload %s is corrupt: size differs; re-present the retained media ZIP", name)
	}
	f, err := i.staging.root.Open(name)
	if err != nil {
		return err
	}
	h := sha1.New()
	_, readErr := io.CopyBuffer(h, contextReader{ctx, f}, make([]byte, 64<<10))
	if err := errors.Join(readErr, f.Close()); err != nil {
		return err
	}
	if base64.StdEncoding.EncodeToString(h.Sum(nil)) != m.Checksum {
		return fmt.Errorf("staged payload %s is corrupt: checksum differs; re-present the retained media ZIP", name)
	}
	return nil
}

func (i *Importer) stage(ctx context.Context, checksum string, source *zip.File) (stageErr error) {
	// A failed write may have left an owned partial file. Recover it before
	// admitting another copy; failed cleanup keeps its actual bytes charged.
	if err := i.cleanupStagingTemps(); err != nil {
		return err
	}
	if i.state.Staging.Fallback {
		return fmt.Errorf("%w: %s", errStorageFallback, i.state.Staging.FallbackReason)
	}
	m := i.state.Media[checksum]
	if m.Staged {
		if err := i.verifyStaged(ctx, m); err == nil {
			return nil
		}
	}
	name, err := payloadName(checksum)
	if err != nil {
		return err
	}
	// A corrupt old copy remains charged while its replacement is written.
	if m.Size >= math.MaxInt64 || i.staging.used > i.state.Staging.MaxBytes || int64(m.Size) > i.state.Staging.MaxBytes-i.staging.used {
		return i.selectFallback(fmt.Sprintf("staging capacity exhausted: retained %d bytes, next payload %d bytes, cap %d bytes; catalog/re-present remaining media without eviction or automatic re-download. Second-pass uploads need guarded workspace with the media size plus 10 GB free", i.staging.used, m.Size, i.state.Staging.MaxBytes))
	}
	if err := i.checkStorage(m.Size, "durable staging"); err != nil {
		return err
	}
	temp := ".tmp-" + i.state.Staging.ID + "-" + rand.Text()
	f, err := i.staging.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err := i.staging.root.Remove(temp); err == nil {
			stageErr = errors.Join(stageErr, syncStage(i.staging.root))
		} else if !errors.Is(err, os.ErrNotExist) {
			stageErr = errors.Join(stageErr, fmt.Errorf("cleanup unfinished staging file %q failed; retained bytes remain charged, keep this ZIP and restore directory access: %w", temp, err))
		}
		stageErr = errors.Join(stageErr, i.recountStaging())
	}()
	r, err := source.Open()
	if err != nil {
		return errors.Join(err, f.Close())
	}
	h := sha1.New()
	n, copyErr := io.CopyBuffer(io.MultiWriter(storageWriter{i, f, "durable staging"}, h), contextReader{ctx, io.LimitReader(r, int64(m.Size)+1)}, make([]byte, 64<<10))
	err = errors.Join(copyErr, r.Close())
	if err == nil && (uint64(n) != m.Size || base64.StdEncoding.EncodeToString(h.Sum(nil)) != checksum) {
		err = errors.New("media changed or failed size/checksum validation during staging")
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return fmt.Errorf("cannot retain media in staging; keep this ZIP and check free disk space: %w", err)
	}
	var oldSize int64
	if info, err := regularStageFile(i.staging.root, name); err == nil {
		oldSize = info.Size()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := i.staging.root.Rename(temp, name); err != nil {
		return err
	}
	i.staging.used += n - oldSize
	if err := syncStage(i.staging.root); err != nil {
		return err
	}
	m.Staged = true
	i.state.Media[checksum] = m
	return i.save()
}

func (i *Importer) cleanupStagingTemps() error {
	entries, err := readStageDir(i.staging.root)
	if err != nil {
		return err
	}
	var failures []error
	removed := false
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".tmp-"+i.state.Staging.ID+"-") && !i.ownedWorkspace(name) {
			continue
		}
		if _, err := regularStageFile(i.staging.root, name); err != nil {
			failures = append(failures, err)
			continue
		}
		if err := i.staging.root.Remove(name); err != nil {
			failures = append(failures, fmt.Errorf("cleanup unfinished staging file %q failed; retained bytes remain charged, keep this ZIP and restore directory access: %w", name, err))
			continue
		}
		removed = true
	}
	if removed {
		failures = append(failures, syncStage(i.staging.root))
	}
	return errors.Join(append(failures, i.recountStaging())...)
}

func (i *Importer) recountStaging() error {
	var used int64
	for checksum := range i.state.Media {
		name, err := payloadName(checksum)
		if err != nil {
			return err
		}
		info, err := regularStageFile(i.staging.root, name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Size() > math.MaxInt64-used {
			return errors.New("staging byte count overflow")
		}
		used += info.Size()
	}
	entries, err := readStageDir(i.staging.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".tmp-"+i.state.Staging.ID+"-") {
			continue
		}
		info, err := regularStageFile(i.staging.root, entry.Name())
		if err != nil {
			return err
		}
		if info.Size() > math.MaxInt64-used {
			return errors.New("staging byte count overflow")
		}
		used += info.Size()
	}
	i.staging.used = used
	return nil
}

// Receipt-save precedes deletion. Staged stays true until deletion is synced;
// restart can finish either side of this cleanup without uploading again.
func (i *Importer) releaseStaged(checksum string) error {
	m := i.state.Media[checksum]
	if !m.Staged || m.AssetID == "" {
		return nil
	}
	name, err := payloadName(checksum)
	if err != nil {
		return err
	}
	info, err := regularStageFile(i.staging.root, name)
	if err == nil {
		if err := i.staging.root.Remove(name); err != nil {
			return err
		}
		i.staging.used -= info.Size()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncStage(i.staging.root); err != nil {
		return err
	}
	m.Staged = false
	i.state.Media[checksum] = m
	return i.save()
}
