# Batched Upload with Date-Range Scoped Server Queries

## Problem

When uploading an iCloud archive to an Immich server with a large database, the tool downloads metadata for **every** asset on the server before uploading anything. For a 500K-asset server, this means hundreds of API calls and hundreds of MB of RAM just for the preparation phase — even if the upload is only a few thousand files.

## Solution

Process uploads **month by month**, scoping server queries to each month's date range. Track progress in a persistent state file so uploads can be interrupted and resumed at any time.

## Architecture

### Pipeline Change

Current flow (everything at once):

```
1. Fetch ALL server assets → build full index in memory
2. Fetch ALL albums → link to index
3. Browse ALL local files → group
4. Upload loop: check each file against full index
```

New flow (month by month):

```
1. Pre-scan: read iCloud CSVs, extract file dates, determine month list
2. Load state file, skip completed months
3. For each remaining month:
   a. Fetch server assets for that month only (±1 day padding)
   b. Fetch albums scoped to that month's assets
   c. Browse local files filtered to that month
   d. Upload loop: check against month-scoped index
   e. Record each upload in state file
   f. Mark month complete, check --batch-limit
4. Print summary
```

### Pre-scan Phase

For iCloud takeout, the CSV first pass already exists (`icloud.go`). During this pass, `UseICloudPhotoDetails()` parses `Photo Details.csv` which contains `originalCreationDate` for every file. The pre-scan extends this to also track the set of active months.

For files not in the CSV (videos, files without CSV entries):
- Use filename-derived dates from `InfoCollector` (e.g., `IMG_20220604_*`)
- Fall back to file modification date

For files with no determinable date, they are collected into a special `"no-date"` batch processed last.

For non-iCloud sources (`from-folder`), the pre-scan walks directories and extracts dates from filenames only. If no dates are determinable, it falls back to current behavior (fetch all server assets, no batching).

### Date-Range Scoped Server Queries

Instead of `GetAllAssets()` which uses `SearchOptions().All()`, use `GetFilteredAssetsFn()` with a date range:

```go
so := immich.SearchOptions().All().WithDateRange(monthDateRange)
err = uc.client.Immich.GetFilteredAssetsFn(ctx, so, filter)
```

The `SearchMetadataQuery` already has `TakenBefore` and `TakenAfter` fields. The infrastructure exists — it is just not used during the upload index fetch today.

Each month's query uses ±1 day padding to handle timezone differences between iCloud export and Immich storage.

### Memory Efficiency

The asset index is rebuilt per month and discarded after. Memory usage changes from O(total server assets) to O(max assets in a single month):

| Scenario | Current | Per-month |
|---|---|---|
| 500K server assets | ~500K entries in 3 sync maps (hundreds of MB) | ~5K entries (a few MB) |
| Local file tracking | Entire archive in memory | One month's files |
| Album cache | Accumulates across full run | Flushes per month |

The tool can handle arbitrarily large archives and server databases without memory being a bottleneck.

## State File

### Location

`~/.config/immich-go/state/<hash>/state.json`

Where `<hash>` is derived from the archive path + server URL, so multiple archives or servers don't collide.

### Structure

```json
{
  "version": 1,
  "archive_path": "/path/to/icloud-export",
  "server_url": "https://immich.example.com",
  "created_at": "2026-03-06T10:00:00Z",
  "updated_at": "2026-03-06T12:34:56Z",
  "date_range": {
    "min": "2015-06-01",
    "max": "2024-11-30"
  },
  "completed_months": [
    "2015-06",
    "2015-07",
    "2015-08"
  ],
  "in_progress": {
    "month": "2015-09",
    "uploaded_files": [
      {
        "source": "icloud-export-part1.zip",
        "path": "Photos/2015/09/IMG_1234.HEIC",
        "size": 3456789
      },
      {
        "source": "icloud-export-part2.zip",
        "path": "Photos/2015/09/IMG_5678.JPG",
        "size": 1234567
      }
    ]
  }
}
```

