package incremental

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/simulot/immich-go/internal/filetypes"
)

// ArchivePlan binds an ordered, complete filename inventory to this export state.
// Names are case sensitive and contain no source paths or expiring download URLs.
type ArchivePlan struct {
	Expected []string
	Archives map[string]ArchiveRecord
	Current  string `json:",omitempty"`
}

type ArchiveRecord struct {
	Digest  string // SHA-256 of the entire compressed ZIP, not its entry names.
	Scanned bool
	// A later unreadable local copy can hold progress, but cannot reopen the
	// changed-byte repair permission of a previously completed inventory.
	IdentitySealed bool              `json:",omitempty"`
	Media          []string          // Actual supported-media checksums read from this ZIP.
	Sources        map[string]string `json:",omitempty"` // Exact ZIP paths to those checksums.
	// Non-nil, even when empty: older failed records without sidecar provenance
	// cannot authorize a changed-byte repair. Digests cover readable raw JSON.
	Sidecars     map[string]string
	CatalogError string
}

// ArchiveAction is guidance, never permission to delete a ZIP. Action is one of
// inventory, keep/retry, reconcile, download, replay, blocked, attention, complete.
// A download/replay can use an existing local file; no downloader is provided.
type ArchiveAction struct {
	Action  string
	Archive string
	Reason  string
}

const archiveActionBlocked = "blocked"

func archiveName(name string) bool {
	return safeName(name) && strings.EqualFold(path.Ext(name), ".zip")
}

func validateArchiveNames(names []string) error {
	seen := map[string]bool{}
	for _, name := range names {
		if !archiveName(name) {
			return fmt.Errorf("archive inventory requires plain ZIP basenames, no URLs or paths: %q", name)
		}
		if seen[name] {
			return fmt.Errorf("duplicate archive inventory filename %q", name)
		}
		seen[name] = true
	}
	return nil
}

// validateArchives is also called by every planning API: a malformed optional
// persisted plan must never be mistaken for a completed export.
func (i *Importer) validateArchives() error {
	p := i.state.Archives
	if p == nil {
		return nil
	}
	if err := validateArchiveNames(p.Expected); err != nil {
		return err
	}
	if p.Archives == nil {
		return errors.New("invalid archive plan: missing archive records")
	}
	digests := map[string]string{}
	for _, name := range sortedKeys(p.Archives) {
		r := p.Archives[name]
		if !archiveName(name) || (len(p.Expected) > 0 && !slices.Contains(p.Expected, name)) {
			return fmt.Errorf("invalid archive record name %q", name)
		}
		if r.Digest == "" {
			if r.Scanned || r.IdentitySealed || r.CatalogError == "" || len(r.Media) > 0 || len(r.Sources) > 0 || len(r.Sidecars) > 0 {
				return fmt.Errorf("invalid archive identity for %q", name)
			}
		} else {
			digest, err := hex.DecodeString(r.Digest)
			if err != nil || len(digest) != sha256.Size || hex.EncodeToString(digest) != r.Digest {
				return fmt.Errorf("invalid archive digest for %q", name)
			}
			if other := digests[r.Digest]; other != "" {
				return fmt.Errorf("archive %q repeats the compressed identity of %q; use the original filename", name, other)
			}
			digests[r.Digest] = name
		}
		if r.Scanned != (r.CatalogError == "") {
			return fmt.Errorf("invalid archive scan status for %q", name)
		}
		checksums := map[string]bool{}
		for _, checksum := range r.Media {
			if _, err := payloadName(checksum); err != nil || checksums[checksum] {
				return fmt.Errorf("invalid archive media checksum for %q", name)
			}
			checksums[checksum] = true
		}
		found := map[string]bool{}
		for source, checksum := range r.Sources {
			if !safePath(source) || !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(source)) || !checksums[checksum] {
				return fmt.Errorf("invalid archive source inventory for %q", name)
			}
			found[checksum] = true
			if r.Scanned {
				m, ok := i.state.Media[checksum]
				if !ok || m.Checksum != checksum || i.state.Occurrences[source].Checksum != checksum {
					return fmt.Errorf("archive %q has no matching catalog receipt for %q", name, source)
				}
			}
		}
		if len(found) != len(checksums) {
			return fmt.Errorf("archive %q has media without source provenance", name)
		}
		for source, digest := range r.Sidecars {
			decoded, err := hex.DecodeString(digest)
			if !safePath(source) || !isSidecar(source) || !safePath(sidecarTarget(source)) ||
				err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != digest {
				return fmt.Errorf("invalid archive sidecar provenance for %q", name)
			}
		}
	}
	if p.Current != "" {
		if _, ok := p.Archives[p.Current]; !ok {
			return errors.New("invalid archive plan: current archive has no record")
		}
	}
	return nil
}

