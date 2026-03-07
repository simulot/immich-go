package folder

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MonthToDateRange tests ---

func TestMonthToDateRange(t *testing.T) {
	tests := []struct {
		name        string
		month       string
		expectAfter time.Time
		expectBefore time.Time
		expectErr   bool
	}{
		{
			name:         "June 2022",
			month:        "2022-06",
			expectAfter:  time.Date(2022, 5, 31, 0, 0, 0, 0, time.UTC),
			expectBefore: time.Date(2022, 7, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "January 2020 (year boundary)",
			month:        "2020-01",
			expectAfter:  time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC),
			expectBefore: time.Date(2020, 2, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "December 2023 (year boundary end)",
			month:        "2023-12",
			expectAfter:  time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
			expectBefore: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "February 2024 (leap year)",
			month:        "2024-02",
			expectAfter:  time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			expectBefore: time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "invalid format",
			month:     "2022",
			expectErr: true,
		},
		{
			name:      "invalid month string",
			month:     "not-a-date",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			after, before, err := adapters.MonthToDateRange(tc.month)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectAfter, after, "after mismatch")
			assert.Equal(t, tc.expectBefore, before, "before mismatch")
		})
	}
}

func TestMonthToDateRange_Padding(t *testing.T) {
	// Verify that a date on the last day of the previous month is included
	after, before, err := adapters.MonthToDateRange("2022-06")
	require.NoError(t, err)

	// May 31 at 23:59 should be in range (>= after)
	may31 := time.Date(2022, 5, 31, 23, 59, 59, 0, time.UTC)
	assert.True(t, (may31.Equal(after) || may31.After(after)) && may31.Before(before),
		"May 31 23:59 should be in padded range")

	// July 1 at 23:59 should be in range (< before which is July 2)
	jul1 := time.Date(2022, 7, 1, 23, 59, 59, 0, time.UTC)
	assert.True(t, (jul1.Equal(after) || jul1.After(after)) && jul1.Before(before),
		"July 1 23:59 should be in padded range")

	// July 2 at 00:00 should NOT be in range (== before, exclusive)
	jul2 := time.Date(2022, 7, 2, 0, 0, 0, 0, time.UTC)
	assert.False(t, jul2.Before(before),
		"July 2 00:00 should NOT be in padded range")
}

// --- BrowseMonth tests ---

// newTestIFC creates a minimal ImportFolderCmd suitable for testing BrowseMonth.
// It creates temp files with date-bearing filenames in the given directory.
func newTestIFC(t *testing.T, dir string) *ImportFolderCmd {
	t.Helper()
	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	logger := slog.Default()
	recorder := fileevent.NewRecorder(logger)
	tracker := assettracker.New()
	proc := fileprocessor.New(tracker, recorder)

	ifc := &ImportFolderCmd{
		Recursive:      true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		activeMonthSet: make(map[string]struct{}),
		pool:           worker.NewPool(1),
		groupers:       []groups.Grouper{series.Group},
		processor:      proc,
	}
	return ifc
}

func TestBrowseMonth_FiltersCorrectMonth(t *testing.T) {
	dir := t.TempDir()

	// Create files with dates in different months and set their mod times
	// to match the date in their name, since assetDate checks FileDate before Taken.
	type testFile struct {
		name    string
		modTime time.Time
	}
	// Note: MonthToDateRange uses ±1 day padding for timezone safety.
	// For "2022-06", the range is [2022-05-31, 2022-07-02).
	// So 2022-07-01 is still within the padded range.
	// We use 2022-07-10 and 2023-01-15 as clearly-outside dates.
	files := []testFile{
		{"IMG_20220601_120000.jpg", time.Date(2022, 6, 1, 12, 0, 0, 0, time.UTC)},
		{"IMG_20220615_120000.jpg", time.Date(2022, 6, 15, 12, 0, 0, 0, time.UTC)},
		{"IMG_20220710_120000.jpg", time.Date(2022, 7, 10, 12, 0, 0, 0, time.UTC)},
		{"IMG_20230115_120000.jpg", time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)},
	}
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		err := os.WriteFile(p, []byte("fake"), 0o644)
		require.NoError(t, err)
		err = os.Chtimes(p, f.modTime, f.modTime)
		require.NoError(t, err)
	}

	ifc := newTestIFC(t, dir)

	ctx := context.Background()
	ch := ifc.BrowseMonth(ctx, "2022-06")

	var collected []string
	for g := range ch {
		for _, a := range g.Assets {
			collected = append(collected, a.OriginalFileName)
		}
	}

	assert.Contains(t, collected, "IMG_20220601_120000.jpg")
	assert.Contains(t, collected, "IMG_20220615_120000.jpg")
	assert.NotContains(t, collected, "IMG_20220710_120000.jpg")
	assert.NotContains(t, collected, "IMG_20230115_120000.jpg")
}

