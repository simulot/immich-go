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
	filename string
	state    State
	client   Client
	lock     *os.File
}

// ImportZIP scans metadata and checksums before uploading anything. Only small
// metadata and receipts survive the call; deferred media must be presented again.
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
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() || !safePath(f.Name) || !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(f.Name)) {
			continue
		}
		if counts[f.Name] > 1 {
			i.problem(f.Name, "duplicate ZIP entry path; association ambiguous, item deferred")
			continue
		}
		a := &assets.Asset{File: fshelper.FSName(&z.Reader, f.Name)}
		checksum, err := a.GetChecksum()
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
			i.problem(f.Name, "ambiguous original path has different media; use a separate export state")
			continue
		}
		if old, ok := i.state.Media[checksum]; ok && old.Size != f.UncompressedSize64 {
			i.problem(f.Name, "checksum matches media with a different size; item deferred")
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
	// The deferred occurrences and sidecars are durable even if an upload fails.
	if err := i.save(); err != nil {
		return err
	}
	for _, checksum := range sortedKeys(media) {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		m := i.state.Media[checksum]
		name, _, reason := i.resolve(checksum)
		if m.AssetID != "" || reason != "" {
			continue
		}
		if m.Attempted && name != m.CanonicalName {
			continue // resolve reports this as an immutable-name conflict.
		}
		previousAttempt := m.Attempted
		m.CanonicalName, m.Attempted = name, true
		m.Reason = "upload outcome unknown; receipt must remain unowned on retry"
		i.state.Media[checksum] = m
		if err := i.save(); err != nil {
			return err
		}
		f := media[checksum]
		a := &assets.Asset{File: fshelper.FSName(&z.Reader, f.Name), OriginalFileName: name, Checksum: checksum}
		result, uploadErr := i.client.AssetUpload(ctx, a)
		closeErr := a.Close()
		if uploadErr != nil || result.ID == "" || (result.Status != immich.UploadCreated && result.Status != immich.UploadDuplicate) {
			if uploadErr == nil {
				uploadErr = errors.New("invalid upload receipt")
			}
			failures = append(failures, fmt.Errorf("upload %s (retry this ZIP; unknown outcomes remain unowned): %w", f.Name, errors.Join(uploadErr, closeErr)))
			continue
		}
		m.AssetID = result.ID
		m.Reason = "preexisting or unknown-outcome duplicate; metadata withheld"
		// Save a known receipt as unowned before fetching a baseline.
		i.state.Media[checksum] = m
		if err := i.save(); err != nil {
			return err
		}
		if result.Status == immich.UploadCreated && !previousAttempt {
			current, err := i.client.GetAssetInfo(ctx, result.ID)
			if err != nil || current == nil {
				if err == nil {
					err = errors.New("empty asset response")
				}
				m.Reason = "uploaded but baseline unavailable; receipt retained as unowned"
				i.state.Media[checksum] = m
				if saveErr := i.save(); saveErr != nil {
					return saveErr
				}
				failures = append(failures, fmt.Errorf("%s for %s: %w", m.Reason, f.Name, err))
				continue
			}
			baseline := values(current)
			m.Baseline, m.Owned, m.Reason = &baseline, true, ""
			i.state.Media[checksum] = m
			if err := i.save(); err != nil {
				return err
			}
		}
		if closeErr != nil {
			failures = append(failures, closeErr)
		}
	}
	return errors.Join(append(failures, i.Reconcile(ctx))...)
}

func (i *Importer) problem(name, reason string) {
	i.state.Problems[name] = addString(i.state.Problems[name], reason)
}
