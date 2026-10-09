// Package incremental implements an experimental, exact-path Takeout importer.
package incremental

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
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

type Scope struct {
	Server   string
	UserID   string
	ExportID string
}
type Media struct {
	Checksum string
	AssetID  string
	Owned    bool
	Baseline *Values
	Applied  *Values
}
type Values struct {
	Date        time.Time
	Description string
}
type State struct {
	Version  int
	Scope    Scope
	Media    map[string]Media
	Metadata map[string]Values
}

type Importer struct {
	filename string
	state    State
	client   Client
	lock     *os.File
}

// Open exclusively locks an export state, binding it to one server and account.
func Open(filename string, scope Scope, client Client) (*Importer, error) {
	if scope.Server == "" || scope.UserID == "" || scope.ExportID == "" {
		return nil, errors.New("server, user and export ID must be set")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filename+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("state is locked (after a crash, verify no importer is running before removing %s.lock): %w", filename, err)
	}
	i := &Importer{filename: filename, client: client, lock: lock, state: State{Version: 1, Scope: scope, Media: map[string]Media{}, Metadata: map[string]Values{}}}
	data, err := os.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(data, &i.state)
		if err == nil && (i.state.Version != 1 || i.state.Scope != scope || i.state.Media == nil || i.state.Metadata == nil) {
			err = errors.New("state version, scope, or contents do not match this import")
		}
	} else if errors.Is(err, os.ErrNotExist) {
		err = i.save()
	}
	if err == nil {
		for name, m := range i.state.Media {
			if !fs.ValidPath(name) || m.AssetID == "" || m.Checksum == "" || (m.Owned && m.Baseline == nil) {
				err = fmt.Errorf("invalid persisted media receipt for %q", name)
				break
			}
		}
	}
	if err != nil {
		_ = i.Close()
		return nil, err
	}
	return i, nil
}

func (i *Importer) Close() error { return errors.Join(i.lock.Close(), os.Remove(i.filename+".lock")) }

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

// ImportZIP retains only metadata and receipts; media is opened one item at a time.
// ZIP files and source media are never deleted by this package.
func (i *Importer) ImportZIP(ctx context.Context, filename string) error {
	z, err := zip.OpenReader(filename)
	if err != nil {
		return err
	}
	defer z.Close()
	// Catalog all sidecars first, independent of their order within this archive.
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !fs.ValidPath(f.Name) {
			return fmt.Errorf("invalid archive path %q", f.Name)
		}
		if !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		if f.UncompressedSize64 > 4<<20 {
			return fmt.Errorf("sidecar too large: %s", f.Name)
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(io.LimitReader(r, (4<<20)+1))
		closeErr := r.Close()
		if err = errors.Join(err, closeErr); err != nil {
			return err
		}
		if len(data) > 4<<20 {
			return fmt.Errorf("sidecar too large: %s", f.Name)
		}
		var md gp.GoogleMetaData
		if err = json.Unmarshal(data, &md); err != nil {
			return fmt.Errorf("sidecar %s: %w", f.Name, err)
		}
		if md.PhotoTakenTime == nil {
			continue
		} // Album metadata is outside this PoC.
		if _, err := strconv.ParseInt(md.PhotoTakenTime.Timestamp, 10, 64); err != nil {
			return fmt.Errorf("invalid capture timestamp in %s: %w", f.Name, err)
		}
		target := strings.TrimSuffix(strings.TrimSuffix(f.Name, ".json"), ".supplemental-metadata")
		if path.Base(target) != md.Title {
			return fmt.Errorf("unsupported renamed/truncated sidecar %s (title %q); no fuzzy match attempted", f.Name, md.Title)
		}
		metadata := md.AsMetadata(fshelper.FSName(&z.Reader, f.Name), false)
		if metadata.DateTaken.IsZero() || metadata.DateTaken.Year() < 1 || metadata.DateTaken.Year() > 9999 {
			return fmt.Errorf("missing or invalid capture time: %s", f.Name)
		}
		v := Values{Date: metadata.DateTaken.UTC(), Description: metadata.Description}
		if old, ok := i.state.Metadata[target]; ok && !old.equal(v) {
			return fmt.Errorf("conflicting sidecars for %s", target)
		}
		i.state.Metadata[target] = v
	}
	if err = i.save(); err != nil {
		return err
	}
	for _, f := range z.File {
		if err = ctx.Err(); err != nil {
			return err
		}
		if f.FileInfo().IsDir() || !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(f.Name)) {
			continue
		}
		a := &assets.Asset{File: fshelper.FSName(&z.Reader, f.Name), OriginalFileName: path.Base(f.Name)}
		checksum, err := a.GetChecksum()
		if err != nil {
			return err
		}
		m, ok := i.state.Media[f.Name]
		if ok && m.Checksum != checksum {
			return fmt.Errorf("ambiguous original path %s has different media; use a separate export state", f.Name)
		}
		if !ok {
			result, err := i.client.AssetUpload(ctx, a)
			closeErr := a.Close()
			if err = errors.Join(err, closeErr); err != nil {
				return fmt.Errorf("upload %s (retry this ZIP; unknown outcomes are treated as unowned duplicates): %w", f.Name, err)
			}
			if result.ID == "" || (result.Status != immich.UploadCreated && result.Status != immich.UploadDuplicate) {
				return fmt.Errorf("invalid upload receipt for %s", f.Name)
			}
			m = Media{Checksum: checksum, AssetID: result.ID, Owned: false}
			i.state.Media[f.Name] = m
			if err = i.save(); err != nil {
				return err
			}
			if result.Status == immich.UploadCreated {
				current, err := i.client.GetAssetInfo(ctx, result.ID)
				if err != nil {
					return fmt.Errorf("uploaded %s but baseline unavailable; receipt retained as unowned for safety: %w", f.Name, err)
				}
				baseline := values(current)
				m.Baseline = &baseline
				m.Owned = true
				i.state.Media[f.Name] = m
				if err = i.save(); err != nil {
					return err
				}
			}
		}
	}
	return i.Reconcile(ctx)
}