func TestBrowseMonth_NoDateIncluded(t *testing.T) {
	dir := t.TempDir()

	// Create a file with no date in the name and a mod time in August 2023
	randomPath := filepath.Join(dir, "random_photo.jpg")
	err := os.WriteFile(randomPath, []byte("fake"), 0o644)
	require.NoError(t, err)
	err = os.Chtimes(randomPath, time.Date(2023, 8, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 8, 10, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	// Also create a date-bearing file with matching mod time
	junePath := filepath.Join(dir, "IMG_20220601_120000.jpg")
	err = os.WriteFile(junePath, []byte("fake"), 0o644)
	require.NoError(t, err)
	err = os.Chtimes(junePath, time.Date(2022, 6, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2022, 6, 1, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	ifc := newTestIFC(t, dir)

	ctx := context.Background()

	// BrowseMonth("2022-06") should only yield the June file, not the random file
	// (random_photo.jpg has FileDate of 2023-08-10, outside June 2022 range)
	ch := ifc.BrowseMonth(ctx, "2022-06")

	var collected []string
	for g := range ch {
		for _, a := range g.Assets {
			collected = append(collected, a.OriginalFileName)
		}
	}

	assert.Contains(t, collected, "IMG_20220601_120000.jpg")
	assert.NotContains(t, collected, "random_photo.jpg",
		"random_photo.jpg has Aug 2023 FileDate, should not appear in 2022-06")
}

func TestBrowseMonth_NoDateBucket(t *testing.T) {
	// Test that BrowseMonth with "no-date" works for the assetDate helper.
	// Since we can't easily make a file with zero FileDate through the filesystem,
	// we test the assetDate function directly.

	// Asset with no dates at all
	a := &assets.Asset{}
	d := assetDate(a)
	assert.True(t, d.IsZero(), "asset with no dates should have zero date")

	// Asset with CaptureDate
	a2 := &assets.Asset{CaptureDate: time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC)}
	d2 := assetDate(a2)
	assert.Equal(t, time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC), d2)

	// Asset with only FileDate
	a3 := &assets.Asset{FileDate: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)}
	d3 := assetDate(a3)
	assert.Equal(t, time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC), d3)

	// Asset with only NameInfo.Taken
	a4 := &assets.Asset{}
	a4.NameInfo.Taken = time.Date(2021, 3, 10, 0, 0, 0, 0, time.UTC)
	d4 := assetDate(a4)
	assert.Equal(t, time.Date(2021, 3, 10, 0, 0, 0, 0, time.UTC), d4)
}

func TestAssetDate_ResolutionOrder(t *testing.T) {
	// CaptureDate takes precedence over FileDate and Taken
	a := &assets.Asset{
		CaptureDate: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		FileDate:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	a.NameInfo.Taken = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), assetDate(a))

	// FileDate takes precedence over Taken when CaptureDate is zero
	a2 := &assets.Asset{
		FileDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	a2.NameInfo.Taken = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), assetDate(a2))

	// Taken is used when CaptureDate and FileDate are both zero
	a3 := &assets.Asset{}
	a3.NameInfo.Taken = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), assetDate(a3))
}

func TestBrowse_StillWorks(t *testing.T) {
	// Backward compatibility: Browse() should still return all assets
	dir := t.TempDir()

	type testFile struct {
		name    string
		modTime time.Time
	}
	files := []testFile{
		{"IMG_20220601_120000.jpg", time.Date(2022, 6, 1, 12, 0, 0, 0, time.UTC)},
		{"IMG_20220701_120000.jpg", time.Date(2022, 7, 1, 12, 0, 0, 0, time.UTC)},
		{"IMG_20230115_120000.jpg", time.Date(2023, 1, 15, 12, 0, 0, 0, time.UTC)},
	}
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		err := os.WriteFile(p, []byte("fake"), 0o644)
		require.NoError(t, err)
		err = os.Chtimes(p, f.modTime, f.modTime)
		require.NoError(t, err)
	}

	ifc := newTestIFC(t, dir)

	ctx := context.Background()
	ch := ifc.Browse(ctx)

	var collected []string
	for g := range ch {
		for _, a := range g.Assets {
			collected = append(collected, a.OriginalFileName)
		}
	}

	// All files should be present
	assert.Len(t, collected, 3)
	assert.Contains(t, collected, "IMG_20220601_120000.jpg")
	assert.Contains(t, collected, "IMG_20220701_120000.jpg")
	assert.Contains(t, collected, "IMG_20230115_120000.jpg")
}

// Verify that ImportFolderCmd implements DateRangeProvider at compile time.
var _ adapters.DateRangeProvider = (*ImportFolderCmd)(nil)
