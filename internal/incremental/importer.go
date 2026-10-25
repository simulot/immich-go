// Package incremental implements an experimental, conservatively matched Takeout importer.
package incremental

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
)

type Client interface {
	AssetUpload(context.Context, *assets.Asset) (immich.AssetResponse, error)
	GetAssetInfo(context.Context, string) (*immich.Asset, error)
	UpdateAsset(context.Context, string, immich.UpdAssetField) (*immich.Asset, error)
}

type Importer struct {
	filename        string
	state           State
	client          Client
	lock            *os.File
	staging         *stagingStore
	availableBytes  func(string) (uint64, error)
	storageObserver func(StorageStatus)
	storageStatus   StorageStatus
	storageLevels   map[string]bool
}

// ImportZIP scans metadata and checksums before uploading anything. With staging,
// deferred media is retained; otherwise it must be presented again.
// ZIP files and source media are never deleted by this package.
func (i *Importer) ImportZIP(ctx context.Context, filename string) error {
	z, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer z.Close()
	counts := map[string]int{}
	for _, f := range z.File {
		if !f.FileInfo().IsDir() {
			counts[f.Name]++
		}
	}
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !safePath(f.Name) {
			i.problem(f.Name, "unsafe archive path; item deferred")
			continue
		}
		if isSidecar(f.Name) {
			i.catalogSidecar(&z.Reader, f, counts[f.Name] > 1)
		}
	}
	// Keep ZIP entry pointers only during this call. Hash every occurrence before
	// deciding names so duplicate paths within this archive contribute metadata.
	media := map[string]*zip.File{}
	var failures []error
	retentionFailures := map[string]string{}
	deferMedia := func(name, reason string) {
		i.problem(name, reason)
		if i.staging != nil {
			retentionFailures[name] = reason
		}
	}
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() || !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(f.Name)) {
			continue
		}
		if !safePath(f.Name) {
			deferMedia(f.Name, "unsafe archive path; item deferred")
			continue
		}
		if counts[f.Name] > 1 {
			deferMedia(f.Name, "duplicate ZIP entry path; association ambiguous, item deferred")
			continue
		}
		checksum, err := checksumZIP(ctx, f)
		if err != nil {
			i.problem(f.Name, "cannot checksum media: "+err.Error()+"; item deferred")
			failures = append(failures, fmt.Errorf("checksum %s: %w", f.Name, err))
			continue
		}
		// A repaired or re-downloaded source can be retried without retaining a
		// transient read failure as a permanent matching conflict.
		i.state.Problems[f.Name] = slices.DeleteFunc(i.state.Problems[f.Name], func(reason string) bool {
			return strings.HasPrefix(reason, "cannot checksum media: ")
		})
		if len(i.state.Problems[f.Name]) == 0 {
			delete(i.state.Problems, f.Name)
		}
		if old, ok := i.state.Occurrences[f.Name]; ok && old.Checksum != checksum {
			old.ConflictingChecksums = addString(old.ConflictingChecksums, checksum)
			i.state.Occurrences[f.Name] = old
			deferMedia(f.Name, "ambiguous original path has different media; use a separate export state")
			continue
		}
		if old, ok := i.state.Media[checksum]; ok && old.Size != f.UncompressedSize64 {
			deferMedia(f.Name, "checksum matches media with a different size; item deferred")
			continue
		}
		i.state.Occurrences[f.Name] = Occurrence{Checksum: checksum, ConflictingChecksums: i.state.Occurrences[f.Name].ConflictingChecksums}
		if _, ok := i.state.Media[checksum]; !ok {
			i.state.Media[checksum] = Media{Checksum: checksum, Size: f.UncompressedSize64}
		}
		if _, ok := media[checksum]; !ok {
			media[checksum] = f
		}
	}
	// An issue alone cannot certify that a staged import retained this ZIP's
	// supported media. Reject ambiguous/unsafe sources without guessing an
	// association, while letting healthy inventory, cache and uploads proceed.
	for _, source := range sortedKeys(retentionFailures) {
		failures = append(failures, fmt.Errorf("supported media %q was not safely cataloged/retained: %s; keep this ZIP %q", source, retentionFailures[source], filename))
	}
	// The deferred occurrences and sidecars are durable even if an upload fails.
	if err := i.save(); err != nil {
		return err
	}
	// Metadata from this archive can free cached bytes before new staging writes.
	if err := i.drainStaged(ctx); err != nil {
		failures = append(failures, err)
	}
	for _, checksum := range sortedKeys(media) {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		m := i.state.Media[checksum]
		name, _, reason := i.resolve(checksum)
		if m.AssetID != "" {
			continue
		}
		if m.Staged && i.verifyStaged(ctx, m) == nil {
			continue // Already drained, deferred or failed once in this call.
		}
		f := media[checksum]
		if reason != "" {
			if i.staging != nil {
				if err := i.stage(ctx, checksum, f); err != nil && !isStorageFallback(err) {
					failures = append(failures, fmt.Errorf("stage %s: %w", f.Name, err))
				}
			}
			continue
		}
		if err := i.uploadZIP(ctx, checksum, name, f, &z.Reader); err != nil {
			if isStorageFallback(err) {
				continue
			}
			failures = append(failures, err)
			// A failed direct upload can still avoid a second download if space
			// permits. Uncertain outcomes keep the existing ownership rules.
			if i.staging != nil && i.state.Media[checksum].AssetID == "" {
				if err := i.stage(ctx, checksum, f); err != nil && !isStorageFallback(err) {
					failures = append(failures, err)
				}
			}
		}
	}
	return errors.Join(append(failures, i.reconcileMetadata(ctx))...)
}