// ConfigureArchives persists a full ordered manifest atomically. Once supplied,
// neither its order nor its members may change. A first manifest added after
// imports must include every previously recorded archive name.
func (i *Importer) ConfigureArchives(names []string) error {
	if err := i.validateArchives(); err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("archive inventory must list every ZIP filename in the export")
	}
	if err := validateArchiveNames(names); err != nil {
		return err
	}
	previous := i.state.Archives
	p := &ArchivePlan{Expected: slices.Clone(names), Archives: map[string]ArchiveRecord{}}
	if previous != nil {
		if len(previous.Expected) > 0 && !slices.Equal(previous.Expected, names) {
			return errors.New("archive inventory differs from the persisted export plan; preserve it and use a separate state/export ID for another export")
		}
		for name := range previous.Archives {
			if !slices.Contains(names, name) {
				return fmt.Errorf("archive inventory omits already recorded archive %q", name)
			}
		}
		p.Archives, p.Current = previous.Archives, previous.Current
	}
	i.state.Archives = p
	if err := i.save(); err != nil {
		i.state.Archives = previous
		return err
	}
	return nil
}

func archiveDigest(ctx context.Context, filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, err = io.CopyBuffer(h, contextReader{ctx, f}, make([]byte, 64<<10))
	if err := errors.Join(err, f.Close()); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// inventoryArchive deliberately rereads ZIP media in this PoC. It records the
// bytes actually present, including a checksum rejected by an existing path
// conflict, rather than borrowing another archive's occurrence receipt.
func inventoryArchive(ctx context.Context, filename string) (ArchiveRecord, error) {
	r := ArchiveRecord{Sources: map[string]string{}, Sidecars: map[string]string{}}
	z, err := zip.OpenReader(filename)
	if err != nil {
		return r, err
	}
	defer z.Close()
	counts := map[string]int{}
	for _, f := range z.File {
		if !f.FileInfo().IsDir() {
			counts[f.Name]++
		}
	}
	var failures []error
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return r, errors.Join(append(failures, err)...)
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !safePath(f.Name) {
			failures = append(failures, fmt.Errorf("unsafe archive path %q", f.Name))
			continue
		}
		if counts[f.Name] > 1 {
			failures = append(failures, fmt.Errorf("duplicate ZIP entry path %q; source association ambiguous", f.Name))
			continue
		}
		if filetypes.DefaultSupportedMedia.IsMedia(path.Ext(f.Name)) {
			checksum, err := checksumZIP(ctx, f)
			if err != nil {
				failures = append(failures, fmt.Errorf("cannot inventory media %q: %w", f.Name, err))
				continue
			}
			r.Media = addString(r.Media, checksum)
			r.Sources[f.Name] = checksum
		} else if isSidecar(f.Name) {
			if !safePath(sidecarTarget(f.Name)) || f.UncompressedSize64 > maxSidecarSize {
				failures = append(failures, fmt.Errorf("cannot inventory sidecar %q: unsafe stem or sidecar too large", f.Name))
				continue
			}
			reader, err := f.Open()
			h := sha256.New()
			if err == nil {
				_, err = io.CopyBuffer(h, contextReader{ctx, reader}, make([]byte, 64<<10))
				err = errors.Join(err, reader.Close())
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("cannot inventory sidecar %q: %w", f.Name, err))
			} else {
				r.Sidecars[f.Name] = hex.EncodeToString(h.Sum(nil))
			}
		}
	}
	return r, errors.Join(failures...)
}

