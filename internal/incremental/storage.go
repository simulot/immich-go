package incremental

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/simulot/immich-go/internal/fshelper"
)

const (
	storageWarning = uint64(50_000_000_000)
	storageDanger  = uint64(20_000_000_000)
	storageReserve = uint64(10_000_000_000)
)

var errStorageFallback = errors.New("catalog/re-present fallback selected")

// StorageStatus reports decimal-byte thresholds on the staging filesystem.
// Level is ok, warning (<=50 GB), or danger (<=20 GB). Fallback is sticky.
type StorageStatus struct {
	FreeBytes uint64
	Level     string
	Fallback  bool
	Reason    string
}

// WithStorageObserver reports each warning level and fallback transition
// immediately, at most once per level per importer. It must not reenter Importer.
func WithStorageObserver(observer func(StorageStatus)) Option {
	return func(options *stagingOptions) error {
		if observer == nil {
			return errors.New("storage observer must not be nil")
		}
		options.observer = observer
		return nil
	}
}

// WithAvailableBytes overrides the filesystem availability probe for this
// session. The default measures blocks available to the current user.
func WithAvailableBytes(probe func(string) (uint64, error)) Option {
	return func(options *stagingOptions) error {
		if probe == nil {
			return errors.New("available-bytes probe must not be nil")
		}
		options.available = probe
		return nil
	}
}

func (i *Importer) StorageStatus() StorageStatus {
	status := i.storageStatus
	if i.state.Staging != nil {
		status.Fallback, status.Reason = i.state.Staging.Fallback, i.state.Staging.FallbackReason
	}
	return status
}

func (i *Importer) observeStorage() (uint64, error) {
	free, err := i.availableBytes(i.state.Staging.Directory)
	if err != nil {
		return 0, fmt.Errorf("cannot determine available storage for %q; keep this ZIP and correct the filesystem probe: %w", i.state.Staging.Directory, err)
	}
	level := "ok"
	if free <= storageDanger {
		level = "danger"
	} else if free <= storageWarning {
		level = "warning"
	}
	i.storageStatus = StorageStatus{FreeBytes: free, Level: level, Fallback: i.state.Staging.Fallback, Reason: i.state.Staging.FallbackReason}
	if !i.storageLevels[level] {
		i.storageLevels[level] = true
		if i.storageObserver != nil {
			i.storageObserver(i.storageStatus)
		}
	}
	return free, nil
}

func (i *Importer) selectFallback(reason string) error {
	if !i.state.Staging.Fallback {
		i.state.Staging.Fallback, i.state.Staging.FallbackReason = true, reason
		if err := i.save(); err != nil {
			i.state.Staging.Fallback, i.state.Staging.FallbackReason = false, ""
			return err
		}
		i.storageStatus.Fallback, i.storageStatus.Reason = true, reason
		if i.storageObserver != nil {
			i.storageObserver(i.storageStatus)
		}
	}
	return fmt.Errorf("%w: %s", errStorageFallback, reason)
}

func (i *Importer) checkStorage(size uint64, purpose string) error {
	free, err := i.observeStorage()
	if err != nil {
		return err
	}
	if free < storageReserve || size > free-storageReserve {
		return i.selectFallback(fmt.Sprintf("%s requires %d bytes but only %d bytes are available with a 10 GB reserve; catalog/re-present remaining media. Second-pass uploads require guarded workspace with at least the media size plus 10 GB free", purpose, size, free))
	}
	return nil
}

// Only an exclusively fallback error may be swallowed. A joined cleanup,
// read, write, or state-save failure must still reach the caller.
func isStorageFallback(err error) bool {
	if err == errStorageFallback {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, cause := range joined.Unwrap() {
			if !isStorageFallback(cause) {
				return false
			}
		}
		return len(joined.Unwrap()) > 0
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return isStorageFallback(wrapped.Unwrap())
	}
	return false
}

type storageWriter struct {
	i       *Importer
	dst     io.Writer
	purpose string
}

func (w storageWriter) Write(p []byte) (int, error) {
	if err := w.i.checkStorage(uint64(len(p)), w.purpose); err != nil {
		return 0, err
	}
	return w.dst.Write(p)
}

func (i *Importer) cleanupStorageFile(name string) error {
	if err := i.staging.root.Remove(name); err != nil {
		return fmt.Errorf("cleanup unfinished staging file %q failed; keep this ZIP and restore directory access: %w", name, err)
	}
	return syncStage(i.staging.root)
}

// Ordinary ZIP uploads make a full temporary copy inside the shared uploader.
// Keep this PoC's transient copy on the guarded staging filesystem instead,
// then supply a regular *os.File so the uploader uses it without another copy.
// This workspace is excluded from the durable-payload cap, and is always removed.
func (i *Importer) uploadZIP(ctx context.Context, checksum, name string, source *zip.File, z *zip.Reader) (uploadErr error) {
	if i.staging == nil {
		return i.upload(ctx, checksum, name, fshelper.FSName(z, source.Name))
	}
	if err := i.cleanupStagingTemps(); err != nil {
		return err
	}
	m := i.state.Media[checksum]
	if err := i.checkStorage(m.Size, "upload workspace"); err != nil {
		return err
	}
	workspace := ".upload-" + i.state.Staging.ID + "-" + rand.Text()
	f, err := i.staging.root.OpenFile(workspace, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { uploadErr = errors.Join(uploadErr, i.cleanupStorageFile(workspace)) }()
	r, err := source.Open()
	if err != nil {
		return errors.Join(err, f.Close())
	}
	h := sha1.New()
	n, copyErr := io.CopyBuffer(io.MultiWriter(storageWriter{i, f, "upload workspace"}, h), contextReader{ctx, r}, make([]byte, 64<<10))
	err = errors.Join(copyErr, r.Close(), f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if uint64(n) != m.Size || base64.StdEncoding.EncodeToString(h.Sum(nil)) != checksum {
		return errors.New("upload workspace failed media size/checksum validation; keep this ZIP")
	}
	return i.upload(ctx, checksum, name, fshelper.FSName(i.staging.root.FS(), workspace))
}

func (i *Importer) ownedWorkspace(name string) bool {
	return strings.HasPrefix(name, ".upload-"+i.state.Staging.ID+"-")
}