func (i *Importer) upload(ctx context.Context, checksum, name string, source fshelper.FSAndName) error {
	m := i.state.Media[checksum]
	previousAttempt := m.Attempted
	m.CanonicalName, m.Attempted = name, true
	m.Reason = "upload outcome unknown; receipt must remain unowned on retry"
	i.state.Media[checksum] = m
	if err := i.save(); err != nil {
		return err
	}
	a := &assets.Asset{File: source, OriginalFileName: name, Checksum: checksum}
	result, uploadErr := i.client.AssetUpload(ctx, a)
	closeErr := a.Close()
	if uploadErr != nil || result.ID == "" || (result.Status != immich.UploadCreated && result.Status != immich.UploadDuplicate) {
		if uploadErr == nil {
			uploadErr = errors.New("invalid upload receipt")
		}
		return fmt.Errorf("upload %s (retry cached media or this ZIP; unknown outcomes remain unowned): %w", source.Name(), errors.Join(uploadErr, closeErr))
	}
	m.AssetID = result.ID
	m.Reason = "preexisting or unknown-outcome duplicate; metadata withheld"
	// Save a known receipt as unowned before fetching a baseline or deleting bytes.
	i.state.Media[checksum] = m
	if err := i.save(); err != nil {
		// Do not let another operation in this session treat an undurable
		// receipt as permission to delete the only retained payload.
		m.AssetID = ""
		m.Reason = "upload outcome unknown; receipt must remain unowned on retry"
		i.state.Media[checksum] = m
		return err
	}
	var failures []error
	if result.Status == immich.UploadCreated && !previousAttempt {
		current, err := i.client.GetAssetInfo(ctx, result.ID)
		if err != nil || current == nil {
			if err == nil {
				err = errors.New("empty asset response")
			}
			m.Reason = "uploaded but baseline unavailable; receipt retained as unowned"
			failures = append(failures, fmt.Errorf("%s for %s: %w", m.Reason, source.Name(), err))
		} else {
			baseline := values(current)
			m.Baseline, m.Owned, m.Reason = &baseline, true, ""
		}
		i.state.Media[checksum] = m
		if err := i.save(); err != nil {
			return err
		}
	}
	return errors.Join(append(failures, closeErr, i.releaseStaged(checksum))...)
}

func (i *Importer) drainStaged(ctx context.Context) error {
	if i.staging == nil {
		return nil
	}
	var failures []error
	if err := i.cleanupStagingTemps(); err != nil {
		failures = append(failures, err)
	}
	for _, checksum := range sortedKeys(i.state.Media) {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		m := i.state.Media[checksum]
		if !m.Staged {
			continue
		}
		if m.AssetID != "" {
			if err := i.releaseStaged(checksum); err != nil {
				failures = append(failures, err)
			}
			continue
		}
		if err := i.verifyStaged(ctx, m); err != nil {
			failures = append(failures, err)
			continue
		}
		name, _, reason := i.resolve(checksum)
		if reason != "" {
			continue
		}
		payload, err := payloadName(checksum)
		if err == nil {
			err = i.upload(ctx, checksum, name, fshelper.FSName(i.staging.root.FS(), payload))
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (i *Importer) problem(name, reason string) {
	i.state.Problems[name] = addString(i.state.Problems[name], reason)
}
