package incremental

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/simulot/immich-go/immich"
)

func values(a *immich.Asset) Values {
	return Values{Date: a.ExifInfo.DateTimeOriginal.UTC(), Description: a.ExifInfo.Description}
}

// Reconcile retries patches without media, combining all known source occurrences.
// Preexisting/unknown-outcome duplicates never gain ownership from another path.
func (i *Importer) Reconcile(ctx context.Context) error {
	var failures []error
	for _, checksum := range sortedKeys(i.state.Media) {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		m := i.state.Media[checksum]
		name, desired, reason := i.resolve(checksum)
		if !m.Attempted && reason == "" {
			m.CanonicalName = name
		}
		if !m.Owned || reason != "" {
			i.state.Media[checksum] = m
			continue
		}
		if m.Baseline == nil {
			return fmt.Errorf("missing baseline for managed asset %s", m.AssetID)
		}
		current, err := i.client.GetAssetInfo(ctx, m.AssetID)
		if err != nil || current == nil {
			if err == nil {
				err = errors.New("empty asset response")
			}
			m.Reason = "metadata read pending: " + err.Error()
			failures = append(failures, fmt.Errorf("asset %s: %w", m.AssetID, err))
		} else {
			actual := values(current)
			err = i.reconcileAsset(ctx, &m, desired, actual)
			if err != nil {
				m.Reason = err.Error()
				failures = append(failures, err)
			} else {
				m.Reason = ""
			}
		}
		i.state.Media[checksum] = m
		// Keep each successful patch receipt durable before processing another ID.
		if err := i.save(); err != nil {
			return errors.Join(append(failures, err)...)
		}
	}
	return errors.Join(append(failures, i.save())...)
}

func (i *Importer) reconcileAsset(ctx context.Context, m *Media, desired, actual Values) error {
	// Always check previously applied fields, including when new fields arrive.
	if m.Applied != nil && (!actual.Date.Equal(m.Applied.Date) ||
		(m.Applied.Description != "" && actual.Description != m.Applied.Description)) {
		return fmt.Errorf("asset %s metadata drifted after enrichment; refusing to overwrite", m.AssetID)
	}
	if desired.Date.IsZero() {
		// UpdAssetField serializes even a zero time.Time. Retain description until
		// a validated capture date arrives, rather than corrupting the server date.
		return nil
	}
	if !validDate(desired.Date) {
		return fmt.Errorf("asset %s has invalid capture date; refusing zero-date patch", m.AssetID)
	}
	if (!actual.Date.Equal(m.Baseline.Date) && !actual.Date.Equal(desired.Date)) ||
		(desired.Description != "" && actual.Description != m.Baseline.Description && actual.Description != desired.Description) {
		return fmt.Errorf("asset %s changed since baseline; refusing to overwrite metadata", m.AssetID)
	}
	if m.Applied != nil && m.Applied.equal(desired) {
		return nil
	}
	_, err := i.client.UpdateAsset(ctx, m.AssetID, immich.UpdAssetField{DateTimeOriginal: desired.Date, Description: desired.Description})
	if err != nil {
		return fmt.Errorf("pending metadata for asset %s: %w", m.AssetID, err)
	}
	m.Applied = &desired
	return nil
}

func (v Values) equal(other Values) bool {
	return v.Date.Equal(other.Date) && v.Description == other.Description
}

// Summary counts logical assets separately from paths. Deferred assets have no
// upload receipt; UnownedMetadata explicitly counts metadata withheld for safety.
type Summary struct {
	Uploaded, Enriched, Unowned, PendingMedia, PendingMetadata       int
	LogicalAssets, Occurrences, Deferred, Conflicts, UnownedMetadata int
}

func (i *Importer) Summary() Summary {
	s := Summary{LogicalAssets: len(i.state.Media), Occurrences: len(i.state.Occurrences)}
	for checksum, m := range i.state.Media {
		_, desired, reason := i.resolve(checksum)
		if strings.Contains(reason, "conflict") || strings.Contains(reason, "ambiguous") || strings.Contains(m.Reason, "refusing to overwrite") {
			s.Conflicts++
		}
		if m.Attempted && !m.Owned {
			s.Unowned++
			s.UnownedMetadata++
		}
		if m.AssetID == "" {
			s.Deferred++
			continue
		}
		s.Uploaded++
		if !m.Owned {
			continue
		}
		if m.Applied != nil {
			s.Enriched++
		}
		if reason != "" || m.Reason != "" || desired.Date.IsZero() || m.Applied == nil || !m.Applied.equal(desired) {
			s.PendingMetadata++
		}
	}
	for source, md := range i.state.Metadata {
		if _, ok := i.state.Occurrences[source]; !ok {
			s.PendingMedia++
			if len(md.Names) > 1 || len(md.Dates) > 1 || len(md.Descriptions) > 1 {
				s.Conflicts++
			}
		}
	}
	return s
}
