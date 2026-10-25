package incremental

import (
	"archive/zip"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
)

const maxSidecarSize = 4 << 20

func safePath(name string) bool {
	return name != "." && fs.ValidPath(name) && !strings.ContainsAny(name, "\\:") && utf8.ValidString(name) &&
		!strings.ContainsFunc(name, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) })
}

func safeName(name string) bool {
	return name != "" && name != "." && name != ".." && safePath(name) && path.Base(name) == name &&
		strings.TrimSpace(name) == name && !strings.HasSuffix(name, ".") && strings.TrimSuffix(name, path.Ext(name)) != "" && !strings.ContainsAny(name, `:*?"<>|`)
}

func validDate(date time.Time) bool {
	return !date.IsZero() && date.Year() >= 1 && date.Year() <= 9999
}

func isSidecar(name string) bool { return strings.HasSuffix(name, ".json") }

func sidecarTarget(name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, ".json"), ".supplemental-metadata")
}

func (i *Importer) catalogSidecar(z *zip.Reader, f *zip.File, duplicate bool) {
	target := sidecarTarget(f.Name)
	if !safePath(target) {
		i.problem(f.Name, "unsafe or empty sidecar stem; item deferred")
		return
	}
	md := i.state.Metadata[target]
	retain := true
	defer func() {
		if retain {
			i.state.Metadata[target] = md
		}
	}()
	reject := func(reason string) {
		md.Problems = addString(md.Problems, f.Name+": "+reason)
	}
	if duplicate {
		reject("duplicate ZIP sidecar entry; association ambiguous")
		return
	}
	if f.UncompressedSize64 > maxSidecarSize {
		reject("sidecar too large")
		return
	}
	r, err := f.Open()
	if err != nil {
		reject("cannot read sidecar: " + err.Error())
		return
	}
	data, err := io.ReadAll(io.LimitReader(r, maxSidecarSize+1))
	if err = errors.Join(err, r.Close()); err != nil {
		reject("cannot read sidecar: " + err.Error())
		return
	}
	if len(data) > maxSidecarSize {
		reject("sidecar too large")
		return
	}
	var google gp.GoogleMetaData
	if err = json.Unmarshal(data, &google); err != nil {
		reject("invalid sidecar JSON: " + err.Error())
		return
	}
	// Album metadata is outside this PoC. A media stem or a media title is
	// sufficient to retain a filename-only sidecar, even without a capture date.
	if !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(target)) &&
		!filetypes.DefaultSupportedMedia.IsMedia(path.Ext(google.Title)) && google.PhotoTakenTime == nil {
		retain = false
		return
	}
	if !filetypes.DefaultSupportedMedia.IsMedia(path.Ext(target)) {
		reject("unsupported duplicate/truncated sidecar stem; exact media-path association required, item deferred")
		return
	}
	// Use the upstream filename normalization, but never turn a path or control
	// character into an apparently safe authoritative filename.
	if strings.ContainsAny(google.Title, "/\\") || strings.ContainsFunc(google.Title, func(r rune) bool {
		return unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
	}) {
		reject("unsafe sidecar title; item deferred")
		return
	}
	metadata := google.AsMetadata(fshelper.FSName(z, f.Name), false)
	if !safeName(metadata.FileName) || !strings.EqualFold(path.Ext(metadata.FileName), path.Ext(target)) {
		reject("missing/unsafe title or incompatible media extension; exact association unresolved, item deferred")
		return
	}
	md.Names = addString(md.Names, metadata.FileName)
	if google.PhotoTakenTime != nil && google.PhotoTakenTime.Timestamp != "" {
		if _, err := strconv.ParseInt(google.PhotoTakenTime.Timestamp, 10, 64); err != nil || !validDate(metadata.DateTaken) {
			reject("invalid capture timestamp; metadata deferred")
		} else {
			date := metadata.DateTaken.UTC()
			if !slices.ContainsFunc(md.Dates, func(old time.Time) bool { return old.Equal(date) }) {
				md.Dates = append(md.Dates, date)
				slices.SortFunc(md.Dates, func(a, b time.Time) int { return a.Compare(b) })
			}
		}
	}
	if metadata.Description != "" {
		md.Descriptions = addString(md.Descriptions, metadata.Description)
	}
}

