package folder

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddMonth(t *testing.T) {
	tests := []struct {
		name           string
		dates          []time.Time
		expectedMonths []string
	}{
		{
			name:           "single date",
			dates:          []time.Time{time.Date(2022, 6, 4, 12, 0, 0, 0, time.UTC)},
			expectedMonths: []string{"2022-06"},
		},
		{
			name: "multiple dates same month are deduplicated",
			dates: []time.Time{
				time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2022, 6, 30, 0, 0, 0, 0, time.UTC),
			},
			expectedMonths: []string{"2022-06"},
		},
		{
			name: "multiple months sorted",
			dates: []time.Time{
				time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2022, 6, 4, 0, 0, 0, 0, time.UTC),
			},
			expectedMonths: []string{"2020-01", "2022-06", "2023-12"},
		},
		{
			name:           "zero time is ignored",
			dates:          []time.Time{time.Time{}, time.Date(2022, 6, 4, 0, 0, 0, 0, time.UTC)},
			expectedMonths: []string{"2022-06"},
		},
		{
			name:           "all zero times produce empty list",
			dates:          []time.Time{time.Time{}, time.Time{}},
			expectedMonths: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ifc := &ImportFolderCmd{
				activeMonthSet: make(map[string]struct{}),
			}
			for _, d := range tc.dates {
				ifc.addMonth(d)
			}
			ifc.buildSortedMonths()
			assert.Equal(t, tc.expectedMonths, ifc.ActiveMonths())
		})
	}
}

func TestAddMonth_Concurrent(t *testing.T) {
	ifc := &ImportFolderCmd{
		activeMonthSet: make(map[string]struct{}),
	}

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func(month int) {
			defer func() { done <- struct{}{} }()
			d := time.Date(2020, time.Month((month%12)+1), 1, 0, 0, 0, 0, time.UTC)
			ifc.addMonth(d)
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	ifc.buildSortedMonths()
	months := ifc.ActiveMonths()
	// Should have exactly 12 unique months
	assert.Len(t, months, 12)
	assert.Equal(t, "2020-01", months[0])
	assert.Equal(t, "2020-12", months[11])
}

func TestICloudPhotoDetailsCollectsMonths(t *testing.T) {
	// Create a temporary directory with a Photo Details.csv file
	dir := t.TempDir()
	csvContent := `imgName,fileChecksum,favorite,hidden,deleted,originalCreationDate,viewCount,importDate
IMG_001.HEIC,abc123,no,no,no,"Saturday June 4,2022 12:11 PM GMT",10,"Saturday June 4,2022 12:11 PM GMT"
IMG_002.HEIC,def456,no,no,no,"Monday January 15,2024 09:30 AM GMT",5,"Monday January 15,2024 09:30 AM GMT"
IMG_003.HEIC,ghi789,no,no,no,"Saturday June 18,2022 14:00 PM GMT",3,"Saturday June 18,2022 14:00 PM GMT"
`
	csvPath := filepath.Join(dir, "Photo Details.csv")
	err := os.WriteFile(csvPath, []byte(csvContent), 0o644)
	require.NoError(t, err)

	m := gen.NewSyncMap[string, iCloudMeta]()
	fsys := os.DirFS(dir)

	// Track months via DateCollector
	ifc := &ImportFolderCmd{
		activeMonthSet: make(map[string]struct{}),
	}

	err = UseICloudPhotoDetails(m, fsys, "Photo Details.csv", ifc.addMonth)
	require.NoError(t, err)

	ifc.buildSortedMonths()
	months := ifc.ActiveMonths()

	// Should have 2 unique months: 2022-06 and 2024-01
	assert.Equal(t, []string{"2022-06", "2024-01"}, months)

	// Verify the metas were also stored correctly
	meta, ok := m.Load("IMG_001.HEIC")
	assert.True(t, ok)
	assert.Equal(t, 2022, meta.originalCreationDate.Year())
	assert.Equal(t, time.June, meta.originalCreationDate.Month())

	meta, ok = m.Load("IMG_002.HEIC")
	assert.True(t, ok)
	assert.Equal(t, 2024, meta.originalCreationDate.Year())
	assert.Equal(t, time.January, meta.originalCreationDate.Month())
}

func TestICloudPhotoDetailsEmptyCSV(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "Photo Details.csv")
	err := os.WriteFile(csvPath, []byte(""), 0o644)
	require.NoError(t, err)

	m := gen.NewSyncMap[string, iCloudMeta]()
	fsys := os.DirFS(dir)

	ifc := &ImportFolderCmd{
		activeMonthSet: make(map[string]struct{}),
	}

	err = UseICloudPhotoDetails(m, fsys, "Photo Details.csv", ifc.addMonth)
	require.NoError(t, err)

	ifc.buildSortedMonths()
	assert.Empty(t, ifc.ActiveMonths())
}