### Completed months

Skipped entirely on resume — no local scanning, no server queries.

### In-progress month

On resume, files are matched by source + path + size (available from `fs.Stat()`, no file content reading needed). Matched files are skipped without computing checksums.

### Persistence

Written atomically: write to temp file, then rename. Prevents corruption if killed mid-write.

## CLI Flags

New flags on `upload from-icloud`:

```
--batch-limit N      Process at most N months then stop (0 = unlimited, default: 0)
--state-dir PATH     Override state directory (default: ~/.config/immich-go/state/)
--reset-state        Clear saved state and start fresh
--show-state         Print current state (completed/remaining months) and exit
```

The existing `--date-range` flag works as a manual filter on top of batching — restricts which months are eligible.

## User Experience

```bash
# First run — processes all months, can interrupt anytime with Ctrl+C
immich-go upload from-icloud --server=https://immich.local --api-key=xxx /path/to/archive.zip

# Interrupted? Just re-run the same command — resumes automatically
immich-go upload from-icloud --server=https://immich.local --api-key=xxx /path/to/archive.zip

# Conservative: 3 months at a time
immich-go upload from-icloud --batch-limit 3 --server=... /path/to/archive.zip

# Check progress without processing
immich-go upload from-icloud --show-state --server=... /path/to/archive.zip
# Output: "42/120 months completed. Next: 2019-01. Last run: 2026-03-06 12:34"

# Only process 2022
immich-go upload from-icloud --date-range 2022 --server=... /path/to/archive.zip
```

End-of-run summary:

```
Batch complete: processed 2015-06 through 2015-08 (3 months)
  Uploaded: 1,234 assets (4.2 GB)
  Skipped:    567 (already on server)
  Errors:       3
Progress: 3/120 months complete. Next: 2015-09
```

## Error Handling

### Archive changes between runs

New files in completed months are not reprocessed. The tool prints a warning: "Archive has N new files in already-completed months. Use --reset-state to reprocess."

### Server-side deletions between runs

State tracks "we uploaded this" not "it exists on server." Files uploaded in previous runs but later deleted from Immich are not re-uploaded. Use `--reset-state` to force re-upload.

### Multiple archives to same server

Each archive path produces a different state file hash. No conflicts.

### Zip files that span months

A single zip can contain photos from many months. During pre-scan, each file is mapped to its month. During `BrowseMonth`, the zip is re-opened but only files for the target month are yielded. Zip central directory reads are cheap, so this is acceptable.

### Files with no date

Collected into a `"no-date"` batch, processed last. Server query falls back to checksum-based lookup.

## Implementation Notes

### Adapter Interface Changes

New optional interface for adapters that support date-aware browsing:

```go
type DateRangeProvider interface {
    PreScan(ctx context.Context) (months []string, err error)
    BrowseMonth(ctx context.Context, month string) chan *assets.Group
}
```

If an adapter does not implement `DateRangeProvider`, the upload command falls back to current behavior (full index, no batching).

### Key Files to Modify

| File | Change |
|---|---|
| `adapters/folder/icloud.go` | Track min/max dates during CSV pass |
| `adapters/folder/commands.go` | Implement `DateRangeProvider` for iCloud |
| `adapters/folder/run.go` | Add `PreScan()` and `BrowseMonth()` methods |
| `adapters/adapters.go` | Define `DateRangeProvider` interface |
| `app/upload/upload.go` | Add new CLI flags, state file loading |
| `app/upload/run.go` | Month-by-month loop, scoped index/queries |
| `app/upload/noui.go` | Update runner for batched flow |
| `app/upload/ui.go` | Update runner for batched flow, show month progress |
| `app/upload/state.go` | New file: state file read/write/matching |
| `app/upload/advice.go` | No changes needed (works with scoped index as-is) |
| `immich/metadata.go` | No changes needed (date range filtering already supported) |