// A failed ZIP can be replaced by a healthy download without discarding its
// healthy progress. Previously readable sources and raw sidecars are immutable;
// an unreadable entry had no authoritative bytes and can now be cataloged.
func compatibleArchiveRepair(name string, previous, replacement ArchiveRecord) error {
	if previous.Sidecars == nil {
		return fmt.Errorf("cannot verify repaired archive %q: prior failed record lacks readable sidecar provenance; retry the original source to recover its inventory first", name)
	}
	for _, source := range sortedKeys(previous.Sources) {
		if replacement.Sources[source] != previous.Sources[source] {
			return fmt.Errorf("repaired archive %q changed or omitted previously readable media %q; restore its original healthy bytes", name, source)
		}
	}
	for _, source := range sortedKeys(previous.Sidecars) {
		if replacement.Sidecars[source] != previous.Sidecars[source] {
			return fmt.Errorf("repaired archive %q changed or omitted previously readable sidecar %q; restore its original healthy metadata", name, source)
		}
	}
	return nil
}

// ImportArchive adds stable filename provenance around ImportZIP. Identity and
// manifest failures precede any ImportZIP mutation. Catalog errors still allow
// healthy imports, but hold the exact source ZIP for investigation/local retry.
// HTTP failures alone do not invalidate a complete inventory or require a ZIP
// when its bytes have been retained or an upload receipt is already durable.
func (i *Importer) ImportArchive(ctx context.Context, filename string) error {
	if err := i.validateArchives(); err != nil {
		return err
	}
	name := filepath.Base(filename)
	if !archiveName(name) {
		return fmt.Errorf("archive filename must be a plain ZIP basename: %q", name)
	}
	p := i.state.Archives
	if p == nil {
		p = &ArchivePlan{Archives: map[string]ArchiveRecord{}}
	}
	if len(p.Expected) > 0 && !slices.Contains(p.Expected, name) {
		return fmt.Errorf("archive %q is not listed in this export's archive inventory; import refused", name)
	}
	digest, digestErr := archiveDigest(ctx, filename)
	previous := p.Archives[name]
	repair := false
	if digestErr == nil {
		if previous.Digest != "" && previous.Digest != digest {
			if previous.Scanned || previous.IdentitySealed {
				return fmt.Errorf("archive %q has changed compressed bytes after a completed catalog; preserve the original ZIP and use a separate state/export ID for another export", name)
			}
			repair = true
		}
		for other, record := range p.Archives {
			if other != name && record.Digest == digest {
				return fmt.Errorf("archive %q has the same compressed bytes as %q; use that original filename rather than renaming a source", name, other)
			}
		}
	}
	if digestErr != nil {
		i.state.Archives, p.Current = p, name
		previous.IdentitySealed = previous.IdentitySealed || previous.Scanned
		previous.Scanned = false
		previous.CatalogError = "cannot read archive identity: " + digestErr.Error()
		p.Archives[name] = previous
		return errors.Join(digestErr, i.save())
	}
	r, inventoryErr := inventoryArchive(ctx, filename)
	r.Digest = digest
	r.IdentitySealed = previous.IdentitySealed || previous.Scanned
	if repair {
		if inventoryErr != nil {
			return fmt.Errorf("replacement archive %q is still unreadable/unsafe; original held record preserved: %w", name, inventoryErr)
		}
		if err := compatibleArchiveRepair(name, previous, r); err != nil {
			return err
		}
		for _, source := range sortedKeys(r.Sources) {
			if old, ok := i.state.Occurrences[source]; ok && old.Checksum != r.Sources[source] {
				return fmt.Errorf("repaired archive %q conflicts with cataloged media at %q; original held record preserved", name, source)
			}
		}
	} else if previous.Digest == digest {
		// A transient failure on another read of the same compressed bytes must
		// not forget healthy provenance established by an earlier partial scan.
		for source, checksum := range previous.Sources {
			if _, ok := r.Sources[source]; !ok {
				r.Sources[source] = checksum
				r.Media = addString(r.Media, checksum)
			}
		}
		for source, checksum := range previous.Sidecars {
			if _, ok := r.Sidecars[source]; !ok {
				r.Sidecars[source] = checksum
			}
		}
		if previous.Sidecars == nil && inventoryErr != nil {
			r.Sidecars = nil
		}
	}
	i.state.Archives, p.Current = p, name
	r.CatalogError = "catalog import has not finished; keep this ZIP for local retry"
	if !repair {
		p.Archives[name] = r
	}
	// An interrupted import must leave a durable current-file hold, never a
	// complete record published before ImportZIP has cataloged its contents.
	if err := i.save(); err != nil {
		return err
	}
	importErr := i.ImportZIP(ctx, filename)
	var catalogErrors []error
	catalogErrors = append(catalogErrors, inventoryErr, ctx.Err())
	for _, source := range sortedKeys(r.Sources) {
		checksum := r.Sources[source]
		m, ok := i.state.Media[checksum]
		if !ok || m.Checksum != checksum || i.state.Occurrences[source].Checksum != checksum {
			catalogErrors = append(catalogErrors, fmt.Errorf("source %q could not be cataloged with its actual checksum; different media at an original path must use a separate export state", source))
		}
	}
	// ImportZIP opens the file itself. Check again before publishing completion
	// so a replaced or damaged source cannot silently change the saved identity.
	after, err := archiveDigest(ctx, filename)
	if err != nil {
		catalogErrors = append(catalogErrors, err)
	} else if after != digest {
		catalogErrors = append(catalogErrors, errors.New("archive bytes changed during import; preserve the source and investigate before retrying"))
	}
	catalogErr := errors.Join(catalogErrors...)
	if repair && catalogErr != nil {
		// The original identity remains the repair anchor until the replacement
		// is fully cataloged. ImportZIP's healthy receipts remain durable.
		return errors.Join(importErr, catalogErr, i.save())
	}
	r.Scanned, r.CatalogError = catalogErr == nil, ""
	r.IdentitySealed = r.IdentitySealed || r.Scanned
	if catalogErr != nil {
		r.CatalogError = catalogErr.Error()
	}
	incomplete := p.Archives[name]
	p.Archives[name] = r
	if err := i.save(); err != nil {
		p.Archives[name] = incomplete
		return errors.Join(importErr, catalogErr, err)
	}
	return errors.Join(importErr, catalogErr)
}