func addString(values []string, value string) []string {
	if !slices.Contains(values, value) {
		values = append(values, value)
		slices.Sort(values)
	}
	return values
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// resolve combines metadata from every known occurrence of these bytes.
// Missing fields have no vote; contradictory fields block the logical asset.
func (i *Importer) resolve(checksum string) (string, Values, string) {
	var names, descriptions, reasons []string
	var dates []time.Time
	for _, source := range sortedKeys(i.state.Occurrences) {
		occurrence := i.state.Occurrences[source]
		if occurrence.Checksum != checksum {
			continue
		}
		if len(occurrence.ConflictingChecksums) > 0 {
			reasons = addString(reasons, "ambiguous original path "+source+" has different media")
		}
		for _, problem := range i.state.Problems[source] {
			reasons = addString(reasons, source+": "+problem)
		}
		md := i.state.Metadata[source]
		for _, name := range md.Names {
			names = addString(names, name)
		}
		for _, description := range md.Descriptions {
			descriptions = addString(descriptions, description)
		}
		for _, date := range md.Dates {
			if !slices.ContainsFunc(dates, func(old time.Time) bool { return old.Equal(date) }) {
				dates = append(dates, date)
			}
		}
		for _, problem := range md.Problems {
			reasons = addString(reasons, problem)
		}
	}
	var name string
	var desired Values
	if len(names) == 1 {
		name = names[0]
	} else if len(names) > 1 {
		reasons = addString(reasons, fmt.Sprintf("conflicting canonical filenames %q; originalFileName cannot be renamed", names))
	} else {
		reasons = addString(reasons, "canonical filename unresolved; re-present/re-download media ZIP after sidecar discovery")
	}
	if len(dates) == 1 {
		desired.Date = dates[0]
	} else if len(dates) > 1 {
		reasons = addString(reasons, "conflicting capture dates across source occurrences")
	}
	if len(descriptions) == 1 {
		desired.Description = descriptions[0]
	} else if len(descriptions) > 1 {
		reasons = addString(reasons, fmt.Sprintf("conflicting descriptions across source occurrences %q", descriptions))
	}
	m := i.state.Media[checksum]
	if m.Attempted && name != "" && name != m.CanonicalName {
		reasons = addString(reasons, fmt.Sprintf("canonical filename %q conflicts with upload name %q; originalFileName cannot be renamed", name, m.CanonicalName))
	}
	return name, desired, strings.Join(reasons, "; ")
}

// Issue describes an item requiring attention. These are persisted facts, not
// evidence that every sidecar or every server-side background job was reconciled.
type Issue struct {
	Source   string
	Checksum string
	Reason   string
}

func (i *Importer) Issues() []Issue {
	var issues []Issue
	for _, source := range sortedKeys(i.state.Problems) {
		issues = append(issues, Issue{Source: source, Reason: strings.Join(i.state.Problems[source], "; ")})
	}
	for _, source := range sortedKeys(i.state.Metadata) {
		if _, ok := i.state.Occurrences[source]; !ok {
			md := i.state.Metadata[source]
			reasons := slices.Clone(md.Problems)
			if len(md.Names) > 1 {
				reasons = addString(reasons, fmt.Sprintf("conflicting canonical filenames %q", md.Names))
			}
			if len(md.Dates) > 1 {
				reasons = addString(reasons, "conflicting capture dates")
			}
			if len(md.Descriptions) > 1 {
				reasons = addString(reasons, fmt.Sprintf("conflicting descriptions %q", md.Descriptions))
			}
			reasons = addString(reasons, "sidecar awaiting a known source occurrence; scan/re-present media ZIP")
			issues = append(issues, Issue{Source: source, Reason: strings.Join(reasons, "; ")})
		}
	}
	for _, source := range sortedKeys(i.state.Occurrences) {
		o := i.state.Occurrences[source]
		m := i.state.Media[o.Checksum]
		_, desired, reason := i.resolve(o.Checksum)
		if m.Reason != "" {
			reason = joinReason(reason, m.Reason)
		}
		if m.AssetID == "" {
			reason = joinReason(reason, "media deferred; re-present/re-download media ZIP after metadata discovery")
		} else if !m.Owned {
			reason = joinReason(reason, "unowned logical asset; metadata withheld")
		} else if desired.Date.IsZero() {
			reason = joinReason(reason, "awaiting validated capture date; optional description is retained without sending a zero-date patch")
		}
		if reason != "" {
			issues = append(issues, Issue{Source: source, Checksum: o.Checksum, Reason: reason})
		}
	}
	slices.SortFunc(issues, func(a, b Issue) int {
		if c := cmp.Compare(a.Source, b.Source); c != 0 {
			return c
		}
		return cmp.Compare(a.Reason, b.Reason)
	})
	return issues
}

func joinReason(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}