func TestPreScanWithFilenames(t *testing.T) {
	// Create a temp directory structure with files that have date-bearing names
	dir := t.TempDir()

	// Create files with date patterns in their names
	files := map[string]string{
		"IMG_20220604_121100.jpg": "",
		"IMG_20240115_093000.heic": "",
		"IMG_20220604_140000.mp4":  "",
		"no_date_file.jpg":         "",
	}
	for name, content := range files {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
		require.NoError(t, err)
	}

	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	ifc := &ImportFolderCmd{
		Recursive:      true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		activeMonthSet: make(map[string]struct{}),
	}

	months, err := ifc.PreScan(context.Background())
	require.NoError(t, err)

	// Should find months from the filename dates
	// IMG_20220604 -> 2022-06
	// IMG_20240115 -> 2024-01
	// no_date_file.jpg -> falls back to file mod time (which is now)
	assert.Contains(t, months, "2022-06")
	assert.Contains(t, months, "2024-01")
	// At least 2 date-bearing months, possibly more from mod times
	assert.GreaterOrEqual(t, len(months), 2)
}

func TestPreScanWithICloudMetas(t *testing.T) {
	// Create a temp directory with some files
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "no_date.jpg"), []byte(""), 0o644)
	require.NoError(t, err)

	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	// Pre-populate iCloud metas
	metas := gen.NewSyncMap[string, iCloudMeta]()
	metas.Store("photo1.heic", iCloudMeta{
		originalCreationDate: time.Date(2019, 3, 15, 10, 0, 0, 0, time.UTC),
	})
	metas.Store("photo2.heic", iCloudMeta{
		originalCreationDate: time.Date(2021, 11, 20, 14, 0, 0, 0, time.UTC),
	})

	ifc := &ImportFolderCmd{
		Recursive:      true,
		ICloudTakeout:  true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		icloudMetas:    metas,
		activeMonthSet: make(map[string]struct{}),
	}

	months, err := ifc.PreScan(context.Background())
	require.NoError(t, err)

	// Should contain months from iCloud metas
	assert.Contains(t, months, "2019-03")
	assert.Contains(t, months, "2021-11")
}

func TestPreScanNoDateTracking(t *testing.T) {
	// Create a temp directory with a file that has no date in its name
	dir := t.TempDir()

	// Create a file with no date pattern and a very old mod time
	filePath := filepath.Join(dir, "random.jpg")
	err := os.WriteFile(filePath, []byte("fake image"), 0o644)
	require.NoError(t, err)

	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	ifc := &ImportFolderCmd{
		Recursive:      true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		activeMonthSet: make(map[string]struct{}),
	}

	_, err = ifc.PreScan(context.Background())
	require.NoError(t, err)

	// The file has no date in its name, but it has a mod time (the time of creation during test).
	// So it should still get a month from the mod time fallback.
	// hasNoDateFiles should be false because mod time was available.
	assert.False(t, ifc.HasNoDateFiles())
	assert.GreaterOrEqual(t, len(ifc.ActiveMonths()), 1)
}

func TestPreScanRecursive(t *testing.T) {
	// Create nested directory structure
	dir := t.TempDir()
	subDir := filepath.Join(dir, "2022", "vacation")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, "IMG_20200101_120000.jpg"), []byte(""), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "IMG_20220815_180000.heic"), []byte(""), 0o644)
	require.NoError(t, err)

	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	ifc := &ImportFolderCmd{
		Recursive:      true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		activeMonthSet: make(map[string]struct{}),
	}

	months, err := ifc.PreScan(context.Background())
	require.NoError(t, err)

	assert.Contains(t, months, "2020-01")
	assert.Contains(t, months, "2022-08")
}

func TestSetNoDateFiles(t *testing.T) {
	ifc := &ImportFolderCmd{}

	assert.False(t, ifc.HasNoDateFiles())

	ifc.setNoDateFiles()
	assert.True(t, ifc.HasNoDateFiles())

	// Calling again is idempotent
	ifc.setNoDateFiles()
	assert.True(t, ifc.HasNoDateFiles())
}

func TestPreScanContextCancellation(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "IMG_20220604_121100.jpg"), []byte(""), 0o644)
	require.NoError(t, err)

	tz := time.UTC
	sm := filetypes.DefaultSupportedMedia

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ifc := &ImportFolderCmd{
		Recursive:      true,
		tz:             tz,
		supportedMedia: sm,
		infoCollector:  filenames.NewInfoCollector(tz, sm),
		fsyss:          []fs.FS{os.DirFS(dir)},
		activeMonthSet: make(map[string]struct{}),
	}

	_, err = ifc.PreScan(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}