// A persisted Staged reference is not proof that its bytes still exist. Verify
// pending payloads before promising a ZIP-free retry, without modifying cache,
// receipts or ownership. This deliberately adds a full cache read in this PoC.
func (i *Importer) cachedArchiveAction(order []string) ArchiveAction {
	failures := map[string]error{}
	ready := false
	for _, checksum := range sortedKeys(i.state.Media) {
		m := i.state.Media[checksum]
		if m.AssetID != "" || !m.Staged {
			continue // A durable upload receipt needs no media bytes for cleanup.
		}
		err := errors.New("staging directory is not open; restore its bound location and reopen the importer")
		if i.staging != nil {
			err = i.verifyStaged(context.Background(), m)
		}
		if err != nil {
			failures[checksum] = err
		} else if _, _, reason := i.resolve(checksum); reason == "" {
			ready = true
		}
	}
	if len(failures) > 0 {
		if p := i.state.Archives; p != nil {
			for _, name := range order {
				for _, checksum := range p.Archives[name].Media {
					if err := failures[checksum]; err != nil {
						return ArchiveAction{Action: "keep/retry", Archive: name, Reason: "Cached media cannot be read or verified. Restore its original payload in the bound cache, or re-present this exact source ZIP (use the local file or re-download if necessary) to retry the original bytes. Preserve the existing cache and upload receipts while investigating: " + err.Error()}
					}
				}
			}
		}
		checksum := sortedKeys(failures)[0]
		return ArchiveAction{Action: archiveActionBlocked, Reason: fmt.Sprintf("Cached media %q cannot be read or verified, and no source ZIP filename was recorded. Restore its original payload in the bound cache or re-present the original media ZIP; the filename cannot be inferred. Preserve the cache and upload receipts while investigating: %s", checksum, failures[checksum])}
	}
	if ready {
		return ArchiveAction{Action: "reconcile", Reason: "Run reconcile to retry ready, verified cached uploads; no ZIP download is needed for those bytes."}
	}
	return ArchiveAction{}
}

