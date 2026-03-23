package synology

import (
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
)

// newTestAdapter returns a minimal Adapter suitable for unit tests that do not
// require a network connection or an app.Application instance.
func newTestAdapter() *Adapter {
	return &Adapter{
		albumCache:      make(map[string]Album),
		tagCache:        make(map[string]int),
		peopleCache:     make(map[string]int),
		albumImageCache: make(map[string]mapset.Set[int]),
	}
}

// ---------------------------------------------------------------------------
// matchesAlbumFilter
// ---------------------------------------------------------------------------

func TestMatchesAlbumFilter_NoAlbumsRequested(t *testing.T) {
	sa := newTestAdapter()
	// Albums slice is empty → no filter, every item passes
	item := &Item{ID: 1}
	if !sa.matchesAlbumFilter(item) {
		t.Error("expected item to pass when no albums are requested")
	}
}

func TestMatchesAlbumFilter_ItemInRequestedAlbum(t *testing.T) {
	sa := newTestAdapter()
	sa.Albums = []string{"Vacation"}
	sa.albumImageCache["Vacation"] = mapset.NewSet(1, 2, 3)

	item := &Item{ID: 2}
	if !sa.matchesAlbumFilter(item) {
		t.Error("expected item to pass when it belongs to a requested album")
	}
}

func TestMatchesAlbumFilter_ItemNotInRequestedAlbum(t *testing.T) {
	sa := newTestAdapter()
	sa.Albums = []string{"Vacation"}
	sa.albumImageCache["Vacation"] = mapset.NewSet(1, 2, 3)

	item := &Item{ID: 99}
	if sa.matchesAlbumFilter(item) {
		t.Error("expected item to be filtered out when it does not belong to any requested album")
	}
}

func TestMatchesAlbumFilter_ItemInOneOfMultipleAlbums(t *testing.T) {
	sa := newTestAdapter()
	sa.Albums = []string{"Vacation", "Family"}
	sa.albumImageCache["Vacation"] = mapset.NewSet(1, 2)
	sa.albumImageCache["Family"] = mapset.NewSet(3, 4)

	item := &Item{ID: 3}
	if !sa.matchesAlbumFilter(item) {
		t.Error("expected item to pass when it belongs to at least one requested album")
	}
}

// ---------------------------------------------------------------------------
// correctTimestamp
// ---------------------------------------------------------------------------

func TestCorrectTimestamp_UTC(t *testing.T) {
	sa := newTestAdapter()
	// In UTC there is no offset, so the timestamp should be unchanged.
	now := time.Now().Unix()
	got := sa.correctTimestamp(now, time.UTC)
	if got != now {
		t.Errorf("UTC: expected %d, got %d", now, got)
	}
}

func TestCorrectTimestamp_FixedOffset(t *testing.T) {
	sa := newTestAdapter()
	// Use a fixed +8h timezone (Asia/Shanghai-like).
	// Synology stores local time as a Unix timestamp, so the "wrong" value is
	// actually local-time seconds since epoch. correctTimestamp should subtract
	// the UTC offset to obtain the real UTC timestamp.
	loc := time.FixedZone("UTC+8", 8*60*60)

	// Pick a reference time: 2024-01-15 12:00:00 UTC
	realUTC := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).Unix()
	// Synology would store 2024-01-15 20:00:00 as if it were a UTC timestamp
	synologyWrong := realUTC + int64(8*60*60)

	got := sa.correctTimestamp(synologyWrong, loc)
	if got != realUTC {
		t.Errorf("UTC+8: expected %d (%s), got %d (%s)",
			realUTC, time.Unix(realUTC, 0).UTC(),
			got, time.Unix(got, 0).UTC())
	}
}