func values(a *immich.Asset) Values {
	return Values{Date: a.ExifInfo.DateTimeOriginal.UTC(), Description: a.ExifInfo.Description}
}

// Reconcile retries pending patches without requiring previous ZIPs.
// Preexisting/unknown-outcome duplicates are never changed automatically.
func (i *Importer) Reconcile(ctx context.Context) error {
	for name, m := range i.state.Media {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !m.Owned {
			continue
		}
		if m.Baseline == nil {
			return fmt.Errorf("missing baseline for managed asset %s", name)
		}
		desired, ok := i.state.Metadata[name]
		if !ok {
			continue
		}
		current, err := i.client.GetAssetInfo(ctx, m.AssetID)
		if err != nil {
			return err
		}
		actual := values(current)
		if m.Applied != nil && m.Applied.equal(desired) {
			if !actual.Date.Equal(desired.Date) || (desired.Description != "" && actual.Description != desired.Description) {
				return fmt.Errorf("asset %s metadata drifted after enrichment for %s; refusing to overwrite", m.AssetID, name)
			}
			continue
		}
		if (!actual.Date.Equal(m.Baseline.Date) && !actual.Date.Equal(desired.Date)) || (desired.Description != "" && actual.Description != m.Baseline.Description && actual.Description != desired.Description) {
			return fmt.Errorf("asset %s changed since baseline; refusing to overwrite metadata for %s", m.AssetID, name)
		}
		_, err = i.client.UpdateAsset(ctx, m.AssetID, immich.UpdAssetField{DateTimeOriginal: desired.Date, Description: desired.Description})
		if err != nil {
			return fmt.Errorf("pending metadata for %s: %w", name, err)
		}
		m.Applied = &desired
		i.state.Media[name] = m
		if err = i.save(); err != nil {
			return err
		}
	}
	return nil
}

func (v Values) equal(other Values) bool {
	return v.Date.Equal(other.Date) && v.Description == other.Description
}

// Summary describes persisted work, including items requiring operator attention.
type Summary struct{ Uploaded, Enriched, Unowned, PendingMedia, PendingMetadata int }

func (i *Importer) Summary() Summary {
	var s Summary
	for _, m := range i.state.Media {
		s.Uploaded++
		if !m.Owned {
			s.Unowned++
			continue
		}
		if m.Applied != nil {
			s.Enriched++
		} else {
			s.PendingMetadata++
		}
	}
	for name := range i.state.Metadata {
		if _, ok := i.state.Media[name]; !ok {
			s.PendingMedia++
		}
	}
	return s
}