// NextArchive chooses catalog discovery before replaying ordinary uncached
// bytes. Damaged cache references need explicit repair, not an endless retry.
// An unseen ZIP may contain needed metadata, but its contents are never guessed.
func (i *Importer) NextArchive() ArchiveAction {
	if err := i.validateArchives(); err != nil {
		return ArchiveAction{Action: archiveActionBlocked, Reason: "Invalid persisted archive plan; preserve state and resolve it before continuing: " + err.Error()}
	}
	p := i.state.Archives
	var order []string
	if p != nil {
		order = p.Expected
		if len(order) == 0 {
			order = sortedKeys(p.Archives)
		}
		holds := append([]string{p.Current}, order...)
		for _, name := range holds {
			if r, ok := p.Archives[name]; ok && !r.Scanned {
				reason := "Keep this ZIP and retry it locally after investigating the catalog failure. A repaired replacement with the same filename must preserve all previously readable media and sidecars; download availability is unknown. "
				if r.IdentitySealed {
					reason = "Restore this previously cataloged ZIP with its exact original compressed bytes and retry it locally; its completed identity is immutable and download availability is unknown. "
				}
				return ArchiveAction{Action: "keep/retry", Archive: name, Reason: reason + r.CatalogError}
			}
		}
	}
	if action := i.cachedArchiveAction(order); action.Action != "" {
		return action
	}
	if p == nil || len(p.Expected) == 0 {
		return ArchiveAction{Action: "inventory", Reason: "Supply --archive-list with the complete ordered ZIP basenames for this export. Without that inventory, the next unseen filename and export completion are unknown; do not guess a sequence."}
	}
	for _, name := range p.Expected {
		if !p.Archives[name].Scanned {
			return ArchiveAction{Action: "download", Archive: name, Reason: "Scan this next listed archive; use the local ZIP if present, otherwise download it. All listed archives must be cataloged before planning an uncached replay."}
		}
	}
	for _, name := range p.Expected {
		for _, checksum := range p.Archives[name].Media {
			m := i.state.Media[checksum]
			if _, _, reason := i.resolve(checksum); m.AssetID == "" && !m.Staged && reason == "" {
				return ArchiveAction{Action: "replay", Archive: name, Reason: "Re-present this ZIP to upload resolved media whose bytes were not retained; use the local file or re-download it once if necessary. Existing upload receipts are skipped."}
			}
		}
	}
	var blockers, attention []string
	pendingMetadata := false
	for _, checksum := range sortedKeys(i.state.Media) {
		m := i.state.Media[checksum]
		_, desired, reason := i.resolve(checksum)
		if m.Staged && m.AssetID != "" {
			pendingMetadata = true // Receipt cleanup also applies to unowned uploads.
		}
		switch {
		case m.AssetID == "" || reason != "":
			blockers = append(blockers, checksum+": "+joinReason(reason, "no further archive can be selected from the completed inventory; inspect Issues()"))
		case !m.Owned:
			attention = append(attention, checksum+": unowned upload receipt; metadata remains withheld")
		case strings.Contains(m.Reason, "refusing to overwrite"):
			blockers = append(blockers, checksum+": "+m.Reason)
		case desired.Date.IsZero():
			blockers = append(blockers, checksum+": validated capture date is still missing")
		case m.Staged || m.Reason != "" || m.Applied == nil || !m.Applied.equal(desired):
			pendingMetadata = true
		}
	}
	for _, source := range sortedKeys(i.state.Problems) {
		if len(i.state.Problems[source]) > 0 {
			blockers = append(blockers, source+": "+strings.Join(i.state.Problems[source], "; "))
		}
	}
	for _, source := range sortedKeys(i.state.Metadata) {
		if _, ok := i.state.Occurrences[source]; !ok {
			blockers = append(blockers, source+": sidecar has no known media occurrence")
		}
	}
	if pendingMetadata {
		return ArchiveAction{Action: "reconcile", Reason: "Run reconcile to retry pending metadata/receipt cleanup without downloading a ZIP; inspect Issues() for any remaining conflicts or unowned metadata."}
	}
	if len(blockers) > 0 {
		return ArchiveAction{Action: archiveActionBlocked, Reason: "All listed archives were cataloged, but unresolved items require attention rather than repeated downloads: " + strings.Join(blockers, "; ")}
	}
	if len(attention) > 0 {
		return ArchiveAction{Action: "attention", Reason: "All listed archives were cataloged; metadata is still withheld and cannot be repaired by downloading the ZIP again: " + strings.Join(attention, "; ")}
	}
	return ArchiveAction{Action: "complete", Reason: "All listed archives were cataloged and known media/metadata receipts are reconciled. This is not permission to delete source ZIPs or proof of server-side processing completion."}
}
